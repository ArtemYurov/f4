package main

import (
	"github.com/unxed/f4/internal/term"
	"github.com/unxed/vtui"
)

// termApplication is the application side of term.Application. Most of it is
// what a Unix client attach needs: a daemon rebuilds the interface for the
// terminal that just connected, and the interface is not the terminal's to
// build.
type termApplication struct{}

func (termApplication) InitCore() *vtui.ScreenBuf { return InitCore() }

func (termApplication) OpenEditFile() { openDashEFileIfRequested() }

func (termApplication) ClientAttached(startLeft, startRight, editPath string) {
	top := vtui.FrameManager.GetTopFrame()
	pf, ok := top.(*PanelsFrame)
	if !ok || pf == nil {
		if editPath != "" {
			vtui.DebugLog("SERVER: -e %q: top frame is not a *PanelsFrame (%T)", editPath, top)
		}
		return
	}
	// A workspace that had its panels hidden gets its host console back.
	if pf.shellMode == term.ShellModeHost && !pf.showPanels {
		pf.enterHostConsole()
	}
	// A client that attached to a running daemon moves its workspace to its
	// own directory, as a normal start would.
	if startLeft != "" {
		applyStartupDirs(pf, startLeft, startRight)
	}
	if editPath != "" {
		openEditFileIn(pf, editPath)
	}
}

func (termApplication) ClientDetached() {
	for _, s := range vtui.FrameManager.Screens {
		if s == nil {
			continue
		}
		for _, f := range s.Frames {
			if pf, ok := f.(*PanelsFrame); ok && pf != nil {
				if pf.shellMode == term.ShellModeHost && pf.isHostConsoleActive() {
					pf.leaveHostConsole()
				}
			}
		}
	}
}

func (termApplication) DecodeImage(data []byte) (*vtui.ImageSurface, error) {
	return decodeImageWithStdlib(data)
}

func (termApplication) VersionInfo() string { return getFormattedVersionInfo() }

func (termApplication) EditFilePath() string { return editFilePath }

func (termApplication) StartupDirs() (string, string) { return startupDirs() }

var _ term.Application = termApplication{}

func init() { term.App = termApplication{} }
