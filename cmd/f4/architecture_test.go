package main

import (
	"os/exec"
	"sort"
	"strings"
	"testing"
)

// The module boundary auditor. Four of the dependency rules in
// .ai-factory/ARCHITECTURE.md are mechanically checkable, and this is where
// they are checked. A rule that is only written down is a rule that gets
// broken by the next person who does not know it exists.
//
// Standard library only, and no new module dependency: the import graph comes
// from the toolchain itself.

const architectureModule = "github.com/unxed/f4"

// architectureLayers places each internal package in the dependency order the
// architecture document defines: 0 depends on nothing of ours, 4 is the
// application that composes the rest. Extracting a package adds one line here.
//
// internal/hideconsole is deliberately absent. It is a vendored fork carrying
// its own go.mod, so it is a separate module and `go list ./...` never returns
// it — adding it here would describe a package this test cannot see.
var architectureLayers = map[string]int{
	"internal/netproxy": 0,
	"internal/ttyx":     0,
	"internal/wincon":   0,
}

// architectureGOOS is the set of platforms the graph is collected for. One
// pass would only see the files the host's build tags select, and this
// repository keeps a quarter of its sources behind a tag: a windows-only
// upward import would be invisible from a darwin test run.
var architectureGOOS = []string{"", "linux", "windows"}

func TestArchitectureModuleBoundaries(t *testing.T) {
	graph := architectureImportGraph(t)

	// Rule 1: the public contract stays public. sdk/ is what third-party
	// plugins compile against and vfs/ is what they speak; an internal
	// import in either turns a private decision into a published one.
	t.Run("PublicContractStaysPublic", func(t *testing.T) {
		var offenders []string
		for importer, imports := range graph {
			if !underAny(importer, architectureModule+"/sdk/", architectureModule+"/vfs/", architectureModule+"/sdk", architectureModule+"/vfs") {
				continue
			}
			for _, imported := range imports {
				if strings.Contains(imported, "/internal/") {
					offenders = append(offenders, importer+" -> "+imported)
				}
			}
		}
		reportEdges(t, "a public package imports module-private code", offenders)
	})

	// Rule 2: nothing imports the entry point. cmd/f4 is a composition root,
	// and a composition root with importers is just another library.
	t.Run("NothingImportsTheEntryPoint", func(t *testing.T) {
		var offenders []string
		for importer, imports := range graph {
			for _, imported := range imports {
				if imported == architectureModule+"/cmd/f4" {
					offenders = append(offenders, importer+" -> "+imported)
				}
			}
		}
		reportEdges(t, "the entry point is imported", offenders)
	})

	// Rule 3: no upward imports into the application. internal/app composes
	// the lower layers; a lower layer reaching back into it is the cycle the
	// whole extraction exists to prevent.
	t.Run("NothingBelowTheApplicationImportsIt", func(t *testing.T) {
		application := architectureModule + "/internal/app"
		var offenders []string
		for importer, imports := range graph {
			if importer == architectureModule+"/cmd/f4" || importer == application {
				continue
			}
			for _, imported := range imports {
				if imported == application {
					offenders = append(offenders, importer+" -> "+imported)
				}
			}
		}
		reportEdges(t, "a package below the application imports it", offenders)
	})

	// Rule 4: the module's own import graph is acyclic. The compiler refuses
	// a cycle before this test ever runs, so this is a second pair of eyes
	// whose value is the message: it names the path, which a build error on
	// a twelve-package loop does not.
	t.Run("ImportGraphIsAcyclic", func(t *testing.T) {
		if cycle := findImportCycle(graph); cycle != nil {
			t.Fatalf("import cycle: %s", strings.Join(cycle, " -> "))
		}
	})
}

// TestArchitectureLayerMapMatchesTheTree keeps the layer map honest: a package
// listed here that no longer exists is a stale line, and the map is what later
// layer rules and the architecture document are checked against.
func TestArchitectureLayerMapMatchesTheTree(t *testing.T) {
	graph := architectureImportGraph(t)
	for suffix := range architectureLayers {
		if _, ok := graph[architectureModule+"/"+suffix]; !ok {
			t.Errorf("architectureLayers names %q, which is not a package in this module", suffix)
		}
	}
}

func architectureImportGraph(t *testing.T) map[string][]string {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not available")
	}
	graph := make(map[string][]string)
	for _, goos := range architectureGOOS {
		command := exec.Command("go", "list", "-f", "{{.ImportPath}} {{join .Imports \" \"}}", "./...")
		command.Dir = moduleRootDir(t)
		if goos != "" {
			command.Env = append(command.Environ(), "GOOS="+goos, "CGO_ENABLED=0")
		}
		out, err := command.Output()
		if err != nil {
			stderr := ""
			if exitErr, ok := err.(*exec.ExitError); ok {
				stderr = string(exitErr.Stderr)
			}
			// Not a skip: a silent skip would disable the auditor for the
			// rest of the migration, which is exactly when it is needed.
			t.Fatalf("go list for GOOS=%q: %v\n%s", goos, err, stderr)
		}
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			fields := strings.Fields(line)
			if len(fields) == 0 {
				continue
			}
			graph[fields[0]] = append(graph[fields[0]], fields[1:]...)
		}
	}
	return graph
}

func underAny(path string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if path == strings.TrimSuffix(prefix, "/") || strings.HasPrefix(path, strings.TrimSuffix(prefix, "/")+"/") {
			return true
		}
	}
	return false
}

// findImportCycle returns one cycle as the path that closes it, restricted to
// this module's own packages, or nil when there is none.
func findImportCycle(graph map[string][]string) []string {
	const (
		visiting = 1
		done     = 2
	)
	state := make(map[string]int, len(graph))
	var path []string
	var walk func(node string) []string
	walk = func(node string) []string {
		switch state[node] {
		case done:
			return nil
		case visiting:
			for index, seen := range path {
				if seen == node {
					return append(append([]string(nil), path[index:]...), node)
				}
			}
			return []string{node, node}
		}
		state[node] = visiting
		path = append(path, node)
		for _, next := range graph[node] {
			if !strings.HasPrefix(next, architectureModule+"/") && next != architectureModule {
				continue
			}
			if cycle := walk(next); cycle != nil {
				return cycle
			}
		}
		path = path[:len(path)-1]
		state[node] = done
		return nil
	}

	roots := make([]string, 0, len(graph))
	for node := range graph {
		roots = append(roots, node)
	}
	sort.Strings(roots)
	for _, node := range roots {
		if cycle := walk(node); cycle != nil {
			return cycle
		}
	}
	return nil
}

func reportEdges(t *testing.T, what string, offenders []string) {
	t.Helper()
	if len(offenders) == 0 {
		return
	}
	sort.Strings(offenders)
	offenders = dedupeStrings(offenders)
	t.Fatalf("%s:\n\t%s", what, strings.Join(offenders, "\n\t"))
}

func dedupeStrings(sorted []string) []string {
	out := sorted[:0]
	for index, value := range sorted {
		if index == 0 || value != sorted[index-1] {
			out = append(out, value)
		}
	}
	return out
}
