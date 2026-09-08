package main

import (
	"path/filepath"
	"strings"

	"github.com/unxed/f4/internal/fileops"
	"github.com/unxed/f4/internal/i18n"
	"github.com/unxed/f4/internal/macro"
	"github.com/unxed/f4/internal/numeric"
	"github.com/unxed/f4/sdk/extui"
	"github.com/unxed/vtinput"
	"github.com/unxed/vtui"
)

// SemanticNode экспортирует PanelsFrame в семантическое дерево ShellModel
func (pf *PanelsFrame) SemanticNode(ctx *vtui.SemanticContext) map[string]any {
	shell := extui.ShellModel{
		ID:             vtui.SemanticID(pf),
		Title:          strings.TrimSpace(pf.GetTitle()),
		Mode:           "panels",
		ActivePanel:    pf.activeIdx,
		ShowPanels:     pf.showPanels,
		ShowKeyBar:     pf.showKeyBar,
		TerminalBusy:   pf.isPtyBusy(),
		TerminalActive: !pf.showPanels,
	}
	if !pf.showPanels {
		shell.Mode = "terminal"
	}

	for i, panel := range pf.panels {
		if fsp, ok := panel.(*FileSystemPanel); ok {
			shell.Panels = append(shell.Panels, fsp.semanticPanelModel(ctx, i, i == pf.activeIdx))
		}
	}

	if pf.cmdLine != nil {
		shell.CommandLine = pf.cmdLine.SemanticModel(ctx)
	}
	if pf.termView != nil {
		shell.Terminal = pf.termView.SemanticModel(ctx)
	}
	if macro.MacroMgr != nil && macro.MacroMgr.Recording {
		shell.MacroRecording = true
	}

	return shell.ToMap()
}

// HandleSemanticAction глобально маршрутизирует семантические действия из внешнего GUI
func HandleSemanticAction(action map[string]any) bool {
	if action == nil {
		return false
	}
	actionName := semanticString(action["action"])
	target := semanticString(action["target"])
	if strings.HasPrefix(actionName, "workspace.") || actionName == "tab.activate" || strings.HasPrefix(target, "workspace-") {
		return vtui.FrameManager.HandleSemanticAction(action)
	}
	if kind, _ := action["kind"].(string); kind == "command" {
		return vtui.FrameManager.EmitCommand(semanticInt(action["command"]), action["args"])
	}
	if semanticString(action["action"]) == "menu_bar_activate" || semanticString(action["action"]) == "menuBar.activate" {
		if mb := vtui.FrameManager.GetActiveMenuBar(); mb != nil {
			idx := semanticInt(action["index"])
			if idx >= 0 && idx < len(mb.Items) {
				mb.Active = true
				mb.ActivateSubMenu(idx)
				vtui.FrameManager.Redraw()
				return true
			}
		}
	}

	activeIdx := vtui.FrameManager.ActiveIdx
	frames := vtui.FrameManager.GetActiveFrames(activeIdx)
	for i := len(frames) - 1; i >= 0; i-- {
		if h, ok := frames[i].(vtui.SemanticActionHandler); ok && h.HandleSemanticAction(action) {
			vtui.FrameManager.Redraw()
			return true
		}
	}

	target = semanticString(action["target"])
	if target == "" {
		return false
	}
	for i := len(frames) - 1; i >= 0; i-- {
		if handleSemanticFrameAction(frames[i], target, action) {
			vtui.FrameManager.Redraw()
			return true
		}
	}
	return false
}

func handleSemanticFrameAction(frame vtui.Frame, target string, action map[string]any) bool {
	if vtui.SemanticID(frame) == target {
		switch semanticString(action["action"]) {
		case "close", "dialog.close", "window.close":
			frame.Close()
			return true
		case "menu_activate", "menu.activate":
			if menu, ok := frame.(*vtui.VMenu); ok {
				idx := semanticInt(action["index"])
				if idx >= 0 && idx < len(menu.Items) && !menu.Items[idx].Separator {
					menu.SetSelectPos(idx)
					return menu.ProcessKey(&vtinput.InputEvent{Type: vtinput.KeyEventType, KeyDown: true, VirtualKeyCode: vtinput.VK_RETURN, InputSource: "qt_semantic"})
				}
			}
		}
	}
	if c, ok := frame.(vtui.Container); ok {
		return handleSemanticChildrenAction(c.GetChildren(), target, action)
	}
	return false
}

