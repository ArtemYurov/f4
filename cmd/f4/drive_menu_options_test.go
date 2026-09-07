package main

import (
	"strings"
	"testing"

	"github.com/unxed/vtinput"
	"github.com/unxed/vtui"
)

func TestDriveMenuOptions_DefaultsAndFormatting(t *testing.T) {
	if parseDriveMenuOptions("") != defaultDriveMenuOptions {
		t.Fatalf("empty options did not use defaults: %#x", parseDriveMenuOptions(""))
	}
	if parseDriveMenuOptions("not-a-number") != defaultDriveMenuOptions {
		t.Fatalf("invalid options did not use defaults")
	}
	if got := driveMenuPlatformItemText(DriveEntry{Name: "/ Root"}, driveMenuShowType|driveMenuShowFilesystem); !strings.Contains(got, "/") {
		t.Fatalf("root row lost its name: %q", got)
	}
	if got := driveMenuSize(1024*1024*3, false); got != "3 MiB" {
		t.Fatalf("integer drive size = %q, want 3 MiB", got)
	}
	if got := driveMenuSize(1024*1024*3, true); !strings.Contains(got, "3.0") {
		t.Fatalf("decimal drive size = %q, want a decimal value", got)
	}
}

func TestPanelsFrame_DriveMenu_F9OpensOptions(t *testing.T) {
	vtui.FrameManager.Init(vtui.NewSilentScreenBuf())
	pf := NewPanelsFrame()
	defer pf.Close()
	pf.ResizeConsole(80, 25)

	oldOptions := AppConfig.DriveMenuOptions
	AppConfig.DriveMenuOptions = defaultDriveMenuOptions
	t.Cleanup(func() { AppConfig.DriveMenuOptions = oldOptions })

	pf.showDriveMenu(0)
	menu, ok := driveMenuFromFrame(vtui.FrameManager.GetTopFrame())
	if !ok {
		t.Fatalf("drive menu not opened: %T", vtui.FrameManager.GetTopFrame())
	}
	if !strings.Contains(Msg("Drive.BottomHint"), "F9") {
		t.Fatalf("drive menu hint does not advertise F9: %q", Msg("Drive.BottomHint"))
	}
	if !menu.ProcessKey(&vtinput.InputEvent{
		Type: vtinput.KeyEventType, KeyDown: true, VirtualKeyCode: vtinput.VK_F9,
	}) {
		t.Fatal("F9 was not consumed by the drive menu")
	}
	if vtui.FrameManager.GetTopFrame() == menu {
		t.Fatal("F9 did not open the drive options dialog")
	}
	dlg, ok := vtui.FrameManager.GetTopFrame().(vtui.Container)
	if !ok {
		t.Fatalf("drive options frame is not a container: %T", vtui.FrameManager.GetTopFrame())
	}
	vtui.AssertLayout(t, dlg)
	vtui.FrameManager.Pop()
	menu.Close()
}
