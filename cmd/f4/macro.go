package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/unxed/f4/internal/config"
	"github.com/unxed/f4/internal/i18n"
	"github.com/unxed/f4/internal/ini"
	"github.com/unxed/f4/internal/keymap"
	"github.com/unxed/vtinput"
	"github.com/unxed/vtui"
)

var MacroMgr *MacroManager

// MacroManager handles recording, playback and storage of simple keyboard macros.
type MacroManager struct {
	Macros    map[string]map[string][]*vtinput.InputEvent
	Recording bool
	Assigning bool
	Buffer    []*vtinput.InputEvent
	iniPath   string
	StartArea string
	// Lua is the Far-compatible macro engine, present only when the user has
	// macros. Recorded macros keep working either way: this is a second
	// backend, not a replacement.
	Lua *LuaMacroEngine
}

func NewMacroManager(iniPath string) *MacroManager {
	mgr := &MacroManager{
		Macros:  make(map[string]map[string][]*vtinput.InputEvent),
		iniPath: iniPath,
	}
	mgr.Load()
	return mgr
}

func configuredHotkeyAction(hm *HotkeyManager, area, key string) string {
	if hm == nil {
		return ""
	}
	action := hm.GetAction(area, key)
	if !strings.HasPrefix(key, "RCtrl") {
		return action
	}

	plainKey := "Ctrl" + strings.TrimPrefix(key, "RCtrl")
	rctrlExplicit := hm.hasExplicitBinding(area, key)
	if rctrlExplicit && action != "" && !strings.EqualFold(action, "none") {
		return action
	}
	if hm.hasExplicitBinding(area, plainKey) {
		if plainAction := hm.GetAction(area, plainKey); plainAction != "" {
			return plainAction
		}
	}
	if rctrlExplicit {
		// The RCtrl spelling was explicitly unbound (or its condition is not
		// met): it no longer claims the key, so Right Ctrl acts as plain Ctrl.
		return hm.GetAction(area, plainKey)
	}
	if action != "" {
		return action
	}
	return hm.GetAction(area, plainKey)
}

// configurableHotkeyOwnsPanelBookmark lets an explicit configurable binding
// take the place of far2l's built-in Right Ctrl/Ctrl+Alt bookmark shortcuts.
// Unmodified defaults keep their historical bookmark behavior, while a user
// binding on either Ctrl spelling is honored without requiring both spellings.
func configurableHotkeyOwnsPanelBookmark(hm *HotkeyManager, area string, e *vtinput.InputEvent) bool {
	if hm == nil || e == nil || !isPanelBookmarkHotkey(e) {
		return false
	}
	key := keymap.EventToHotkeyString(e)
	if hm.hasExplicitBinding(area, key) {
		return true
	}
	if strings.HasPrefix(key, "RCtrl") {
		return hm.hasExplicitBinding(area, "Ctrl"+strings.TrimPrefix(key, "RCtrl"))
	}
	return false
}

func (m *MacroManager) GetCurrentArea() string {
	if vtui.FrameManager == nil {
		return "Common"
	}
	top := vtui.FrameManager.GetTopFrame()
	if top == nil {
		return "Common"
	}
	switch top.GetType() {
	case vtui.TypeDialog:
		return "Dialog"
	case vtui.TypeMenu:
		if menu, ok := top.(*vtui.VMenu); ok {
			if strings.Contains(menu.GetTitle(), "Drive") {
				return "Disks"
			}
		}
		return "Menu"
	case vtui.TypeUser + 1:
		if pf, ok := top.(*PanelsFrame); ok {
			if !pf.showPanels {
				return "Terminal"
			}
		}
		return "Shell"
	case vtui.TypeUser + 2:
		return "Editor"
	case vtui.TypeUser + 3:
		return "Viewer"
	}
	return "Other"
}

