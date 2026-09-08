package main

import (
	"github.com/unxed/f4/internal/term"
	"github.com/unxed/f4/vfs"
)

// The environment manager itself is in internal/term, next to the pty backends
// that hand it to a child. What stays here is coreAPI's implementation of the
// plugin-facing contract, and the runtime and broadcast it needs from the
// panel side.

var _ vfs.ProcessEnvironmentHost = (*coreAPI)(nil)

func (c *coreAPI) SnapshotProcessEnvironment() vfs.ProcessEnvironmentSnapshot {
	// EnvMan calls Snapshot during plugin initialization, making this the
	// earliest reliable point to establish this process's isolated runtime.
	_ = initializeProcessEnvironmentRuntime()
	snapshot, _ := term.GlobalProcessEnvironment.Snapshot()
	return snapshot
}

func (c *coreAPI) ApplyProcessEnvironment(changes []vfs.ProcessEnvironmentChange) (vfs.ProcessEnvironmentSnapshot, error) {
	snapshot, generations, err := term.ApplyProcessEnvironmentWithRuntime(term.GlobalProcessEnvironment, initializeProcessEnvironmentRuntime, changes)
	broadcastProcessEnvironmentGenerations(generations)
	return snapshot, err
}
