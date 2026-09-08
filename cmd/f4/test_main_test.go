package main

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/unxed/f4/internal/fusefs"
	"github.com/unxed/f4/internal/testutil"
	"github.com/unxed/f4/vfs"
	"github.com/unxed/vtinput"
	"github.com/unxed/vtui"
)

// pressKey is testutil.PressKey with this package's macro filter, which is
// where action hotkeys are dispatched. The managers are created on demand
// because most tests never touch them.
func pressKey(f vtui.Frame, e *vtinput.InputEvent) bool {
	if GlobalHotkeysMgr == nil {
		GlobalHotkeysMgr = NewHotkeyManager("")
	}
	if MacroMgr == nil {
		MacroMgr = NewMacroManager("")
	}
	return testutil.PressKey(f, e, MacroMgr.Filter)
}

// preserveActionRegistry keeps tests that register synthetic actions from
// leaking them into later tests or the next -count iteration.
func preserveActionRegistry(t *testing.T) {
	t.Helper()
	oldRegistry := actionRegistry
	oldOrder := actionOrder
	actionRegistry = make(map[string]Action, len(oldRegistry))
	for key, action := range oldRegistry {
		actionRegistry[key] = action
	}
	actionOrder = append([]string(nil), oldOrder...)
	t.Cleanup(func() {
		actionRegistry = oldRegistry
		actionOrder = oldOrder
	})
}

func TestMain(m *testing.M) {
	os.Exit(testutil.Main(m, installTestSeams, unmountTestFilesystems))
}

// installTestSeams points this package's escape hatches somewhere harmless for
// the duration of the run.
func installTestSeams() {
	vfs.InitSudoClient("/usr/bin/f4", "")

	// Unit tests must never hand control to the user's desktop. Individual
	// tests that exercise these routes install per-dialog/per-frame recorders.
	defaultExternalUICommandRunner = func(string, []string, string) error { return nil }
	defaultNativePropertiesOpener = func(string) error { return nil }

	// Frames must not fork the user's shell during unit tests; the few
	// tests that exercise the PTY path construct one explicitly.
	spawnLocalShellPTY = false

	// Toast behavior is still exercised through vtui's real asynchronous
	// setup and expiry paths, but unit tests do not need production-length
	// display times. Keep a small observable window: tests may observe another
	// effect of the same UI task (for example, clipboard contents) before they
	// pump the nested toast task, and a 1 ms toast can expire in that gap.
	toastDurationOverride = func(time.Duration) time.Duration {
		const minimumObservableToastDuration = 100 * time.Millisecond
		return minimumObservableToastDuration
	}
	queueShowToast = func(string, time.Duration) {}

	// os.UserConfigDir ignores XDG_CONFIG_HOME and APPDATA on darwin, so the
	// seam is what actually isolates the suite from the developer's profile.
	if dir := testutil.ConfigDir(); dir != "" {
		userConfigDir = func() (string, error) { return dir, nil }
		resetConfigDirForTest()
	}
}

func unmountTestFilesystems() error {
	return errors.Join(fusefs.UnmountAll()...)
}