// isPanelBookmarkHotkey identifies far2l-compatible folder bookmark keys.
// Built-in bookmark combinations reach PanelsFrame before macro and
// configurable hotkey handling, because keymap.EventToFarString intentionally
// normalizes left and right Ctrl. Explicit configurable bindings are allowed
// to reclaim the combination before this handoff.
func isPanelBookmarkHotkey(e *vtinput.InputEvent) bool {
	if e.Type != vtinput.KeyEventType || !e.KeyDown {
		return false
	}

	rctrl := (e.ControlKeyState & vtinput.RightCtrlPressed) != 0
	lctrl := (e.ControlKeyState & vtinput.LeftCtrlPressed) != 0
	alt := (e.ControlKeyState & (vtinput.LeftAltPressed | vtinput.RightAltPressed)) != 0
	shift := (e.ControlKeyState & vtinput.ShiftPressed) != 0
	isGoto := (rctrl && !shift && !alt) || ((lctrl || rctrl) && alt && !shift)
	isSave := (rctrl && shift && !alt) || ((lctrl || rctrl) && alt && shift)

	if e.VirtualKeyCode >= vtinput.VK_0 && e.VirtualKeyCode <= vtinput.VK_9 {
		return isGoto || isSave
	}
	return e.VirtualKeyCode == vtinput.VK_OEM_3 && isGoto
}

// isPanelFastFindToggleKey identifies contextual panel-toggle keys owned by
// Fast Find. They must reach PanelsFrame before macros and configurable
// hotkeys, otherwise Esc/Del -> Panel.Toggle hides the panels first.
func isPanelFastFindToggleKey(e *vtinput.InputEvent) bool {
	if e.Type != vtinput.KeyEventType || !e.KeyDown ||
		(e.VirtualKeyCode != vtinput.VK_ESCAPE && e.VirtualKeyCode != vtinput.VK_DELETE) {
		return false
	}
	mods := e.ControlKeyState & (vtinput.LeftCtrlPressed | vtinput.RightCtrlPressed |
		vtinput.LeftAltPressed | vtinput.RightAltPressed | vtinput.ShiftPressed)
	if mods != 0 || vtui.FrameManager == nil {
		return false
	}
	pf, ok := vtui.FrameManager.GetTopFrame().(*PanelsFrame)
	if !ok || !pf.showPanels {
		return false
	}
	fsp := pf.getActivePanel()
	return fsp != nil && fsp.fastFindMode
}

func isPanelFastFindActive() bool {
	if vtui.FrameManager == nil {
		return false
	}
	pf, ok := vtui.FrameManager.GetTopFrame().(*PanelsFrame)
	if !ok || !pf.showPanels {
		return false
	}
	fsp := pf.getActivePanel()
	return fsp != nil && fsp.fastFindMode
}

