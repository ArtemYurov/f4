package main

import (
	"reflect"
	"testing"

	"github.com/unxed/f4/piecetable"
	"github.com/unxed/vtinput"
	"github.com/unxed/vtui"
)

func multiCursorEditor(t *testing.T, text string) *EditorView {
	t.Helper()
	vtui.FrameManager.Init(vtui.NewSilentScreenBuf())
	SetDefaultF4Palette()
	ev := NewEditorView(piecetable.New([]byte(text)), nil, "test.txt")
	t.Cleanup(ev.Close)
	ev.SetPosition(0, 0, 80, 12)
	return ev
}

func TestEditor_MultiCursor_ToggleAndOrder(t *testing.T) {
	ev := multiCursorEditor(t, "one\ntwo\nthree")

	if ev.multiCursor() {
		t.Fatal("a fresh editor already reports several carets")
	}
	// Placed out of order, kept in order.
	if !ev.toggleCursorAt(8) || !ev.toggleCursorAt(4) {
		t.Fatal("placing an extra caret reported no change")
	}
	if !reflect.DeepEqual(ev.extraCursors, []int{4, 8}) {
		t.Errorf("extra carets = %v, want [4 8]", ev.extraCursors)
	}
	if !ev.multiCursor() {
		t.Error("multiCursor() = false with two extra carets placed")
	}

	// Placing one twice removes it; the click that made it is the click that
	// takes it away.
	if !ev.toggleCursorAt(4) {
		t.Fatal("removing an extra caret reported no change")
	}
	if !reflect.DeepEqual(ev.extraCursors, []int{8}) {
		t.Errorf("extra carets = %v, want [8]", ev.extraCursors)
	}

	// The primary caret is not a member of the set and cannot be dropped.
	if ev.toggleCursorAt(ev.caretOffset()) {
		t.Error("the primary caret was toggled into the extra set")
	}
	if len(ev.extraCursors) != 1 {
		t.Errorf("extra carets = %v after clicking the primary one", ev.extraCursors)
	}

	// Offsets outside the buffer are refused rather than stored.
	if ev.toggleCursorAt(-1) || ev.toggleCursorAt(ev.pt.Size()+1) {
		t.Error("an out-of-range caret was accepted")
	}
}

func TestEditor_MultiCursor_CaretOffsetsIncludePrimary(t *testing.T) {
	ev := multiCursorEditor(t, "one\ntwo\nthree")
	ev.CursorLine = 1
	ev.CursorPos = 1 // offset 5
	ev.toggleCursorAt(0)
	ev.toggleCursorAt(9)

	if got, want := ev.caretOffsets(), []int{0, 5, 9}; !reflect.DeepEqual(got, want) {
		t.Errorf("caret offsets = %v, want %v", got, want)
	}

	// A stale caret on the primary's own offset is reported once.
	ev.extraCursors = []int{0, 5, 9}
	if got, want := ev.caretOffsets(), []int{0, 5, 9}; !reflect.DeepEqual(got, want) {
		t.Errorf("caret offsets with a duplicate = %v, want %v", got, want)
	}

	// And one past the end of the buffer is dropped rather than reported.
	ev.extraCursors = []int{0, ev.pt.Size() + 10}
	if got, want := ev.caretOffsets(), []int{0, 5}; !reflect.DeepEqual(got, want) {
		t.Errorf("caret offsets with a stale entry = %v, want %v", got, want)
	}
}

// A selection describes one caret's range, so building a caret set puts it
// down rather than leaving it on screen next to the new carets.
func TestEditor_MultiCursor_PlacementDropsSelection(t *testing.T) {
	ev := multiCursorEditor(t, "one\ntwo\nthree")
	ev.selActive = true
	ev.selAnchorOffset = 0
	ev.CursorPos = 3

	ev.toggleCursorAt(5)

	if ev.selActive || ev.rectSelActive {
		t.Error("the selection survived placing an extra caret")
	}
}

func TestEditor_MultiCursor_EscapeClearsBeforeClosing(t *testing.T) {
	ev := multiCursorEditor(t, "one\ntwo")
	ev.toggleCursorAt(4)

	// While carets are up the editor claims Escape from the hotkey
	// dispatcher, so Editor.Quit does not fire underneath it.
	esc := &vtinput.InputEvent{Type: vtinput.KeyEventType, KeyDown: true, VirtualKeyCode: vtinput.VK_ESCAPE}
	if !ev.VetoActionKey(esc) {
		t.Error("Escape was left to the hotkey dispatcher while carets were up")
	}
	if !ev.ProcessKey(esc) {
		t.Error("Escape was not handled by the editor")
	}
	if ev.multiCursor() {
		t.Error("Escape did not put the extra carets down")
	}
	// With the set gone, Escape goes back to being the editor's own key.
	if ev.VetoActionKey(esc) {
		t.Error("Escape is still being claimed after the carets were dropped")
	}
}

