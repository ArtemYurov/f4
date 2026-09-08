package main

import (
	"testing"
	"time"

	"github.com/unxed/f4/internal/terminal"
	"github.com/unxed/f4/internal/testutil"
)

// waitForDirectoryLoads blocks until no directory-load worker is running
// anywhere in the process.
//
// The workers read vtui.FrameManager and config.App while they run, so a test
// that replaces either one has to know they are all finished first. Panels are
// created deep inside PanelsFrame.ResizeConsole as well as directly, so the
// caller usually has no panel to wait on and this asks the question globally
// instead.
func waitForDirectoryLoads(t *testing.T) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		directoryLoadWorkers.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("timeout waiting for the directory-load workers to stop")
	}
}

// drainAsyncClipboard is terminal.WaitForAsyncClipboard in the shape a frame-manager
// drain takes. Clipboard writes run asynchronously because they may wait for
// far2l IPC, and SetClipboard reads vtui.FrameManager.
func drainAsyncClipboard(*testing.T) { terminal.WaitForAsyncClipboard() }

// swapFrameManager is testutil.SwapFrameManager carrying the two background
// workers this package leaves running. Both read the global frame manager, so
// both have to be joined before it is replaced — which is the whole reason the
// shared helper takes its drains from the caller.
func swapFrameManager(t *testing.T) func() {
	t.Helper()
	return testutil.SwapFrameManager(t, drainAsyncClipboard, waitForDirectoryLoads)
}
