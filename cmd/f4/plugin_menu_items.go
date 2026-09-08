package main

import "github.com/unxed/f4/vfs"

// PluginMenuItem is a row a plugin adds to the plugin menu through
// vfs.HostAPI.RegisterPluginMenuItem.
//
// The registry shares pluginRegistryMu with GlobalHotkeys: both are what
// plugins registered, and a plugin's Init can touch either while the menu is
// being built.
type PluginMenuItem struct {
	ActionName string
	Label      string
	Handler    func(app vfs.App)
}

var PluginMenuItems []PluginMenuItem

func RegisterPluginMenuItem(label string, handler func(app vfs.App)) {
	pluginRegistryMu.Lock()
	PluginMenuItems = append(PluginMenuItems, PluginMenuItem{
		ActionName: legacyPluginActionName(len(PluginMenuItems)),
		Label:      label,
		Handler:    handler,
	})
	pluginRegistryMu.Unlock()
}

func pluginMenuItemsSnapshot() []PluginMenuItem {
	pluginRegistryMu.RLock()
	defer pluginRegistryMu.RUnlock()
	return append([]PluginMenuItem(nil), PluginMenuItems...)
}
