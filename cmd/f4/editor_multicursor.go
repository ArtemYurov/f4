package main

import (
	"sort"
	"strings"

	"github.com/unxed/vtinput"
)

// Multi-caret editing. Every key that acts on the text through more than one
// caret goes through one primitive: build the whole set of replacements
// against the text as it stands, then apply them.
//
// Building first is what keeps the offsets comparable. A caret's offset means
// nothing once a neighbour has inserted or removed bytes before it, so the
// edits are described up front, applied from the last one backwards — which
// never moves the text the earlier ones describe — and the carets are then
// placed by arithmetic rather than by re-deriving them from the buffer.

// caretEdit replaces del bytes at off with ins, on behalf of one caret. A
// caret that has nothing to do this round still gets an entry, with neither a
// deletion nor an insertion, so that it survives the edit and is carried along
// by whatever the other carets did before it.
type caretEdit struct {
	off int
	del int
	ins []byte
	// after puts the caret past the inserted text (typing) rather than at
	// the edit site (deleting).
	after bool
	// primary marks the caret that stays the primary one afterwards.
	primary bool
}

// buildCaretEdits calls build for every caret, in ascending order, and marks
// the entry belonging to the primary caret.
func (ev *EditorView) buildCaretEdits(build func(off int) caretEdit) []caretEdit {
	primary := ev.caretOffset()
	offsets := ev.caretOffsets()
	edits := make([]caretEdit, 0, len(offsets))
	primarySeen := false
	for _, off := range offsets {
		edit := build(off)
		if off == primary && !primarySeen {
			edit.primary = true
			primarySeen = true
		}
		edits = append(edits, edit)
	}
	if !primarySeen && len(edits) > 0 {
		edits[0].primary = true
	}
	return edits
}

// applyCaretEdits performs the whole set as one undoable change and leaves a
// caret at each site. It reports whether the buffer changed.
func (ev *EditorView) applyCaretEdits(edits []caretEdit, op undoOpType) bool {
	if len(edits) == 0 {
		return false
	}
	sort.SliceStable(edits, func(i, j int) bool { return edits[i].off < edits[j].off })

	// Two carets close enough for their deletions to meet: the later edit
	// keeps what it claimed and the earlier one gives up the overlap, so no
	// byte is deleted twice.
	for i := 0; i+1 < len(edits); i++ {
		if end := edits[i].off + edits[i].del; end > edits[i+1].off {
			edits[i].del = edits[i+1].off - edits[i].off
			if edits[i].del < 0 {
				edits[i].del = 0
			}
		}
	}

	changed := false
	for _, edit := range edits {
		if edit.del > 0 || len(edit.ins) > 0 {
			changed = true
			break
		}
	}
	if !changed {
		return false
	}

	ev.noteBufferEdit()
	ev.saveUndo(op)
	ev.lastOp = op
	ev.modified = true

	first := edits[0].off
	for i := len(edits) - 1; i >= 0; i-- {
		edit := edits[i]
		if edit.del > 0 {
			ev.pt.Delete(edit.off, edit.del)
			ev.li.UpdateAfterDelete(edit.off, edit.del)
		}
		if len(edit.ins) > 0 {
			ev.pt.Insert(edit.off, edit.ins)
			ev.li.UpdateAfterInsert(edit.off, edit.ins)
		}
	}

	firstLine := ev.li.GetLineAtOffset(first)
	ev.invalidateStates(firstLine)
	ev.engine.InvalidateFrom(firstLine)

	// Each caret lands at its own site, shifted by everything the edits
	// before it added or removed.
	offsets := make([]int, len(edits))
	primary := 0
	shift := 0
	for i, edit := range edits {
		off := edit.off + shift
		if edit.after {
			off += len(edit.ins)
		}
		offsets[i] = off
		if edit.primary {
			primary = i
		}
		shift += len(edit.ins) - edit.del
	}
	ev.setCaretOffsets(offsets, primary)
	return true
}

// setCaretOffsets rebuilds the caret set from absolute offsets, keeping the
// one at index primary as the primary caret. Carets that met in the middle of
// a deletion arrive here as duplicates and leave as one.
func (ev *EditorView) setCaretOffsets(offsets []int, primary int) {
	if len(offsets) == 0 {
		return
	}
	if primary < 0 || primary >= len(offsets) {
		primary = 0
	}
	size := ev.pt.Size()
	clamp := func(off int) int {
		if off < 0 {
			return 0
		}
		if off > size {
			return size
		}
		return off
	}

	primaryOff := clamp(offsets[primary])
	ev.CursorLine = ev.li.GetLineAtOffset(primaryOff)
	ev.CursorPos = primaryOff - ev.li.GetLineOffset(ev.CursorLine)
	ev.CursorVirtualSpaces = 0

	extras := ev.extraCursors[:0]
	for i, off := range offsets {
		if i == primary {
			continue
		}
		off = clamp(off)
		if off == primaryOff {
			continue
		}
		extras = append(extras, off)
	}
	sort.Ints(extras)
	deduped := extras[:0]
	for i, off := range extras {
		if i == 0 || off != extras[i-1] {
			deduped = append(deduped, off)
		}
	}
	ev.extraCursors = deduped

	ev.updateDesiredVisualCol()
	ev.ensureCursorVisible()
}

