package dialog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/unxed/f4/internal/config"
	"github.com/unxed/f4/internal/testutil"
	"github.com/unxed/f4/internal/theme"
	"github.com/unxed/vtinput"
	"github.com/unxed/vtui"
)

func portableSettingsMouseCoordinate(value int) int16 {
	return int16(value) // #nosec G115 -- the test dialog is inside the test screen.
}

// The portable settings dialog is dragged by its resize corner: the width
// follows the pointer while the height stays where the layout put it (#274).
func TestPortableSettingsDialogResizesHorizontallyOnly(t *testing.T) {
	t.Cleanup(testutil.SwapFrameManager(t))
	screen := vtui.NewSilentScreenBuf()
	screen.AllocBuf(120, 30)
	vtui.FrameManager.Init(screen)
	theme.SetDefaultF4Palette()

	tmpDir := t.TempDir()
	exe := filepath.Join(tmpDir, "f4")
	if err := os.WriteFile(exe, nil, 0600); err != nil {
		t.Fatal(err)
	}
	oldExecutable := config.Executable
	oldUserConfigDir := config.UserConfigDir
	config.Executable = func() (string, error) { return exe, nil }
	config.UserConfigDir = func() (string, error) { return tmpDir, nil }
	config.ResetConfigDirForTest()
	t.Cleanup(func() {
		config.Executable = oldExecutable
		config.UserConfigDir = oldUserConfigDir
		config.ResetConfigDirForTest()
	})

	ShowPortableSettings()
	top := vtui.FrameManager.GetTopFrame()
	if top == nil {
		t.Fatal("portable settings did not open a dialog")
	}
	dlg, ok := top.(*portableSettingsDialog)
	if !ok {
		t.Fatalf("portable settings frame has type %T, want *portableSettingsDialog", top)
	}

	startX2, startY2 := dlg.X2, dlg.Y2
	if !dlg.ProcessMouse(&vtinput.InputEvent{
		Type:        vtinput.MouseEventType,
		KeyDown:     true,
		ButtonState: vtinput.FromLeft1stButtonPressed,
		MouseX:      portableSettingsMouseCoordinate(startX2),
		MouseY:      portableSettingsMouseCoordinate(startY2),
	}) {
		t.Fatal("portable settings resize corner was not handled")
	}
	dlg.ProcessMouse(&vtinput.InputEvent{
		Type:        vtinput.MouseEventType,
		ButtonState: vtinput.FromLeft1stButtonPressed,
		MouseX:      portableSettingsMouseCoordinate(startX2 + 8),
		MouseY:      portableSettingsMouseCoordinate(startY2 + 4),
	})
	dlg.ProcessMouse(&vtinput.InputEvent{Type: vtinput.MouseEventType})
	if dlg.X2 != startX2+8 {
		t.Errorf("portable settings right edge = %d, want %d", dlg.X2, startX2+8)
	}
	if dlg.Y2 != startY2 {
		t.Errorf("portable settings bottom edge = %d, want fixed %d", dlg.Y2, startY2)
	}
}