func handleSemanticChildrenAction(children []vtui.UIElement, target string, action map[string]any) bool {
	for _, child := range children {
		if vtui.SemanticID(child) == target {
			return handleSemanticElementAction(child, action)
		}
		if c, ok := child.(vtui.Container); ok {
			if handleSemanticChildrenAction(c.GetChildren(), target, action) {
				return true
			}
		}
	}
	return false
}

func handleSemanticElementAction(el vtui.UIElement, action map[string]any) bool {
	switch semanticString(action["action"]) {
	case "focus", "control.focus":
		el.SetFocus(true)
		return true
	case "activate", "control.activate":
		return el.ProcessKey(&vtinput.InputEvent{Type: vtinput.KeyEventType, KeyDown: true, VirtualKeyCode: vtinput.VK_RETURN, InputSource: "qt_semantic"})
	case "toggle", "control.toggle":
		return el.ProcessKey(&vtinput.InputEvent{Type: vtinput.KeyEventType, KeyDown: true, VirtualKeyCode: vtinput.VK_SPACE, Char: ' ', InputSource: "qt_semantic"})
	case "set_text", "control.setText":
		if edit, ok := el.(*vtui.Edit); ok {
			edit.SetText(semanticString(action["text"]))
			if edit.OnTextChange != nil {
				edit.OnTextChange(edit.GetText())
			}
			return true
		}
	case "insert_text", "control.insertText":
		if edit, ok := el.(*vtui.Edit); ok {
			edit.InsertString(semanticString(action["text"]))
			return true
		}
	case "select", "control.select":
		idx := semanticInt(action["index"])
		switch w := el.(type) {
		case *vtui.RadioGroup:
			if idx >= 0 && idx < len(w.Items) {
				w.SetData(idx)
				return true
			}
		case *vtui.ListBox:
			if idx >= 0 && idx < len(w.Items) {
				w.SetSelectPos(idx)
				return true
			}
		case *vtui.ComboBox:
			if idx >= 0 && idx < len(w.Menu.Items) {
				w.Menu.SetSelectPos(idx)
				w.Edit.SetText(w.Menu.Items[idx].Text)
				return true
			}
		}
	}
	return false
}

func (pf *PanelsFrame) HandleSemanticAction(action map[string]any) bool {
	switch semanticString(action["action"]) {
	case "activate_panel", "panel.activate":
		side := semanticInt(action["side"])
		if side >= 0 && side < len(pf.panels) {
			pf.activeIdx = side
			pf.lastKey = 0
			return true
		}
	case "panel_cursor", "panel.cursor":
		if fsp := pf.panelForSemanticAction(action); fsp != nil {
			fsp.SetCursorIndex(semanticInt(action["index"]))
			return true
		}
	case "panel_open", "panel.open":
		if fsp := pf.panelForSemanticAction(action); fsp != nil {
			idx := semanticInt(action["index"])
			pf.setActivePanelForAction(action)
			fsp.SetCursorIndex(idx)
			return pf.ProcessKey(&vtinput.InputEvent{Type: vtinput.KeyEventType, KeyDown: true, VirtualKeyCode: vtinput.VK_RETURN, InputSource: "qt_semantic"})
		}
	case "panel_toggle_selection", "panel.toggleSelection":
		if fsp := pf.panelForSemanticAction(action); fsp != nil {
			fsp.ToggleSelection(semanticInt(action["index"]))
			return true
		}
	case "panel_refresh", "panel.refresh":
		if fsp := pf.panelForSemanticAction(action); fsp != nil {
			fsp.ReadDirectory()
			return true
		}
	case "submit_command", "command.submit":
		if text := semanticString(action["text"]); text != "" && pf.cmdLine != nil {
			pf.cmdLine.Edit.SetText(text)
		}
		return pf.ProcessKey(&vtinput.InputEvent{Type: vtinput.KeyEventType, KeyDown: true, VirtualKeyCode: vtinput.VK_RETURN, InputSource: "qt_semantic"})
	case "set_command_text", "command.setText":
		if pf.cmdLine != nil {
			pf.cmdLine.Edit.SetText(semanticString(action["text"]))
			return true
		}
	case "emit_command", "command.emit":
		return vtui.FrameManager.EmitCommand(semanticInt(action["command"]), action["args"])
	}
	return false
}