// processMultiCursorKey handles the keys that act through every caret and
// reports whether it took the key. Anything it declines collapses the set and
// is then handled the ordinary single-caret way.
func (ev *EditorView) processMultiCursorKey(e *vtinput.InputEvent) bool {
	ctrl := e.ControlKeyState&(vtinput.LeftCtrlPressed|vtinput.RightCtrlPressed) != 0
	alt := e.ControlKeyState&(vtinput.LeftAltPressed|vtinput.RightAltPressed) != 0
	shift := e.ControlKeyState&vtinput.ShiftPressed != 0

	// A suggestion is a single caret's idea of what comes next, so it has no
	// place in a multi-caret edit.
	ev.acMatches = nil

	switch e.VirtualKeyCode {
	case vtinput.VK_BACK:
		if ctrl || alt {
			return false
		}
		return ev.multiDeleteBackward()
	case vtinput.VK_DELETE:
		// Shift+Del is Cut and Ctrl+Del deletes a word: both are still
		// single-caret operations.
		if ctrl || alt || shift {
			return false
		}
		return ev.multiDeleteForward()
	case vtinput.VK_RETURN:
		if ctrl || alt {
			return false
		}
		return ev.multiInsertNewline()
	case vtinput.VK_TAB:
		// Shift+Tab unindents, which is not a per-caret insertion.
		if ctrl || alt || shift {
			return false
		}
		return ev.multiInsertTab()
	}

	if e.Char != 0 && e.Char >= 32 && !ctrl && !alt {
		return ev.multiInsertText([]byte(string(e.Char)))
	}
	return false
}

// multiInsertText types the same bytes at every caret.
func (ev *EditorView) multiInsertText(data []byte) bool {
	return ev.applyCaretEdits(ev.buildCaretEdits(func(off int) caretEdit {
		edit := caretEdit{off: off, ins: data, after: true}
		if ev.overtype {
			edit.del = ev.overtypeWidthAt(off)
		}
		return edit
	}), opTyping)
}

// multiInsertNewline splits the line at every caret, each one taking the
// indentation of the line it was on when auto-indent is on.
func (ev *EditorView) multiInsertNewline() bool {
	return ev.applyCaretEdits(ev.buildCaretEdits(func(off int) caretEdit {
		data := []byte("\n")
		if ev.AutoIndent {
			data = append(data, ev.lineIndentAt(off)...)
		}
		return caretEdit{off: off, ins: data, after: true}
	}), opOther)
}

// multiInsertTab inserts a tab at every caret, expanded to the next tab stop
// of that caret's own column when tabs are expanded to spaces.
func (ev *EditorView) multiInsertTab() bool {
	tabSize := ev.TabSize
	if tabSize <= 0 {
		tabSize = 8
	}
	return ev.applyCaretEdits(ev.buildCaretEdits(func(off int) caretEdit {
		data := []byte("\t")
		if ev.ExpandTabs > 0 {
			_, vCol := ev.engine.LogicalToVisual(off)
			data = []byte(strings.Repeat(" ", tabSize-(vCol%tabSize)))
		}
		return caretEdit{off: off, ins: data, after: true}
	}), opTyping)
}

// multiDeleteBackward is Backspace at every caret. A caret at the very start
// of the buffer has nothing to delete and simply stays where it is.
func (ev *EditorView) multiDeleteBackward() bool {
	return ev.applyCaretEdits(ev.buildCaretEdits(func(off int) caretEdit {
		if off <= 0 {
			return caretEdit{off: off}
		}
		line := ev.li.GetLineAtOffset(off)
		lineStart := ev.li.GetLineOffset(line)
		start := off - 1
		if off == lineStart {
			// Joining with the line above takes the whole terminator, so a
			// CRLF file does not keep a stray carriage return.
			if off >= 2 {
				if prefix, err := ev.pt.GetRange(off-2, 2); err == nil &&
					len(prefix) == 2 && prefix[0] == '\r' && prefix[1] == '\n' {
					start = off - 2
				}
			}
		} else {
			start = lineStart + ev.previousDeletionBoundaryInLine(lineStart, off-lineStart)
		}
		if start < 0 || start >= off {
			return caretEdit{off: off}
		}
		return caretEdit{off: start, del: off - start}
	}), opOther)
}

// multiDeleteForward is Del at every caret, following the single-caret rule:
// inside a line it removes one grapheme, at the end of one it removes the
// single byte that starts the line break.
func (ev *EditorView) multiDeleteForward() bool {
	size := ev.pt.Size()
	return ev.applyCaretEdits(ev.buildCaretEdits(func(off int) caretEdit {
		if off >= size {
			return caretEdit{off: off}
		}
		line := ev.li.GetLineAtOffset(off)
		lineStart := ev.li.GetLineOffset(line)
		lineLen := ev.getLineLength(line)
		end := off + 1
		if pos := off - lineStart; pos < lineLen {
			end = lineStart + ev.nextGraphemeBoundaryInLine(lineStart, lineLen, pos)
		}
		if end <= off {
			return caretEdit{off: off}
		}
		return caretEdit{off: off, del: end - off}
	}), opOther)
}

// overtypeWidthAt is how much the character under a caret takes, for overtype
// mode. At the end of a line there is nothing to type over: overtype never
// eats the line break.
func (ev *EditorView) overtypeWidthAt(off int) int {
	line := ev.li.GetLineAtOffset(off)
	lineStart := ev.li.GetLineOffset(line)
	lineLen := ev.getLineLength(line)
	pos := off - lineStart
	if pos < 0 || pos >= lineLen {
		return 0
	}
	next := ev.nextGraphemeBoundaryInLine(lineStart, lineLen, pos)
	if next <= pos {
		return 0
	}
	return next - pos
}

// lineIndentAt returns the leading whitespace of the line the offset is on,
// which is what a new line started there inherits.
func (ev *EditorView) lineIndentAt(off int) []byte {
	var indent []byte
	for _, r := range ev.getLogicalLineRunes(ev.li.GetLineAtOffset(off)) {
		if r != ' ' && r != '\t' {
			break
		}
		indent = append(indent, []byte(string(r))...)
	}
	return indent
}
