package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIgnorePatternMatching(t *testing.T) {
	body := `# comments and blanks are skipped

deps
experiments/spikes/*/vendor
build/**/cache
/top-only
docs/**
`
	ig := parseIgnore(body, true)
	cases := []struct {
		name string
		rel  string
		want bool
	}{
		{"a bare segment matches at any depth", "a/b/deps/x/README.md", true},
		{"a bare segment matches the directory itself", "deps", true},
		{"a bare segment does not match a longer name", "a/depsx/README.md", false},
		{"a rooted glob matches one segment", "experiments/spikes/s2/vendor/lib.md", true},
		{"a rooted glob's star does not cross a separator", "experiments/spikes/s2/deep/vendor/lib.md", false},
		{"double star crosses separators", "build/a/b/cache/x.md", true},
		{"a leading slash anchors to the design root", "top-only/x.md", true},
		{"an anchored pattern does not match deeper", "sub/top-only/x.md", false},
		{"a trailing double star covers a whole subtree", "docs/a/b.md", true},
		{"an unlisted path is kept", "BUILD/core.md", false},
		{"the design root itself is kept", ".", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ig.skips(tc.rel); got != tc.want {
				t.Fatalf("skips(%q) = %v, want %v", tc.rel, got, tc.want)
			}
		})
	}
}

func TestIgnoreNilAndEmptyIgnoreNothing(t *testing.T) {
	var nilSet *designIgnore
	if nilSet.skips("anything") {
		t.Fatal("a nil ignore list must ignore nothing")
	}
	if parseIgnore("# only a comment\n\n", true).skips("anything") {
		t.Fatal("an ignore list with no patterns must ignore nothing")
	}
}

// ignoreDesign writes a design with one clean document and one vendored
// subtree carrying an em dash and a duplicated table.
func ignoreDesign(t *testing.T, ignoreBody string) string {
	t.Helper()
	d := t.TempDir()
	if err := os.MkdirAll(filepath.Join(d, "experiments", "spikes", "s1", "deps", "lib"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(d, "experiments", "spikes", "s2", "deps", "lib"), 0755); err != nil {
		t.Fatal(err)
	}
	vendored := "# vendored\n\nprose with an em dash — here.\n\n| a | b |\n|---|---|\n| 1 | 2 |\n"
	for _, s := range []string{"s1", "s2"} {
		p := filepath.Join(d, "experiments", "spikes", s, "deps", "lib", "README.md")
		if err := os.WriteFile(p, []byte(vendored), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(d, "BUILD.md"), []byte("# build\n\nclean prose.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if ignoreBody != "" {
		if err := os.WriteFile(filepath.Join(d, IgnoreFileName), []byte(ignoreBody), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return d
}

func TestIgnoreSilencesTheDesignTreeWalkers(t *testing.T) {
	noisy := CheckLedger(ignoreDesign(t, ""))
	if len(noisy.Warns) == 0 {
		t.Fatal("the fixture must be noisy without an ignore list")
	}
	quiet := CheckLedger(ignoreDesign(t, "# vendored code, not design\nexperiments/spikes/*/deps\n"))
	for _, w := range quiet.Warns {
		if strings.Contains(w, "deps/lib") {
			t.Fatalf("an ignored path still produced a finding: %v", quiet.Warns)
		}
	}
}

func TestIgnoreCountIsOnTheGlCheckedLine(t *testing.T) {
	g := CheckLedger(ignoreDesign(t, "experiments/spikes/*/deps\n"))
	var b strings.Builder
	g.Emit(&b)
	if !strings.Contains(b.String(), "2 paths ignored (.machineryignore)") {
		t.Fatalf("the checked: line does not carry the ignored count:\n%s", b.String())
	}
	// no file, no segment: a design that ignores nothing says nothing
	g = CheckLedger(ignoreDesign(t, ""))
	b.Reset()
	g.Emit(&b)
	if strings.Contains(b.String(), "paths ignored") {
		t.Fatalf("a design with no ignore file must not report a count:\n%s", b.String())
	}
	// the file present but matching nothing still reports, so a list that has
	// stopped matching is visible rather than silent
	g = CheckLedger(ignoreDesign(t, "nothing-matches-this\n"))
	b.Reset()
	g.Emit(&b)
	if !strings.Contains(b.String(), "0 paths ignored (.machineryignore)") {
		t.Fatalf("a matching-nothing list must still report zero:\n%s", b.String())
	}
}

func TestIgnoreIsRereadWhenTheFileChanges(t *testing.T) {
	d := ignoreDesign(t, "experiments/spikes/*/deps\n")
	if !DesignIgnores(d, "experiments/spikes/s1/deps/lib/README.md") {
		t.Fatal("the pattern must match before the edit")
	}
	if err := os.WriteFile(filepath.Join(d, IgnoreFileName), []byte("# nothing now\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// the memo is keyed by the file's stamp, so an edit is picked up
	if DesignIgnores(d, "experiments/spikes/s1/deps/lib/README.md") {
		t.Fatal("the memoized list was served stale after an edit")
	}
}

func TestIgnoreCacheUsesContentNotMtimeAndSize(t *testing.T) {
	d := t.TempDir()
	path := filepath.Join(d, IgnoreFileName)
	if err := os.WriteFile(path, []byte("ignored-a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !DesignIgnores(d, "ignored-a/file.md") {
		t.Fatal("first ignore body did not load")
	}
	if err := os.WriteFile(path, []byte("ignored-b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	if DesignIgnores(d, "ignored-a/file.md") || !DesignIgnores(d, "ignored-b/file.md") {
		t.Fatal("same-size rewrite with restored mtime served stale ignore patterns")
	}
}

func TestIgnoreKeepsEmbedMarkersOutOfVendoredTrees(t *testing.T) {
	d := ignoreDesign(t, "experiments/spikes/*/deps\n")
	marker := "<!-- machinery:embed from=\"x.md\" table=\"a,b\" claims=\"subset\" -->\n| a | b |\n|---|---|\n| 1 | 2 |\n"
	p := filepath.Join(d, "experiments", "spikes", "s1", "deps", "lib", "EMBED.md")
	if err := os.WriteFile(p, []byte(marker), 0644); err != nil {
		t.Fatal(err)
	}
	if EmbedActive(d) {
		t.Fatal("a marker inside an ignored tree must not arm Ge")
	}
}