// Only the primary caret is wired into editing so far, so any other key
// collapses the set instead of leaving carets that the keystroke ignored.
func TestEditor_MultiCursor_TypingCollapsesTheSet(t *testing.T) {
	ev := multiCursorEditor(t, "one\ntwo")
	ev.toggleCursorAt(4)

	ev.ProcessKey(&vtinput.InputEvent{Type: vtinput.KeyEventType, KeyDown: true, Char: 'X'})

	if ev.multiCursor() {
		t.Error("typing left extra carets behind")
	}
	if got, want := ev.pt.String(), "Xone\ntwo"; got != want {
		t.Errorf("buffer = %q, want %q", got, want)
	}

	ev.toggleCursorAt(0)
	ev.ProcessKey(&vtinput.InputEvent{Type: vtinput.KeyEventType, KeyDown: true, VirtualKeyCode: vtinput.VK_DOWN})
	if ev.multiCursor() {
		t.Error("moving the caret left extra carets behind")
	}
}

func TestEditor_MultiCursor_UndoDropsTheSet(t *testing.T) {
	ev := multiCursorEditor(t, "one\ntwo")
	ev.ProcessKey(&vtinput.InputEvent{Type: vtinput.KeyEventType, KeyDown: true, Char: 'X'})
	ev.toggleCursorAt(4)

	ev.Undo()

	if ev.multiCursor() {
		t.Error("undo left carets pointing into text that was replaced")
	}
}

func TestEditor_MultiCursor_PaintsEachCaret(t *testing.T) {
	ev := multiCursorEditor(t, "one\ntwo\nthree")
	ev.CursorLine = 0
	ev.CursorPos = 0
	ev.toggleCursorAt(4) // start of "two"
	ev.SetFocus(true)

	scr := vtui.NewSilentScreenBuf()
	scr.AllocBuf(80, 13)
	ev.Show(scr)

	caretAttr := vtui.Palette[vtui.ColDialogEditSelected]
	// Text starts on screen row 1; the second logical line is row 2.
	cell := scr.GetCell(0, 2)
	if cell.Attributes != caretAttr {
		t.Errorf("extra caret cell attributes = %x, want the caret mark %x", cell.Attributes, caretAttr)
	}
	if cell.Char != 't' {
		t.Errorf("extra caret overwrote the character with %c", rune(cell.Char))
	}
	// The neighbouring cell is ordinary text.
	if got := scr.GetCell(1, 2); got.Attributes == caretAttr {
		t.Error("the caret mark spilled onto the next cell")
	}

	// Once the set is dropped the mark goes with it.
	ev.clearExtraCursors()
	ev.Show(scr)
	if got := scr.GetCell(0, 2); got.Attributes == caretAttr {
		t.Error("the caret mark stayed on screen after the set was dropped")
	}
}

func TestEditor_MultiCursor_AltClickGesture(t *testing.T) {
	alt := vtinput.InputEvent{
		Type:            vtinput.MouseEventType,
		KeyDown:         true,
		ButtonState:     vtinput.FromLeft1stButtonPressed,
		ControlKeyState: vtinput.LeftAltPressed,
	}
	if !editorAddCursorClick(&alt) {
		t.Error("plain Alt+click was not recognised as caret placement")
	}

	withShift := alt
	withShift.ControlKeyState |= vtinput.ShiftPressed
	if editorAddCursorClick(&withShift) {
		t.Error("Alt+Shift+click was taken from the block selection gesture")
	}

	withCtrl := alt
	withCtrl.ControlKeyState |= vtinput.LeftCtrlPressed
	if editorAddCursorClick(&withCtrl) {
		t.Error("Ctrl+Alt+click was taken from the URL gesture")
	}

	dragged := alt
	dragged.MouseEventFlags |= vtinput.MouseMoved
	if editorAddCursorClick(&dragged) {
		t.Error("a drag was treated as caret placement")
	}
}

func TestEditor_MultiCursor_AltClickPlacesCaret(t *testing.T) {
	ev := multiCursorEditor(t, "one\ntwo\nthree")
	ev.CursorLine = 0
	ev.CursorPos = 0

	click := &vtinput.InputEvent{
		Type:            vtinput.MouseEventType,
		KeyDown:         true,
		ButtonState:     vtinput.FromLeft1stButtonPressed,
		ControlKeyState: vtinput.LeftAltPressed,
		MouseX:          1,
		MouseY:          2, // second text row
	}
	if !ev.ProcessMouse(click) {
		t.Fatal("Alt+click was not handled")
	}
	if got, want := ev.extraCursors, []int{5}; !reflect.DeepEqual(got, want) {
		t.Fatalf("extra carets = %v, want %v", got, want)
	}
	// The primary caret stays where it was: placing a caret is not a click.
	if ev.CursorLine != 0 || ev.CursorPos != 0 {
		t.Errorf("primary caret moved to line %d pos %d", ev.CursorLine, ev.CursorPos)
	}

	// Clicking the same spot again takes it away.
	if !ev.ProcessMouse(click) {
		t.Fatal("the second Alt+click was not handled")
	}
	if ev.multiCursor() {
		t.Errorf("extra carets = %v, want none", ev.extraCursors)
	}
}
