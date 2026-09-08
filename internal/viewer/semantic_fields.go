package viewer

// ponytail: copied from cmd/f4/semantic.go rather than hoisted. semantic.go is
// split across five packages and Task 34 owns the split; the panel, terminal,
// editor and command-line slices need these same three readers. Give them one
// home when the last slice leaves, and delete this file.

import (
	"path/filepath"

	"github.com/unxed/f4/internal/numeric"
)

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
func semanticBaseName(v interface{ Base(string) string }, path string) string {
	if path == "" {
		return ""
	}
	if v != nil {
		return v.Base(path)
	}
	return filepath.Base(path)
}
