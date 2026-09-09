package iosfs

import (
	"go/format"
	"os"
	"testing"
)

func TestCoverageFormatDiagnostic(t *testing.T) {
	source, err := os.ReadFile("coverage_format_input.txt")
	if err != nil {
		t.Fatal(err)
	}
	formatted, err := format.Source(source)
	if err != nil {
		t.Fatal(err)
	}
	t.Fatalf("gofmt output:\n%s", formatted)
}