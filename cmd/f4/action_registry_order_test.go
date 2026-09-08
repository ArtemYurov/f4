package main

import (
	"os"
	"strings"
	"testing"
)

// The order actions are presented in is what the user reads in the menu, so it
// is behaviour and not an implementation detail. These tests hold it still
// while the registry's 174 registration calls are split across packages.

func TestActionOrderIsStable(t *testing.T) {
	data, err := os.ReadFile("testdata/action_order.golden")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Fields(string(data))

	var got []string
	for _, action := range GetOrderedActions() {
		got = append(got, action.Name)
	}

	for index := 0; index < len(want) && index < len(got); index++ {
		if got[index] != want[index] {
			t.Fatalf("action %d = %q, want %q\n"+
				"the menu the user sees has been rearranged; if that is intended, "+
				"move the entry in actionMenuOrder and regenerate "+
				"testdata/action_order.golden", index, got[index], want[index])
		}
	}
	if len(got) != len(want) {
		t.Fatalf("presented actions = %d, want %d: %v", len(got), len(want), symmetricDifference(got, want))
	}
}

func TestActionOrderCoversRegistry(t *testing.T) {
	if len(actionOrder) != len(actionRegistry) {
		t.Fatalf("actionOrder holds %d keys, actionRegistry %d", len(actionOrder), len(actionRegistry))
	}
	seen := make(map[string]int, len(actionOrder))
	for _, key := range actionOrder {
		seen[key]++
		if _, registered := actionRegistry[key]; !registered {
			t.Errorf("actionOrder names %q, which is not registered", key)
		}
	}
	for key, count := range seen {
		if count != 1 {
			t.Errorf("actionOrder names %q %d times, want once", key, count)
		}
	}
	for key := range actionRegistry {
		if seen[key] == 0 {
			t.Errorf("registered action %q is missing from actionOrder", key)
		}
	}
}

// TestActionOrderIndependentOfRegistrationSequence is the property the split
// depends on: once the 174 RegisterAction calls live in different packages, Go
// decides their sequence from the import graph, and the answer must not change.
func TestActionOrderIndependentOfRegistrationSequence(t *testing.T) {
	preserveActionRegistry(t)

	forward := []Action{
		{Name: "Workspace.New", Area: "Shell"},
		{Name: "App.Help", Area: "Common"},
		{Name: "App.ScreenGrab", Area: "Common"},
	}
	reversed := []Action{forward[2], forward[1], forward[0]}

	registerOnly := func(actions []Action) []string {
		actionRegistry = make(map[string]Action, len(actions))
		actionOrder = nil
		for _, action := range actions {
			RegisterAction(action)
		}
		var names []string
		for _, action := range GetOrderedActions() {
			names = append(names, action.Name)
		}
		return names
	}

	first := registerOnly(forward)
	second := registerOnly(reversed)
	if strings.Join(first, ",") != strings.Join(second, ",") {
		t.Fatalf("registration sequence decides presentation order: %v vs %v", first, second)
	}
	if want := "App.ScreenGrab,App.Help,Workspace.New"; strings.Join(first, ",") != want {
		t.Fatalf("presentation order = %v, want %s (actionMenuOrder's order)", first, want)
	}
}

func symmetricDifference(got, want []string) []string {
	inWant := make(map[string]bool, len(want))
	for _, name := range want {
		inWant[name] = true
	}
	inGot := make(map[string]bool, len(got))
	for _, name := range got {
		inGot[name] = true
	}
	var only []string
	for _, name := range got {
		if !inWant[name] {
			only = append(only, "+"+name)
		}
	}
	for _, name := range want {
		if !inGot[name] {
			only = append(only, "-"+name)
		}
	}
	return only
}