func (m *MacroManager) Filter(e *vtinput.InputEvent) bool {
	if e.Type != vtinput.KeyEventType {
		return false
	}

	// The user's key remap (keymap.ini) substitutes the key before anything
	// else sees the event, so macros, plugin interception, configurable
	// hotkeys and the frames themselves all agree on which key was pressed.
	if !keymap.ApplyKeyRemap(m.GetCurrentArea(), e) {
		// The built-in Mac layout only sees what keymap.ini left behind, so
		// a rule the user wrote by hand still wins over it.
		keymap.ApplyMacKeys(m.GetCurrentArea(), e)
	}

	// Ctrl+. toggles recording. We check both VK and Char for better terminal compatibility.
	isCtrlDot := (e.VirtualKeyCode == vtinput.VK_OEM_PERIOD || e.Char == '.') &&
		(e.ControlKeyState&(vtinput.LeftCtrlPressed|vtinput.RightCtrlPressed)) != 0

	if isCtrlDot {
		if !e.KeyDown {
			return true // Consume KeyUp of the trigger
		}
		m.ToggleRecording()
		return true // Trigger is ALWAYS consumed
	}

	// The command palette is the application-wide escape hatch for finding
	// commands. Resolve it before recording/assignment so the opening chord is
	// neither captured in the macro buffer nor shadowed by macro playback.
	if e.KeyDown {
		currentArea := m.GetCurrentArea()
		hotkeyStr := keymap.EventToHotkeyString(e)
		if hm := GlobalHotkeysMgr; hm != nil {
			if actionName := configuredHotkeyAction(hm, currentArea, hotkeyStr); strings.EqualFold(actionName, commandPaletteActionName) {
				RunAction(actionName)
				return true
			}
		}
	}

	// Once the palette is open, its remaining query and navigation keys belong
	// to the dialog. In particular, do not record them into a macro that the
	// user may be stopping from the palette itself. The palette chord itself was
	// handled above, so pressing it again is still consumed without adding a
	// second dialog.
	if vtui.FrameManager != nil {
		if _, open := vtui.FrameManager.GetTopFrame().(*commandPaletteDialog); open {
			return false
		}
	}

	if m.Recording || m.Assigning {
		if e.KeyDown && m.Recording {
			switch e.VirtualKeyCode {
			case vtinput.VK_SHIFT, vtinput.VK_LSHIFT, vtinput.VK_RSHIFT,
				vtinput.VK_CONTROL, vtinput.VK_LCONTROL, vtinput.VK_RCONTROL,
				vtinput.VK_MENU, vtinput.VK_LMENU, vtinput.VK_RMENU,
				vtinput.VK_CAPITAL, vtinput.VK_NUMLOCK, vtinput.VK_SCROLL:
				// Ignore standalone modifier keys
			default:
				m.Buffer = append(m.Buffer, e)
			}
		}
		return false // Let it pass to the UI so user sees what they type / dialog catches it
	}

	if !e.KeyDown {
		return false
	}

	currentArea := m.GetCurrentArea()
	if currentArea == "Shell" && isPanelFastFindActive() {
		// zoin-bot: Fast Find is a transient panel input mode, so Shell macros
		// must not consume keys before PanelsFrame can update its query.
		return false
	}
	if currentArea == "Shell" &&
		((isPanelBookmarkHotkey(e) && !configurableHotkeyOwnsPanelBookmark(GlobalHotkeysMgr, currentArea, e)) ||
			isPanelFastFindToggleKey(e)) {
		return false
	}

	// Check if this key triggers a macro
	keyStr := keymap.EventToFarString(e)

	if areaMacros, ok := m.Macros[currentArea]; ok {
		if seq, ok := areaMacros[keyStr]; ok {
			vtui.DebugLog("MACRO: Playing back macro for %s in area %s", keyStr, currentArea)
			vtui.FrameManager.InjectEvents(seq)
			return true
		}
	}

	if commonMacros, ok := m.Macros["Common"]; ok {
		if seq, ok := commonMacros[keyStr]; ok {
			vtui.DebugLog("MACRO: Playing back macro for %s in area Common", keyStr)
			vtui.FrameManager.InjectEvents(seq)
			return true
		}
	}

	// Recorded macros win over scripted ones, as they do in Far.
	if m.Lua != nil && m.Lua.Trigger(currentArea, e) {
		vtui.DebugLog("MACRO: Running Lua macro for %s in area %s", keyStr, currentArea)
		return true
	}

	// Plugin key interception: global plugin hotkeys and the active
	// panel's PanelController may override built-in hotkeys, so they
	// are consulted before the hotkey manager.
	if vtui.FrameManager != nil {
		if top := vtui.FrameManager.GetTopFrame(); top != nil {
			if pi, ok := top.(interface {
				InterceptPluginKey(*vtinput.InputEvent) bool
			}); ok && pi.InterceptPluginKey(e) {
				return true
			}
		}
	}

	// A frame in a modal input state (e.g. editor autocomplete, panel
	// fast-find) may veto system hotkey dispatch: the key then falls
	// through to the frame's own ProcessKey, which handles such states
	// before anything else.
	if vtui.FrameManager != nil {
		if top := vtui.FrameManager.GetTopFrame(); top != nil {
			if v, ok := top.(interface {
				VetoActionKey(*vtinput.InputEvent) bool
			}); ok && v.VetoActionKey(e) {
				return false
			}
		}
	}

	// Hotkey Manager evaluation (System actions)
	if hm := GlobalHotkeysMgr; hm != nil {
		keyStr := keymap.EventToHotkeyString(e)
		if actionName := configuredHotkeyAction(hm, currentArea, keyStr); actionName != "" {
			// F4-assigned plugin shortcuts are menu accelerators, matching
			// far2l: the F11 plugin menu renders them as ampersand hotkeys.
			// They must not steal printable characters from the panel command
			// line while remaining persisted for that menu and its display.
			// A chord assigned in the hotkey dialog (Ctrl+F9 and friends) is a
			// real hotkey and keeps being dispatched here.
			if isPluginActionName(actionName) && isPluginMenuHotkey(keyStr) {
				return false
			}
			if strings.EqualFold(actionName, "none") {
				return true // Intercept and silence (explicitly unbound)
			}
			vtui.DebugLog("HOTKEY: Executing action %s for %s in area %s", actionName, keyStr, currentArea)
			if RunAction(actionName) {
				return true
			}
		}
	}

	return false
}