func (pf *PanelsFrame) setActivePanelForAction(action map[string]any) {
	side := semanticInt(action["side"])
	if side >= 0 && side < len(pf.panels) {
		pf.activeIdx = side
	}
}

func (pf *PanelsFrame) panelForSemanticAction(action map[string]any) *FileSystemPanel {
	side := semanticInt(action["side"])
	if side < 0 || side >= len(pf.panels) {
		side = pf.activeIdx
	}
	if fsp, ok := pf.panels[side].(*FileSystemPanel); ok {
		return fsp
	}
	return nil
}

func (fp *FileSystemPanel) semanticPanelModel(ctx *vtui.SemanticContext, side int, active bool) extui.PanelModel {
	var entries []extui.FileEntryModel
	selectedCount := 0
	var selectedSize int64
	var totalSize int64
	for i, entry := range fp.entries {
		if !entry.IsDir {
			totalSize += entry.Size
		}
		if entry.Selected {
			selectedCount++
			selectedSize += entry.Size
		}
		entries = append(entries, extui.FileEntryModel{
			Index:          i,
			Name:           entry.Name,
			Size:           entry.Size,
			SizeText:       semanticFileSize(entry),
			IsDir:          entry.IsDir,
			IsUp:           entry.Name == "..",
			IsHidden:       entry.IsHidden,
			IsExecutable:   entry.IsExecutable,
			IsCached:       entry.IsCached,
			Selected:       entry.Selected,
			SizeCalculated: entry.SizeCalculated,
			MTime:          entry.MTime.Format("2006-01-02 15:04"),
			Mode:           entry.Mode,
		})
	}

	return extui.PanelModel{
		ID:            vtui.SemanticID(fp),
		Side:          side,
		Active:        active,
		Path:          fp.vfs.GetPath(),
		Title:         fp.frame.GetTitle(),
		ViewMode:      viewModeName(fp.effectiveViewMode()),
		SortMode:      sortModeName(fp.sortMode),
		SortReverse:   fp.sortReverse,
		Cursor:        fp.GetCursorIndex(),
		Top:           fp.table.TopPos,
		Loading:       fp.isLoading,
		FastFind:      fp.fastFindMode,
		FastFindText:  fp.fastFindStr,
		SelectedCount: selectedCount,
		SelectedSize:  selectedSize,
		TotalCount:    len(fp.entries),
		TotalSize:     totalSize,
		Entries:       entries,
	}
}

func semanticFileSize(entry *fileEntry) string {
	if entry.IsDir {
		if entry.SizeCalculated {
			return fileops.FormatIntWithSpaces(entry.Size)
		}
		if entry.Name == ".." {
			return i18n.Msg("Panel.UpDir")
		}
		return ""
	}
	return fileops.FormatIntWithSpaces(entry.Size)
}

func viewModeName(mode ViewMode) string {
	switch mode {
	case ViewModeBrief:
		return "brief"
	case ViewModeDetailed:
		return "detailed"
	case ViewModeWide:
		return "wide"
	default:
		return "medium"
	}
}

func sortModeName(mode SortMode) string {
	switch mode {
	case SortExt:
		return "extension"
	case SortTime:
		return "time"
	case SortSize:
		return "size"
	case SortUnsorted:
		return "unsorted"
	default:
		return "name"
	}
}

func semanticBaseName(v interface{ Base(string) string }, path string) string {
	if path == "" {
		return ""
	}
	if v != nil {
		return v.Base(path)
	}
	return filepath.Base(path)
}

func semanticString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func semanticInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int8:
		return int(n)
	case int16:
		return int(n)
	case int32:
		return int(n)
	case int64:
		value, _ := numeric.BoundedInt64ToInt(n)
		return value
	case uint:
		value, _ := numeric.BoundedUint64ToInt(uint64(n))
		return value
	case uint8:
		return int(n)
	case uint16:
		return int(n)
	case uint32:
		value, _ := numeric.BoundedUint64ToInt(uint64(n))
		return value
	case uint64:
		value, _ := numeric.BoundedUint64ToInt(n)
		return value
	case float32:
		return int(n)
	case float64:
		return int(n)
	}
	return 0
}

func semanticBool(v any) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	if n, ok := v.(int); ok {
		return n != 0
	}
	if f, ok := v.(float64); ok {
		return f != 0
	}
	return false
}
