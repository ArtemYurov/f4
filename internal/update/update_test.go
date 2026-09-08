package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/unxed/f4/vfs"
	"github.com/unxed/sevenzip"
)

type memoryWriteSeeker struct {
	data []byte
	off  int64
}

func TestUpdater_ParseUpdateHelperArgs(t *testing.T) {
	archive, kind, found, err := ParseHelperArgs([]string{HelperFlag, `C:\Users\Test User\f4-update.archive`, "zip"})
	if err != nil || !found {
		t.Fatalf("ParseHelperArgs() failed: found=%v err=%v", found, err)
	}
	if archive != `C:\Users\Test User\f4-update.archive` || kind != "zip" {
		t.Fatalf("ParseHelperArgs() = %q, %q; want archive path and zip", archive, kind)
	}

	if _, _, found, err := ParseHelperArgs([]string{HelperFlag, "archive.zip"}); !found || err == nil {
		t.Fatalf("malformed helper invocation: found=%v err=%v", found, err)
	}
	if _, _, found, err := ParseHelperArgs([]string{"--gui=win32"}); found || err != nil {
		t.Fatalf("normal invocation parsed as update helper: found=%v err=%v", found, err)
	}
}

func TestUpdater_ManualBuildVersionUsesBuildTimestamp(t *testing.T) {
	manual := Build{Version: "manual-build-sha", TimeText: "2026-08-21T12:00:00Z"}

	newerLocalBuild := Release{TagName: "v0.2.0-beta", PublishedAt: "2026-08-20T12:00:00Z"}
	if stableReleaseNeedsUpdate(newerLocalBuild, manual, "") {
		t.Fatal("a manual build newer than the release must not request a downgrade")
	}

	newerRelease := Release{TagName: "v0.2.0-beta", PublishedAt: "2026-08-22T12:00:00Z"}
	if !stableReleaseNeedsUpdate(newerRelease, manual, "") {
		t.Fatal("a release newer than a manual build must be offered")
	}

	exact := Build{Version: "v0.2.0-beta", IsRelease: true, TimeText: manual.TimeText}
	if stableReleaseNeedsUpdate(newerRelease, exact, "") {
		t.Fatal("an exact release build must not request an update to itself")
	}

	// LastVersion is the "already installed" marker and suppresses the offer
	// on its own, whatever the build calls itself.
	if stableReleaseNeedsUpdate(newerRelease, manual, "v0.2.0-beta") {
		t.Fatal("an already installed release must not be offered again")
	}
}

func (w *memoryWriteSeeker) Write(p []byte) (int, error) {
	end := w.off + int64(len(p))
	if w.off < 0 || end < w.off {
		return 0, errors.New("invalid memory write offset")
	}
	if end > int64(len(w.data)) {
		w.data = append(w.data, make([]byte, end-int64(len(w.data)))...)
	}
	copy(w.data[w.off:end], p)
	w.off = end
	return len(p), nil
}

func (w *memoryWriteSeeker) Seek(offset int64, whence int) (int64, error) {
	var base int64
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		base = w.off
	case io.SeekEnd:
		base = int64(len(w.data))
	default:
		return 0, errors.New("invalid seek origin")
	}
	next := base + offset
	if next < 0 {
		return 0, errors.New("negative seek offset")
	}
	w.off = next
	return next, nil
}

