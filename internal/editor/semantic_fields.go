package editor

// ponytail: the second copy of these readers, after internal/viewer's. They
// come from cmd/f4/semantic.go, which is split across five packages and whose
// split Task 34 owns; the panel and command-line slices will want them too.
//
// Two copies is where copying stops being cheaper than hoisting. Task 34 gives
// them one home — reachable by viewer, editor, panel and cmdline, so layer 0 —
// and deletes this file and internal/viewer/semantic_fields.go with it.

import "path/filepath"

func semanticString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
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
