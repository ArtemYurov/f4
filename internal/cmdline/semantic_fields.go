package cmdline

// ponytail: the third copy of this reader, after internal/viewer's and
// internal/terminal's. It comes from cmd/f4/semantic.go, whose split across
// five packages Task 34 owns.
//
// Three copies is past the point where copying pays. Task 34 gives these
// readers one layer-0 home and deletes this file, internal/viewer/semantic_fields.go
// and internal/editor/semantic_fields.go with it.

import (
	"strings"
	"unicode/utf8"

	"github.com/unxed/f4/sdk/extui"
	"github.com/unxed/vtui"
)

func semanticRunsFromCells(cells []vtui.CharInfo) []extui.RunModel {
	if len(cells) == 0 {
		return nil
	}
	var runs []extui.RunModel
	var b strings.Builder
	var attr uint64
	haveRun := false
	flush := func() {
		if !haveRun {
			return
		}
		runs = append(runs, extui.RunModel{
			Text: b.String(),
			Attr: attr,
		})
		b.Reset()
	}
	for _, cell := range cells {
		if cell.Char == vtui.WideCharFiller {
			continue
		}
		ch := cellRune(cell.Char)
		if !haveRun {
			attr = cell.Attributes
			haveRun = true
		} else if cell.Attributes != attr {
			flush()
			attr = cell.Attributes
			haveRun = true
		}
		b.WriteRune(ch)
	}
	flush()
	return runs
}

func cellRune(ch uint64) rune {
	if ch == 0 || ch > utf8.MaxRune || (ch >= 0xD800 && ch <= 0xDFFF) {
		return ' '
	}
	return rune(ch)
}
