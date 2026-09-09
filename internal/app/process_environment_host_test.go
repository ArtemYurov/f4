package app

import (
	"testing"

	"github.com/unxed/f4/vfs"
)

func TestCoreAPIProcessEnvironmentHost(t *testing.T) {
	const name = "F4_LUNOBOT_PROCESS_ENV_HOST_TEST"
	api := &coreAPI{}
	t.Cleanup(func() {
		if _, err := api.ApplyProcessEnvironment([]vfs.ProcessEnvironmentChange{{Name: name, Unset: true}}); err != nil {
			t.Errorf("cleanup environment variable: %v", err)
		}
	})

	before := api.SnapshotProcessEnvironment()
	after, err := api.ApplyProcessEnvironment([]vfs.ProcessEnvironmentChange{{Name: name, Value: "covered"}})
	if err != nil {
		t.Fatalf("ApplyProcessEnvironment: %v", err)
	}
	if after.Generation < before.Generation {
		t.Fatalf("environment generation went backwards: %d -> %d", before.Generation, after.Generation)
	}
	for _, variable := range after.Variables {
		if variable.Name == name {
			if variable.Value != "covered" {
				t.Fatalf("environment value = %q, want covered", variable.Value)
			}
			return
		}
	}
	t.Fatalf("environment snapshot does not contain %q", name)
}
