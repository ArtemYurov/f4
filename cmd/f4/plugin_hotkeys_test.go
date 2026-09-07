package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/unxed/f4/vfs"
	"github.com/unxed/vtinput"
	"github.com/unxed/vtui"
)

func TestPluginMenuItemTextPlacesShortcutInLeftColumn(t *testing.T) {
	if got := pluginMenuItemText("SQLite client", "Q", 3); got != "  &Q SQLite client" {
		t.Fatalf("plugin menu item = %q, want left-aligned shortcut", got)
	}
	if got := pluginMenuItemText("Русский плагин", "Ф", 3); got != "  &Ф Русский плагин" {
		t.Fatalf("Cyrillic plugin menu item = %q, want left-aligned shortcut", got)
	}
	if got := pluginMenuItemText("No shortcut", "", 3); got != "    No shortcut" {
		t.Fatalf("plugin menu item without shortcut = %q, want aligned label", got)
	}
}

func TestEventToHotkeyStringSupportsUnicodeLetters(t *testing.T) {
	for _, tc := range []struct {
		name string
		key  *vtinput.InputEvent
		want string
	}{
		{
			name: "text-only terminal input",
			key:  &vtinput.InputEvent{Type: vtinput.KeyEventType, KeyDown: true, Char: 'ф'},
			want: "Ф",
		},
		{
			name: "Windows key with translated character",
			key:  &vtinput.InputEvent{Type: vtinput.KeyEventType, KeyDown: true, VirtualKeyCode: vtinput.VK_Q, Char: 'й'},
			want: "Й",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := EventToHotkeyString(tc.key); got != tc.want {
				t.Fatalf("EventToHotkeyString = %q, want %q", got, tc.want)
			}
		})
	}
	parsed := ParseFarKey("Ф")
	if got := EventToHotkeyString(parsed); got != "Ф" {
		t.Fatalf("Unicode shortcut round trip = %q, want Ф", got)
	}
}

func TestPluginMenuItemShortcutActivatesWithItsCharacter(t *testing.T) {
	restoreManager := swapFrameManager(t)
	defer restoreManager()
	vtui.FrameManager.Init(vtui.NewSilentScreenBuf())

	for _, tc := range []struct {
		name string
		text string
		key  rune
	}{
		{name: "Latin", text: pluginMenuItemText("SQLite client", "Q", 1), key: 'q'},
		{name: "Cyrillic", text: pluginMenuItemText("Русский клиент", "Ф", 1), key: 'ф'},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			menu := vtui.NewVMenu("Plugins")
			menu.AddItem(vtui.MenuItem{
				Text: tc.text,
				OnClick: func() {
					called = true
				},
			})
			vtui.FrameManager.Push(menu)
			if !menu.ProcessKey(&vtinput.InputEvent{Type: vtinput.KeyEventType, KeyDown: true, Char: tc.key}) {
				t.Fatal("plugin menu did not consume its character hotkey")
			}
			if !called {
				t.Fatal("plugin menu character hotkey did not activate the item")
			}
			menu.Close()
			vtui.FrameManager.Pop()
		})
	}
}

func TestPluginMenuKeyLabelsAdvertiseF4(t *testing.T) {
	labels := pluginMenuKeyLabels(nil)
	if labels == nil || labels.Normal[3] != "F4" {
		t.Fatalf("plugin menu F4 keybar label = %#v, want F4", labels)
	}
}

