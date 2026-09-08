package main

import (
	"sync"

	"github.com/unxed/f4/vfs"
)

// The drive registry is what the drive menu, the command palette and every
// plugin that adds a file system agree on. It needs nothing but a slice and a
// lock, which is why it can sit below everything that reads it.

type DriveEntry struct {
	Name    string
	Factory func() vfs.VFS
}

var (
	driveRegistryMu sync.RWMutex
	DriveRegistry   []DriveEntry
)

// RegisterDrive adds a drive, or replaces the factory of one already
// registered under that name. Replacing in place is deliberate: a plugin
// reloaded at runtime must not appear twice in the menu.
func RegisterDrive(name string, factory func() vfs.VFS) {
	driveRegistryMu.Lock()
	defer driveRegistryMu.Unlock()
	for i, d := range DriveRegistry {
		if d.Name == name {
			DriveRegistry[i].Factory = factory
			return
		}
	}
	DriveRegistry = append(DriveRegistry, DriveEntry{Name: name, Factory: factory})
}

func driveRegistrySnapshot() []DriveEntry {
	driveRegistryMu.RLock()
	defer driveRegistryMu.RUnlock()
	return append([]DriveEntry(nil), DriveRegistry...)
}