// ToggleRecording is shared by the native Ctrl+. handler and the command
// palette. Stopping remains deferred so the assignment dialog is pushed only
// after the current input dispatch (or palette close) has completed.
func (m *MacroManager) ToggleRecording() bool {
	if m == nil || vtui.FrameManager == nil {
		return false
	}
	if m.Recording {
		m.Recording = false
		vtui.FrameManager.PostTask(func() { m.showAssignDialog() })
	} else {
		m.Recording = true
		m.Buffer = make([]*vtinput.InputEvent, 0)
		m.StartArea = m.GetCurrentArea()
		vtui.DebugLog("MACRO: Started recording in area: %s", m.StartArea)
	}
	vtui.FrameManager.Redraw()
	return true
}

// LookupHotkey runs the configured action for e in the current area without
// touching macro-recording or macro-playback state. Frames use it as a
// fallback for synthesized key events that bypass FrameManager.EventFilter —
// notably mouse clicks on the F-key bar, which InjectEvents into the queue
// with is_injected=true and therefore never reach Filter above. Returns true
// when the key is bound (including to "None", which is a deliberate silence).
func (m *MacroManager) LookupHotkey(e *vtinput.InputEvent) bool {
	if m == nil || e == nil || e.Type != vtinput.KeyEventType || !e.KeyDown {
		return false
	}
	hm := GlobalHotkeysMgr
	if hm == nil {
		return false
	}
	keyStr := keymap.EventToHotkeyString(e)
	if keyStr == "" {
		return false
	}
	area := m.GetCurrentArea()
	actionName := configuredHotkeyAction(hm, area, keyStr)
	if actionName == "" {
		return false
	}
	// Plugin menu accelerators are activated by the F11 menu's ampersand
	// hotkeys. They are deliberately not global Shell hotkeys, so an injected
	// key (for example from a key-bar click) must not bypass that rule either.
	if isPluginActionName(actionName) && isPluginMenuHotkey(keyStr) {
		return false
	}
	if strings.EqualFold(actionName, "none") {
		return true
	}
	vtui.DebugLog("HOTKEY: Injected %s → action %s in area %s", keyStr, actionName, area)
	return RunAction(actionName)
}

func (m *MacroManager) showAssignDialog() {
	m.Assigning = true
	frame := NewMacroAssignFrame(m)
	vtui.FrameManager.Push(frame)
}

