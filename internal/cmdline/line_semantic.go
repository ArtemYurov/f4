package cmdline

// The command line's side of the GUI semantic protocol: what an external UI
// is told the prompt currently holds. Go requires it with CommandLine; it was
// in cmd/f4/semantic.go, which serves every frame.

import (
	"github.com/unxed/f4/sdk/extui"
	"github.com/unxed/f4/internal/semantic"
	"github.com/unxed/vtui"
)

func (cl *CommandLine) SemanticModel(ctx *vtui.SemanticContext) *extui.CommandLineModel {
	return &extui.CommandLineModel{
		ID:         vtui.SemanticID(cl),
		Visible:    cl.IsVisible(),
		Focused:    cl.IsFocused(),
		Prompt:     cl.Prompt,
		PromptRuns: semantic.RunsFromCells(cl.RichPrompt),
		Text:       cl.Edit.GetText(),
		Empty:      cl.IsEmpty(),
	}
}
