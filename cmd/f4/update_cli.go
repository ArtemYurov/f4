package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// updateChannelName называет канал так, как он пишется в командной строке.
func updateChannelName(channel int) string {
	if channel == updateChannelNightly {
		return "nightly"
	}
	return "stable"
}

// parseUpdateChannelArg переводит аргумент `--update` в номер канала.
// Пустой аргумент означает канал из настроек.
func parseUpdateChannelArg(arg string, configured int) (channel int, explicit bool, err error) {
	switch strings.ToLower(strings.TrimSpace(arg)) {
	case "":
		return configured, false, nil
	case "stable", "latest":
		return updateChannelStable, true, nil
	case "nightly":
		return updateChannelNightly, true, nil
	}
	return 0, false, fmt.Errorf("unknown update channel %q (expected \"stable\" or \"nightly\")", arg)
}

// runUpdateCLI обслуживает `f4 --update [stable|nightly]`: тот же механизм,
// что и диалог обновления, но без интерфейса — скачивает и ставит сразу.
// Возвращает код завершения процесса.
//
// Весь вывод идёт в stdout: к этому моменту vtui.SetupStderrLog уже увёл
// stderr в файл лога, и написанное туда пользователь не увидит.
func runUpdateCLI(channelArg string) int {
	channel, explicit, err := parseUpdateChannelArg(channelArg, AppConfig.UpdateChannel)
	if err != nil {
		fmt.Printf("f4: %v\n", err)
		return 2
	}

	if explicit && AppConfig.UpdateChannel != channel {
		// Названный каналом становится каналом автопроверок сразу, до
		// опроса GitHub: выход по «уже актуально» или сетевой сбой иначе
		// оставит проверки на прежнем канале, и ближайшая же позовёт
		// обратно на него.
		AppConfig.UpdateChannel = channel
		SaveConfig()
	}

	// Ночной архив на медленном канале качается минутами, поэтому здесь
	// таймаут щедрее десяти секунд, отведённых на опрос API.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	fmt.Printf("Checking the %s channel...\n", updateChannelName(channel))
	cand, err := fetchUpdateCandidate(ctx, channel)
	if err != nil {
		fmt.Printf("f4: update check failed: %v\n", err)
		return 1
	}
	if !cand.needsUpdate {
		fmt.Printf("Already up to date: %s\n", cand.displayVersion)
		return 0
	}

	if _, err := updateTargetDir(); err != nil {
		fmt.Printf("f4: %v\n", err)
		return 1
	}

	fmt.Printf("Installing %s\n", cand.displayVersion)
	// Проценты перерисовываются возвратом каретки, поэтому в файл или в
	// журнал CI они не печатаются вовсе: там от них остаётся мусорная
	// строка вместо хода загрузки.
	showProgress := term.IsTerminal(int(os.Stdout.Fd()))
	lastPct := -1
	data, err := downloadUpdateArchive(ctx, cand.downloadURL, func(percent int) {
		if !showProgress || percent == lastPct {
			return
		}
		lastPct = percent
		fmt.Printf("\rDownloading... %d%%", percent)
	})
	if showProgress {
		fmt.Println()
	}
	if err != nil {
		fmt.Printf("f4: download failed: %v\n", err)
		return 1
	}

	if err := installUpdateArchive(data, cand.archiveKind); err != nil {
		fmt.Printf("f4: install failed: %v\n", err)
		return 1
	}

	AppConfig.LastUpdateVersion = cand.updateKey
	AppConfig.LastUpdateCheck = time.Now().Unix()
	SaveConfig()

	fmt.Printf("Installed %s. Restart f4 to use it.\n", cand.displayVersion)
	return 0
}