func (m *MacroManager) Load() {
	vtui.DebugLog("MACRO: Loading macros from %s", m.iniPath)
	newMacros := make(map[string]map[string][]*vtinput.InputEvent)
	ini := ini.Load(m.iniPath)
	for sectionName, sec := range ini.Sections() {
		if strings.HasPrefix(sectionName, "KeyMacros/") {
			parts := strings.SplitN(sectionName, "/", 3)
			if len(parts) == 3 {
				area := parts[1]
				hotkey := parts[2]
				seqStr := sec["Sequence"]
				if seqStr == "" {
					continue
				}

				var events []*vtinput.InputEvent
				for _, keyStr := range strings.Fields(seqStr) {
					if strings.HasPrefix(keyStr, "callplugin") || strings.HasPrefix(keyStr, "eval") || strings.Contains(keyStr, "(") {
						continue // Skip far2l macro functions for now
					}
					events = append(events, keymap.ParseFarKey(keyStr))
				}

				if newMacros[area] == nil {
					newMacros[area] = make(map[string][]*vtinput.InputEvent)
				}
				newMacros[area][hotkey] = events
			}
		} else {
			targetArea := sectionName
			if sectionName == "Macros" {
				targetArea = "Common" // Migration from legacy format
			}
			if newMacros[targetArea] == nil {
				newMacros[targetArea] = make(map[string][]*vtinput.InputEvent)
			}
			for key, val := range sec {
				if strings.Contains(val, ":") && !strings.Contains(val, " ") {
					var events []*vtinput.InputEvent
					for _, p := range strings.Split(val, ",") {
						fields := strings.Split(p, ":")
						if len(fields) == 3 {
							charValue, charErr := strconv.ParseInt(fields[0], 10, 32)
							vkValue, vkErr := strconv.ParseUint(fields[1], 10, 16)
							modsValue, modsErr := strconv.ParseUint(fields[2], 10, 32)
							if charErr != nil || vkErr != nil || modsErr != nil {
								continue
							}
							// #nosec G115 -- ParseInt with bitSize 32 bounds the conversion; ValidRune rejects negative and surrogate values.
							char := rune(charValue)
							if char != 0 && !utf8.ValidRune(char) {
								continue
							}
							// #nosec G115 -- ParseUint with bitSize 16 bounds the virtual key code.
							vk := uint16(vkValue)
							// #nosec G115 -- ParseUint with bitSize 32 bounds the control-state bit field.
							mods := vtinput.ControlKeyState(uint32(modsValue))
							events = append(events, &vtinput.InputEvent{
								Type:            vtinput.KeyEventType,
								KeyDown:         true,
								Char:            char,
								VirtualKeyCode:  vk,
								ControlKeyState: mods,
							})
						}
					}
					cleanKey := strings.ReplaceAll(key, "+", "")
					newMacros[targetArea][cleanKey] = events
				}
			}
		}
	}
	m.Macros = newMacros
}

func (m *MacroManager) Save() {
	vtui.DebugLog("MACRO: Saving macros to %s", m.iniPath)

	var sb strings.Builder
	for area, areaMacros := range m.Macros {
		for hotkey, seq := range areaMacros {
			if len(seq) == 0 {
				continue
			}
			fmt.Fprintf(&sb, "[KeyMacros/%s/%s]\n", area, hotkey)
			sb.WriteString("DisableOutput=0x1\n")

			var parts []string
			for _, e := range seq {
				parts = append(parts, keymap.EventToFarString(e))
			}
			fmt.Fprintf(&sb, "Sequence=%s\n\n", strings.Join(parts, " "))
		}
	}

	if err := os.MkdirAll(filepath.Dir(m.iniPath), 0700); err != nil {
		vtui.DebugLog("MACRO: Failed to create macro directory: %v", err)
		return
	}
	err := os.WriteFile(m.iniPath, []byte(sb.String()), 0600)
	if err != nil {
		vtui.DebugLog("MACRO: Failed to save: %v", err)
		return
	}
	_ = os.Chmod(m.iniPath, 0600)
}

// MacroAssignFrame is a modal frame that captures a key combination to assign a macro.
type MacroAssignFrame struct {
	*vtui.Window
	mgr *MacroManager
}

func NewMacroAssignFrame(m *MacroManager) *MacroAssignFrame {
	width, height := 42, 7
	base := vtui.NewCenteredDialog(width, height, i18n.Msg("Macro.AssignTitle"))
	f := &MacroAssignFrame{
		Window: base,
		mgr:    m,
	}

	prompt := vtui.NewText(0, 0, i18n.Msg("Macro.AssignPrompt"), vtui.Palette[vtui.ColDialogText])
	f.AddItem(prompt)

	cancelPrompt := vtui.NewText(0, 0, i18n.Msg("Macro.AssignCancel"), vtui.Palette[vtui.ColDialogText])
	f.AddItem(cancelPrompt)

	vbox := vtui.NewVBoxLayout(f.X1+2, f.Y1+2, width-4, height-4)
	vbox.Add(prompt, vtui.Margins{}, vtui.AlignCenter)
	vbox.Add(cancelPrompt, vtui.Margins{Top: 1}, vtui.AlignCenter)
	vbox.Apply()

	return f
}

