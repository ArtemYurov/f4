package app

import (
	"github.com/unxed/f4/internal/appcmd"
	"github.com/unxed/f4/internal/panel"
)

// handlePanelsAppCommand answers the frame commands the panels raise and do not
// serve themselves: every one of them opens a dialog or runs an action, which
// is the application's half of the frame. The order of the cases is the order
// PanelsFrame.HandleCommand had them in, because a command can reach a frame
// that is not the active one and the switch is what decides which panels
// answer.
func handlePanelsAppCommand(pf *panel.PanelsFrame, cmd int, args any) bool {
	switch cmd {
	case appcmd.CmNew:
		actionNewFile(pf)
		return true
	case appcmd.CmView:
		actionViewFile(pf)
		return true
	case appcmd.CmEdit:
		actionEditFile(pf)
		return true
	case appcmd.CmCopy, appcmd.CmMove:
		actionCopyMove(pf, cmd == appcmd.CmMove)
		return true
	case appcmd.CmRename:
		actionRename(pf)
		return true
	case appcmd.CmMkDir:
		actionMkDir(pf)
		return true
	case appcmd.CmDelete:
		actionDelete(pf)
		return true
	case appcmd.CmFindFile:
		actionFindFile(pf)
		return true
	case appcmd.CmPanelSettings:
		actionPanelSettings(pf)
		return true
	case appcmd.CmEditorSettings:
		actionEditorSettings(pf)
		return true
	case appcmd.CmColorerSettings:
		actionColorerSettings(pf)
		return true
	case appcmd.CmAppearanceSettings:
		actionAppearanceSettings(pf)
		return true
	case appcmd.CmConfirmationsSettings:
		actionConfirmationsSettings(pf)
		return true
	case appcmd.CmHotkeyConfig:
		actionHotkeyConfig(pf)
		return true
	case appcmd.CmLanguage:
		actionLanguage(pf)
		return true
	case appcmd.CmHelpLanguage:
		actionHelpLanguage(pf)
		return true
	case appcmd.CmUpdateSettings:
		actionUpdateSettings(pf)
		return true
	case appcmd.CmPlugins:
		actionManagePlugins(pf)
		return true
	case appcmd.CmPlugRing:
		actionPlugRing(pf)
		return true
	case appcmd.CmBackground:
		return actionBackground()
	case appcmd.CmWorkspaceNew:
		return actionWorkspaceNew()
	}
	return false
}
