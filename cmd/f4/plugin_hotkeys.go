package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/mattn/go-runewidth"
	"github.com/unxed/f4/vfs"
	"github.com/unxed/vtui"
)

// Plugin menu entries are actions too, but their lifetime is controlled by a
// plugin registration rather than by the built-in action registry. Keeping a
// separate namespace lets hotkeys.ini refer to them without leaving stale
// Action values behind when an RPC plugin disconnects.
func pluginCommandActionName(id string) string { return "Plugin.Command." + id }

func legacyPluginActionName(index int) string {
	return "Plugin.Legacy." + strconv.Itoa(index)
}

func isPluginActionName(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	return strings.HasPrefix(name, "plugin.command.") || strings.HasPrefix(name, "plugin.legacy.")
}

func pluginActionForName(name string) (Action, bool) {
	rawName := strings.TrimSpace(name)
	lowerName := strings.ToLower(rawName)
	switch {
	case strings.HasPrefix(lowerName, "plugin.command."):
		id := strings.TrimSpace(rawName[len("Plugin.Command."):])
		pluginCommandRegistry.RLock()
		registered, ok := pluginCommandRegistry.byID[strings.ToLower(id)]
		if ok {
			registered.command = clonePluginCommand(registered.command)
		}
		pluginCommandRegistry.RUnlock()
		if !ok {
			return Action{}, false
		}
		command := registered.command
		actionName := pluginCommandActionName(command.ID)
		return Action{
			Name:        actionName,
			Area:        "Shell",
			Label:       pluginCommandDisplayLabel(command),
			Description: pluginCommandDisplayDescription(command),
			Handler:     func() bool { return runPluginHotkeyAction(actionName) },
		}, true
	case strings.HasPrefix(lowerName, "plugin.legacy."):
		index, err := strconv.Atoi(strings.TrimSpace(rawName[len("Plugin.Legacy."):]))
		if err != nil || index < 0 {
			return Action{}, false
		}
		items := pluginMenuItemsSnapshot()
		if index >= len(items) {
			return Action{}, false
		}
		item := items[index]
		actionName := item.ActionName
		if actionName == "" {
			actionName = legacyPluginActionName(index)
		}
		return Action{
			Name:        actionName,
			Area:        "Shell",
			Label:       item.Label,
			Description: "Run the selected plugin command",
			Handler:     func() bool { return runPluginHotkeyAction(actionName) },
		}, true
	default:
		return Action{}, false
	}
}

func runPluginHotkeyAction(name string) bool {
	name = strings.TrimSpace(name)
	if strings.HasPrefix(strings.ToLower(name), "plugin.command.") {
		id := strings.TrimSpace(name[len("Plugin.Command."):])
		pluginCommandRegistry.RLock()
		registered, ok := pluginCommandRegistry.byID[strings.ToLower(id)]
		if ok {
			registered.command = clonePluginCommand(registered.command)
		}
		pluginCommandRegistry.RUnlock()
		if !ok {
			return false
		}
		pf := findPanelsFrame()
		if pf == nil {
			return false
		}
		return executeRegisteredPluginCommand(registered.command.Location, registered.command.ID, pf)
	}

	if strings.HasPrefix(strings.ToLower(name), "plugin.legacy.") {
		index, err := strconv.Atoi(strings.TrimSpace(name[len("Plugin.Legacy."):]))
		if err != nil || index < 0 {
			return false
		}
		items := pluginMenuItemsSnapshot()
		if index >= len(items) || items[index].Handler == nil {
			return false
		}
		if pf := findPanelsFrame(); pf != nil {
			items[index].Handler(pf)
			return true
		}
	}
	return false
}

func pluginActionShortcut(name string) string {
	if GlobalHotkeysMgr == nil {
		return ""
	}
	if key := GlobalHotkeysMgr.GetKeyForAction("Shell", name); key != "" {
		return FormatKeyForUI(key)
	}
	return ""
}

func pluginCommandShortcut(command vfs.PluginCommand) string {
	if shortcut := pluginActionShortcut(pluginCommandActionName(command.ID)); shortcut != "" {
		return shortcut
	}
	return command.Shortcut
}

// pluginMenuItemShortcut returns the shortcut which should be shown in the
// left-hand shortcut column of the F11 menu. Legacy menu items can carry an
// ampersand marker in their label instead of a persisted hotkey binding.
func pluginMenuItemShortcut(label, shortcut string) string {
	if shortcut = strings.TrimSpace(shortcut); shortcut != "" {
		return shortcut
	}
	_, hotkey, _ := vtui.ParseAmpersandString(label)
	if hotkey == 0 {
		return ""
	}
	return string(unicode.ToUpper(hotkey))
}