func TestPanelsFrameDoesNotConsumePluginMenuShortcut(t *testing.T) {
	previousHotkeys := GlobalHotkeysMgr
	previousMacro := MacroMgr
	GlobalHotkeysMgr = &HotkeyManager{
		Bindings: map[string]map[string]string{
			"Shell": {"Q": "Plugin.Command.test.menu-only"},
		},
		Defaults: map[string]map[string]string{},
	}
	MacroMgr = &MacroManager{Macros: make(map[string]map[string][]*vtinput.InputEvent)}
	t.Cleanup(func() {
		GlobalHotkeysMgr = previousHotkeys
		MacroMgr = previousMacro
	})
	called := 0
	registration, err := (&coreAPI{}).RegisterPluginCommand(vfs.PluginCommand{
		ID:       "test.menu-only",
		Location: vfs.PluginCommandPanel,
		Label:    "Menu-only plugin command",
		Run:      func(vfs.App) { called++ },
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(registration.Unregister)
	restoreManager := swapFrameManager(t)
	defer restoreManager()
	vtui.FrameManager.Init(vtui.NewSilentScreenBuf())

	pf := &PanelsFrame{showPanels: true}
	defer setFrameManagerScreensForTest(t, []*vtui.AppScreen{{Frames: []vtui.Frame{pf}}}, 0)()
	e := &vtinput.InputEvent{
		Type:           vtinput.KeyEventType,
		KeyDown:        true,
		VirtualKeyCode: vtinput.VK_Q,
		Char:           'q',
	}
	if MacroMgr.Filter(e) {
		t.Fatal("plugin menu shortcut must remain available to the panel command line")
	}
	if MacroMgr.LookupHotkey(e) {
		t.Fatal("injected plugin menu shortcut must remain available to the panel command line")
	}
	if called != 0 {
		t.Fatalf("plugin menu shortcut ran a command %d times", called)
	}
	if pf.InterceptPluginKey(e) {
		t.Fatal("plugin menu shortcut must remain available to the panel command line")
	}
}

func TestDeletePluginHotkeyRemovesBinding(t *testing.T) {
	hm := &HotkeyManager{
		Bindings: map[string]map[string]string{
			"Shell": {"Q": "Plugin.Command.test.delete"},
		},
		Defaults: map[string]map[string]string{},
		iniPath:  filepath.Join(t.TempDir(), "hotkeys.ini"),
	}
	if !deletePluginHotkey(hm, "Shell", "Q") {
		t.Fatal("deletePluginHotkey returned false")
	}
	if _, ok := hm.Bindings["Shell"]["Q"]; ok {
		t.Fatal("plugin hotkey binding was not removed")
	}
	if _, err := os.Stat(hm.iniPath); err != nil {
		t.Fatalf("deletePluginHotkey did not persist the change: %v", err)
	}
}

func TestPluginCommandHotkeyUsesConfiguredShortcutAndRunsCommand(t *testing.T) {
	previousHotkeys := GlobalHotkeysMgr
	GlobalHotkeysMgr = &HotkeyManager{
		Bindings: map[string]map[string]string{
			"Shell": {"CtrlF9": "Plugin.Command.test.hotkey"},
		},
		Defaults: map[string]map[string]string{},
	}
	t.Cleanup(func() { GlobalHotkeysMgr = previousHotkeys })

	called := 0
	registration, err := (&coreAPI{}).RegisterPluginCommand(vfs.PluginCommand{
		ID:       "test.hotkey",
		Location: vfs.PluginCommandPanel,
		Label:    "Plugin hotkey command",
		Shortcut: "F1",
		Run: func(vfs.App) {
			called++
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(registration.Unregister)

	if got := pluginCommandShortcut(vfs.PluginCommand{ID: "test.hotkey", Shortcut: "F1"}); got != "Ctrl+F9" {
		t.Fatalf("configured plugin shortcut = %q, want Ctrl+F9", got)
	}
	action, ok := GetAction("Plugin.Command.test.hotkey")
	if !ok || action.Label != "Plugin hotkey command" || action.Area != "Shell" {
		t.Fatalf("plugin hotkey action = %#v, registered=%v", action, ok)
	}

	restoreManager := swapFrameManager(t)
	defer restoreManager()
	vtui.FrameManager.Init(vtui.NewSilentScreenBuf())
	pf := &PanelsFrame{}
	defer setFrameManagerScreensForTest(t, []*vtui.AppScreen{{Frames: []vtui.Frame{pf}}}, 0)()
	if !RunAction(action.Name) {
		t.Fatal("configured plugin action was not dispatched")
	}
	if called != 1 {
		t.Fatalf("plugin command calls = %d, want 1", called)
	}
}

func TestBuildHotkeyRowsIncludesUnassignedPluginCommand(t *testing.T) {
	previousHotkeys := GlobalHotkeysMgr
	GlobalHotkeysMgr = &HotkeyManager{
		Bindings: map[string]map[string]string{},
		Defaults: map[string]map[string]string{},
	}
	t.Cleanup(func() { GlobalHotkeysMgr = previousHotkeys })

	registration, err := (&coreAPI{}).RegisterPluginCommand(vfs.PluginCommand{
		ID:       "test.hotkey-row",
		Location: vfs.PluginCommandPanel,
		Label:    "Plugin row command",
		Shortcut: "Shift+F6",
		Run:      func(vfs.App) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(registration.Unregister)

	for _, row := range buildHotkeyRows(GlobalHotkeysMgr) {
		if row.Action == "Plugin.Command.test.hotkey-row" {
			if row.Label != "Plugin row command" || row.Key != "Shift+F6" || !row.Editable {
				t.Fatalf("plugin hotkey row = %#v", row)
			}
			return
		}
	}
	t.Fatal("unassigned plugin command is missing from the hotkey dialog")
}
