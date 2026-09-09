package panel

import (
	"testing"

	"github.com/unxed/f4/internal/editor"
	"github.com/unxed/f4/internal/testutil"
	"github.com/unxed/f4/plugins/visren"
	"github.com/unxed/vtui"
)

func TestOpenVisRenEditorCreatesTemporaryEditorScreen(t *testing.T) {
	t.Cleanup(testutil.SwapFrameManager(t))
	scr := vtui.NewSilentScreenBuf()
	scr.AllocBuf(100, 30)
	vtui.FrameManager.Init(scr)
	initialScreens := len(vtui.FrameManager.Screens)

	pf := &PanelsFrame{LastW: 100, LastH: 30}
	err := pf.OpenVisRenEditor(visren.EditorRequest{
		Title:      "VisRen rules",
		Content:    []byte("*.go => go\n"),
		CursorLine: 2,
		CursorCol:  4,
	})
	if err != nil {
		t.Fatalf("OpenVisRenEditor() error = %v", err)
	}

	if len(vtui.FrameManager.Screens) != initialScreens+1 {
		t.Fatalf("screens = %d, want %d", len(vtui.FrameManager.Screens), initialScreens+1)
	}
	screen := vtui.FrameManager.Screens[len(vtui.FrameManager.Screens)-1]
	if len(screen.Frames) != 1 {
		t.Fatalf("frames = %d, want 1", len(screen.Frames))
	}
	ev, ok := screen.Frames[0].(*editor.EditorView)
	if !ok {
		t.Fatalf("screen frame has type %T, want *editor.EditorView", screen.Frames[0])
	}
	t.Cleanup(ev.Close)
	if ev.DisplayTitle != "VisRen rules" {
		t.Fatalf("editor title = %q, want %q", ev.DisplayTitle, "VisRen rules")
	}
	if ev.CursorLine != 2 || ev.CursorPos != 4 {
		t.Fatalf("cursor = (%d, %d), want (2, 4)", ev.CursorLine, ev.CursorPos)
	}
}
