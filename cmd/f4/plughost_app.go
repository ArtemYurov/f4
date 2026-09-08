package main

import (
	"context"

	"github.com/unxed/f4/internal/plughost"
	"github.com/unxed/f4/internal/terminal"
	"github.com/unxed/f4/vfs"
	"github.com/unxed/vtui"
)

// hostApplication is the application side of plughost.Application: what the
// plugin host is allowed to ask of f4.
//
// It is deliberately not vfs.HostAPI, which coreAPI implements. That one is
// the plugin-facing perimeter and any plugin from the catalogue reaches every
// method on it; SetupUI and SetClipboard must never become reachable that way.
type hostApplication struct{}

func (hostApplication) Current() vfs.App {
	pf := findPanelsFrame()
	if pf == nil {
		// A typed nil inside the interface would pass the caller's nil check.
		return nil
	}
	return pf
}

func (hostApplication) OpenPanelProvider(app vfs.App, providerID string) {
	openRegisteredPanelProvider(app, providerID)
}

func (hostApplication) IsStale(app vfs.App) bool {
	panels, ok := app.(*PanelsFrame)
	if !ok {
		// Not a frame this application manages — a plugin's own app object or
		// a test double. There is nothing here to have gone stale.
		return false
	}
	return panels == nil || panels.closed ||
		vtui.FrameManager != nil && findPanelsFrameAnyScreen() != panels
}

func (hostApplication) SetupUI() { SetupUI() }

func (hostApplication) SetClipboard(text string) { terminal.SetF4Clipboard(text) }

func (hostApplication) RunSemanticAction(action map[string]any) bool {
	return HandleSemanticAction(action)
}

func (hostApplication) AskOverwrite(ctx context.Context, destPath string, src, dst vfs.VFSItem, anchor vtui.Frame) (int, bool) {
	return AskOverwrite(ctx, destPath, src, dst, anchor)
}

func (hostApplication) AskError(ctx context.Context, op string, err error, anchor vtui.Frame) int {
	return AskError(ctx, op, err, anchor)
}

var _ plughost.Application = hostApplication{}

func init() { plughost.App = hostApplication{} }