// pluginMenuItemText renders the shortcut in a stable column before the
// command name. A single unmodified character remains an ampersand hotkey so
// it can also activate the item while the F11 menu is open. Longer chords are
// display-only metadata; their actual dispatch happens in PanelsFrame.
func pluginMenuItemText(label, shortcut string, shortcutWidth int) string {
	cleanLabel, _, _ := vtui.ParseAmpersandString(label)
	shortcut = pluginMenuItemShortcut(label, shortcut)
	if shortcutWidth < runewidth.StringWidth(shortcut) {
		shortcutWidth = runewidth.StringWidth(shortcut)
	}
	prefix := strings.Repeat(" ", shortcutWidth-runewidth.StringWidth(shortcut))
	if shortcut != "" {
		if len([]rune(shortcut)) == 1 && !unicode.IsSpace([]rune(shortcut)[0]) {
			prefix += "&" + shortcut
		} else {
			prefix += shortcut
		}
	}
	return prefix + " " + cleanLabel
}

func configuredHotkeyBinding(hm *HotkeyManager, actionName string) (string, string) {
	if hm == nil {
		return "", ""
	}
	for _, area := range []string{"Shell", "Common"} {
		var keys []string
		for key, binding := range hm.Bindings[area] {
			namePart := strings.SplitN(binding, ":", 2)[0]
			if strings.EqualFold(namePart, actionName) {
				keys = append(keys, key)
			}
		}
		sort.Strings(keys)
		if len(keys) != 0 {
			return area, keys[0]
		}
	}
	return "", ""
}

func pluginActionConfiguredBinding(name string) (string, string) {
	return configuredHotkeyBinding(GlobalHotkeysMgr, name)
}

func pluginActionConfiguredKey(name string) string {
	_, key := pluginActionConfiguredBinding(name)
	return key
}

func pluginActionDefaultShortcut(name string) string {
	lowerName := strings.ToLower(strings.TrimSpace(name))
	if !strings.HasPrefix(lowerName, "plugin.command.") {
		return ""
	}
	id := strings.TrimSpace(name[len("Plugin.Command."):])
	pluginCommandRegistry.RLock()
	registered, ok := pluginCommandRegistry.byID[strings.ToLower(id)]
	if ok {
		shortcut := registered.command.Shortcut
		pluginCommandRegistry.RUnlock()
		return shortcut
	}
	pluginCommandRegistry.RUnlock()
	return ""
}

func assignPluginHotkey(menu *vtui.VMenu, index int, actionName string) {
	hm := GlobalHotkeysMgr
	if hm == nil || vtui.FrameManager == nil || !isPluginActionName(actionName) {
		return
	}
	oldArea, oldKey := configuredHotkeyBinding(hm, actionName)
	frame := NewHotkeyAssignFrame(hm, actionName, "Shell", func() {
		newArea, newKey := configuredHotkeyBinding(hm, actionName)
		if oldArea != "" && (oldArea != newArea || oldKey != newKey) {
			hm.Unbind(oldArea, oldKey)
		}
		hm.Save()
		if menu != nil && index >= 0 && index < len(menu.Items) {
			menu.Items[index].Shortcut = pluginActionShortcut(actionName)
		}
		vtui.FrameManager.Redraw()
	})
	vtui.FrameManager.Push(frame)
}

func deletePluginHotkey(hm *HotkeyManager, area, key string) bool {
	if hm == nil || area == "" || key == "" {
		return false
	}
	hm.Unbind(area, key)
	hm.Save()
	return true
}

func pluginHotkeyDeleteQuestion(key string) string {
	return fmt.Sprintf("Remove plugin hotkey %s?", FormatKeyForUI(key))
}

func pluginMenuKeyLabels(pf *PanelsFrame) *vtui.KeySet {
	if pf != nil && MacroMgr != nil {
		if base := pf.GetKeyLabels(); base != nil {
			labels := *base
			labels.Normal[3] = "F4"
			return &labels
		}
	}
	return &vtui.KeySet{Normal: vtui.KeyBarLabels{"", "", "", "F4"}}
}

// pluginHotkeyActionsSnapshot includes commands that are currently hidden from
// the F11 menu as well. A user can therefore assign a shortcut once and keep
// it when moving to another drive or when a plugin changes its visibility.
func pluginHotkeyActionsSnapshot() []Action {
	pluginCommandRegistry.RLock()
	commandIDs := append([]string(nil), pluginCommandRegistry.order...)
	pluginCommandRegistry.RUnlock()

	actions := make([]Action, 0, len(commandIDs)+len(pluginMenuItemsSnapshot()))
	for _, id := range commandIDs {
		pluginCommandRegistry.RLock()
		registered, ok := pluginCommandRegistry.byID[id]
		pluginCommandRegistry.RUnlock()
		if !ok {
			continue
		}
		if action, ok := pluginActionForName(pluginCommandActionName(registered.command.ID)); ok {
			actions = append(actions, action)
		}
	}
	for index, item := range pluginMenuItemsSnapshot() {
		name := item.ActionName
		if name == "" {
			name = legacyPluginActionName(index)
		}
		if action, ok := pluginActionForName(name); ok {
			actions = append(actions, action)
		}
	}
	return actions
}