func TestUpdater_Extractors(t *testing.T) {
	binaryContent := []byte("fake_executable_data")
	pluginContent := []byte("plugin_data")
	// Path Traversal items
	badAbsPath := "/etc/passwd"
	badRelPath := "../../windows/system32/cmd.exe"

	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	f1, err := zw.Create("f4.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f1.Write(binaryContent); err != nil {
		t.Fatal(err)
	}
	f2, err := zw.Create("plugins/dummy.dll")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f2.Write(pluginContent); err != nil {
		t.Fatal(err)
	}
	fBad1, err := zw.Create(badAbsPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fBad1.Write([]byte("hacked")); err != nil {
		t.Fatal(err)
	}
	fBad2, err := zw.Create(badRelPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fBad2.Write([]byte("hacked")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	destZip := t.TempDir()
	err = ExtractZip(zipBuf.Bytes(), destZip)
	if err != nil {
		t.Fatalf("ExtractZip failed: %v", err)
	}
	b1, _ := os.ReadFile(filepath.Join(destZip, "f4.exe"))
	b2, _ := os.ReadFile(filepath.Join(destZip, "plugins", "dummy.dll"))
	if string(b1) != "fake_executable_data" || string(b2) != "plugin_data" {
		t.Errorf("Zip extraction mismatch")
	}
	if _, err := os.Stat(filepath.Join(destZip, "etc", "passwd")); !os.IsNotExist(err) {
		t.Error("Zip Slip vulnerability detected (absolute path extracted)!")
	}

	var tgzBuf bytes.Buffer
	gw := gzip.NewWriter(&tgzBuf)
	tw := tar.NewWriter(gw)
	if err := tw.WriteHeader(&tar.Header{Name: "f4", Size: int64(len(binaryContent)), Mode: 0755}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(binaryContent); err != nil {
		t.Fatal(err)
	}
	if err := tw.WriteHeader(&tar.Header{Name: "plugins/dummy.so", Size: int64(len(pluginContent)), Mode: 0755}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(pluginContent); err != nil {
		t.Fatal(err)
	}
	if err := tw.WriteHeader(&tar.Header{Name: badAbsPath, Size: 6, Mode: 0644}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("hacked")); err != nil {
		t.Fatal(err)
	}
	if err := tw.WriteHeader(&tar.Header{Name: badRelPath, Size: 6, Mode: 0644}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("hacked")); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}

	destTar := t.TempDir()
	err = ExtractTarGz(tgzBuf.Bytes(), destTar)
	if err != nil {
		t.Fatalf("ExtractTarGz failed: %v", err)
	}
	b1, _ = os.ReadFile(filepath.Join(destTar, "f4"))
	b2, _ = os.ReadFile(filepath.Join(destTar, "plugins", "dummy.so"))
	if string(b1) != "fake_executable_data" || string(b2) != "plugin_data" {
		t.Errorf("TarGz extraction mismatch")
	}

	var sevenBuf memoryWriteSeeker
	sw, err := sevenzip.NewWriter(&sevenBuf)
	if err != nil {
		t.Fatal(err)
	}
	sf1, err := sw.Create("f4.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sf1.Write(binaryContent); err != nil {
		t.Fatal(err)
	}
	if err := sf1.Close(); err != nil {
		t.Fatal(err)
	}
	sf2, err := sw.Create("plugins/dummy.dll")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sf2.Write(pluginContent); err != nil {
		t.Fatal(err)
	}
	if err := sf2.Close(); err != nil {
		t.Fatal(err)
	}
	sfBad, err := sw.Create(badAbsPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sfBad.Write([]byte("hacked")); err != nil {
		t.Fatal(err)
	}
	if err := sfBad.Close(); err != nil {
		t.Fatal(err)
	}
	if err := sw.Close(); err != nil {
		t.Fatal(err)
	}
	sevenData := append([]byte(nil), sevenBuf.data...)
	dest7z := t.TempDir()
	err = extract7z(sevenData, dest7z)
	if err != nil {
		t.Fatalf("extract7z failed: %v", err)
	}
	b1, _ = os.ReadFile(filepath.Join(dest7z, "f4.exe"))
	b2, _ = os.ReadFile(filepath.Join(dest7z, "plugins", "dummy.dll"))
	if string(b1) != "fake_executable_data" || string(b2) != "plugin_data" {
		t.Errorf("7z extraction mismatch")
	}
	if _, err := os.Stat(filepath.Join(dest7z, "etc", "passwd")); !os.IsNotExist(err) {
		t.Error("7z Zip Slip vulnerability detected (absolute path extracted)!")
	}
	runtime.KeepAlive(sw)
}

func TestSanitizeExtractPathRejectsPlatformSeparators(t *testing.T) {
	for _, name := range []string{`..\\outside`, `folder\\..\\outside`, "nul\x00name"} {
		if _, err := SanitizePath(name, t.TempDir()); err == nil {
			t.Errorf("SanitizePath(%q) accepted an unsafe archive name", name)
		}
	}
}

func TestUpdater_WriteFileSafe_FallbackOldName(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "binary.exe")
	oldPath := targetPath + ".old"

	if err := os.WriteFile(targetPath, []byte("v1"), 0755); err != nil { // #nosec G306 -- the updater fixture represents an executable binary.
		t.Fatal(err)
	}

	// Make os.Remove(oldPath) fail by creating a non-empty directory
	if err := os.Mkdir(oldPath, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldPath, "lock"), []byte("lock"), 0600); err != nil {
		t.Fatal(err)
	}

	err := writeFileSafe(targetPath, strings.NewReader("v2"), 0755)
	if err != nil {
		t.Fatalf("writeFileSafe failed with fallback: %v", err)
	}

	b, _ := os.ReadFile(targetPath)
	if string(b) != "v2" {
		t.Errorf("Expected 'v2', got %q", string(b))
	}

	b, _ = os.ReadFile(oldPath + ".1")
	if string(b) != "v1" {
		t.Errorf("Expected old file to be renamed to .old.1, got %q", string(b))
	}
}

func TestUpdater_WriteFileSafe(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "binary.exe")

	// 1. Initial write
	err := writeFileSafe(targetPath, strings.NewReader("v1"), 0755)
	if err != nil {
		t.Fatalf("writeFileSafe failed: %v", err)
	}

	b, _ := os.ReadFile(targetPath)
	if string(b) != "v1" {
		t.Errorf("Expected 'v1', got %q", string(b))
	}

	// 2. Overwrite existing file
	err = writeFileSafe(targetPath, strings.NewReader("v2"), 0755)
	if err != nil {
		t.Fatalf("writeFileSafe overwrite failed: %v", err)
	}

	b, _ = os.ReadFile(targetPath)
	if string(b) != "v2" {
		t.Errorf("Expected 'v2', got %q", string(b))
	}
}

func TestUpdater_WriteFileSafe_SudoElevationFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix-specific sudo elevation test on Windows")
	}

	tmpDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("failed to resolve temp dir: %v", err)
	}

	// Создаем директорию с ограниченными правами доступа (только чтение и выполнение)
	protectedDir := filepath.Join(tmpDir, "protected_dir")
	if err := os.Mkdir(protectedDir, 0555); err != nil { // #nosec G301 -- the read-only directory is the behavior under test.
		t.Fatalf("failed to create read-only dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(protectedDir, 0755); err != nil { // #nosec G302 -- cleanup must restore access to the deliberately locked directory.
			t.Errorf("restore protected directory permissions: %v", err)
		}
	})

	targetPath := filepath.Join(protectedDir, "binary.exe")

	// Проверяем, что система действительно запрещает запись под обычным пользователем
	_, errDirect := os.Create(targetPath)
	if errDirect == nil {
		t.Skip("System is running as root; skipping elevation test")
	}

	// Инициализируем глобальный SudoClient
	vfs.InitSudoClient("/nonexistent/f4", "")

	// Пытаемся записать файл. Операция должна пойти по пути эскалации и упасть
	// на попытке соединения с сокетом диспетчера (так как парольный диалог мы гасим),
	// что доказывает успешный переход управления в SudoClient!
	err = writeFileSafe(targetPath, strings.NewReader("v2"), 0755)
	if err == nil {
		t.Error("expected writeFileSafe to fail under restricted directory")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "elevated dispatcher") && !strings.Contains(errStr, "sudo process") {
		t.Errorf("expected error to originate from sudo elevation fallback, got: %q", errStr)
	}
}

func TestFormatBuildTimeUsesOneClockForVCSAndNightlyMetadata(t *testing.T) {
	fromVCS := FormatBuildTime("2026-08-23T06:49:17Z")
	fromReleaseBody := FormatBuildTime("2026-08-23 06:49:17")
	if fromVCS != fromReleaseBody {
		t.Fatalf("VCS time %q and release-body time %q diverged", fromVCS, fromReleaseBody)
	}
	if got := FormatBuildTime("not a timestamp"); got != "not a timestamp" {
		t.Fatalf("invalid timestamp = %q, want unchanged input", got)
	}
}