func (f *MacroAssignFrame) ProcessKey(e *vtinput.InputEvent) bool {
	if e.Type == vtinput.FocusEventType {
		return f.Window.ProcessKey(e)
	}

	if !e.KeyDown {
		return false
	}

	if e.VirtualKeyCode == vtinput.VK_ESCAPE {
		f.mgr.Buffer = nil
		f.SetExitCode(-1)
		vtui.FrameManager.Redraw()
		return true
	}

	// Only ignore "pure" modifiers without any other key.
	// Everything else (including Esc and Alt-combos) can be a macro.
	switch e.VirtualKeyCode {
	case vtinput.VK_SHIFT, vtinput.VK_LSHIFT, vtinput.VK_RSHIFT,
		vtinput.VK_CONTROL, vtinput.VK_LCONTROL, vtinput.VK_RCONTROL,
		vtinput.VK_MENU, vtinput.VK_LMENU, vtinput.VK_RMENU,
		vtinput.VK_CAPITAL, vtinput.VK_NUMLOCK, vtinput.VK_SCROLL:
		return false
	}

	key := keymap.EventToFarString(e)
	if f.mgr.Macros == nil {
		f.mgr.Macros = make(map[string]map[string][]*vtinput.InputEvent)
	}

	area := f.mgr.StartArea
	if area == "" {
		area = "Common"
	}
	if f.mgr.Macros[area] == nil {
		f.mgr.Macros[area] = make(map[string][]*vtinput.InputEvent)
	}

	keyDesc := key
	var msg string

	removedIni := false
	if f.mgr.Macros[area] != nil {
		if _, exists := f.mgr.Macros[area][key]; exists {
			delete(f.mgr.Macros[area], key)
			removedIni = true
		}
	}
	if !removedIni && f.mgr.Macros["Common"] != nil {
		if _, exists := f.mgr.Macros["Common"][key]; exists {
			delete(f.mgr.Macros["Common"], key)
			removedIni = true
			area = "Common"
		}
	}

	removedLua := false
	if f.mgr.Lua != nil && f.mgr.Lua.Remove(area, key) {
		scriptDir := filepath.Join(config.GetF4ConfigDir(), "Macros", "scripts")
		scriptPath := filepath.Join(scriptDir, RecordedMacroFileName(area, key))
		os.Remove(scriptPath)
		removedLua = true
	}

	if len(f.mgr.Buffer) == 0 {
		if removedIni || removedLua {
			msg = fmt.Sprintf("Macro removed from key:\n%s\nArea: %s", keyDesc, area)
		} else {
			msg = fmt.Sprintf("No macro found for key:\n%s", keyDesc)
		}
		if removedIni {
			f.mgr.Save()
		}
	} else {
		if removedIni {
			f.mgr.Save()
		}
		if config.App.MacroRecordFormat == 1 {
			scriptDir := filepath.Join(config.GetF4ConfigDir(), "Macros", "scripts")
			err := f.mgr.SaveRecordedMacro(scriptDir, area, key, "", f.mgr.Buffer)
			if err != nil {
				msg = fmt.Sprintf("Failed to save Lua macro:\n%v", err)
			} else {
				msg = fmt.Sprintf("Lua macro assigned to key:\n%s\nArea: %s", keyDesc, area)
			}
		} else {
			if f.mgr.Macros[area] == nil {
				f.mgr.Macros[area] = make(map[string][]*vtinput.InputEvent)
			}
			f.mgr.Macros[area][key] = f.mgr.Buffer
			f.mgr.Save()
			msg = fmt.Sprintf("Macro assigned to key:\n%s\nArea: %s", keyDesc, area)
		}
	}

	f.mgr.Buffer = nil
	f.SetExitCode(0)

	vtui.FrameManager.PostTask(func() {
		vtui.ShowMessage(" Macro ", msg, []string{"&Ok"})
	})

	vtui.FrameManager.Redraw()
	return true
}

func (f *MacroAssignFrame) ProcessMouse(e *vtinput.InputEvent) bool {
	return true // Block clicks from falling through
}
func (f *MacroAssignFrame) GetType() vtui.FrameType { return vtui.TypeDialog }
func (f *MacroAssignFrame) IsModal() bool           { return true }
func (f *MacroAssignFrame) GetTitle() string        { return "Macro Assign" }

func (f *MacroAssignFrame) SetExitCode(code int) {
	f.mgr.Assigning = false
	f.Window.SetExitCode(code)
}
