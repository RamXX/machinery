package repo_test

// C-REPO-23 (BUILD.md 7.2): no repo method exists to mutate an existing Activity
// body/occurredAt. activity-immutable is structural (append-only; no update
// path), so this is a compile/contract check over the Repo interface and runs in
// the default (hermetic) build, unlike the other C-REPO tests which are
// integration and behind the `ladybug` tag. Green by construction against the
// correct interface shape.

import (
	"reflect"
	"strings"
	"testing"

	"crm/internal/repo"
)

func TestCRepo23ActivityImmutableNoUpdatePath(t *testing.T) {
	rt := reflect.TypeOf((*repo.Repo)(nil)).Elem()

	// The append-only write method must exist.
	if _, ok := rt.MethodByName("SaveActivity"); !ok {
		t.Errorf("C-REPO-23: Repo must expose the append-only SaveActivity")
	}

	// No mutate/update path for Activity may exist.
	for _, name := range []string{"UpdateActivity", "SetActivity", "MutateActivity", "EditActivity", "ReplaceActivity"} {
		if _, ok := rt.MethodByName(name); ok {
			t.Errorf("C-REPO-23: Repo must not expose %s (activity-immutable)", name)
		}
	}
	// Defensive: SaveActivity is the only Activity-touching method.
	for i := 0; i < rt.NumMethod(); i++ {
		if n := rt.Method(i).Name; strings.Contains(n, "Activity") && n != "SaveActivity" {
			t.Errorf("C-REPO-23: unexpected Activity method %q may allow mutation", n)
		}
	}
}
