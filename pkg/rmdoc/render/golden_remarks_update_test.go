package render

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

func remarksFixturesForGoldens(t *testing.T, root string) []string {
	t.Helper()

	var fixtures []string

	globs := []string{
		filepath.Join(root, "cmd", "remarquee-ui", "testdata", "*.rmdoc"),
		filepath.Join(root, "cmd", "remarquee-ui", "testdata", "generated", "*.rmdoc"),
	}

	for _, g := range globs {
		matches, err := filepath.Glob(g)
		if err != nil {
			t.Fatalf("glob %q: %v", g, err)
		}
		fixtures = append(fixtures, matches...)
	}

	// Stable order to keep logs deterministic.
	sort.Strings(fixtures)
	return fixtures
}

func TestUpdateRemarksGoldens(t *testing.T) {
	if !*updateGolden {
		t.Skip("skipping: pass -update-golden to regenerate committed remarks goldens")
	}

	root := repoRootFromThisFile(t)
	fixtures := remarksFixturesForGoldens(t, root)
	if len(fixtures) == 0 {
		t.Fatalf("no .rmdoc fixtures found")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	work := t.TempDir()
	for _, fixture := range fixtures {
		if _, err := os.Stat(fixture); err != nil {
			t.Logf("skip missing fixture: %s (%v)", fixture, err)
			continue
		}
		_ = ensureRemarksReferencePDF(ctx, t, work, fixture)
	}
}
