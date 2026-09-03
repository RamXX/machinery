package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeC4Fixture(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestStructurizrInventoryRejectsEntryBeyondFixedCeiling(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"c", "a", "b"} {
		writeC4Fixture(t, filepath.Join(dir, name), name)
	}
	f, err := os.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close() //nolint:errcheck // test cleanup
	if _, err := readStructurizrDirEntries(f, 2, "test inventory"); err == nil || !strings.Contains(err.Error(), "inventory bound") {
		t.Fatalf("high-entry inventory was accepted: %v", err)
	}
}

func TestFingerprintStructurizrTreeRejectsContinuousAppender(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tool.jar")
	writeC4Fixture(t, path, strings.Repeat("x", 1<<20))
	old := structurizrFingerprintAfterOpen
	t.Cleanup(func() { structurizrFingerprintAfterOpen = old })
	done := make(chan struct{})
	stopped := make(chan struct{})
	structurizrFingerprintAfterOpen = func(name string) {
		if name != "tool.jar" {
			return
		}
		structurizrFingerprintAfterOpen = func(string) {}
		first := make(chan struct{})
		go func() {
			defer close(stopped)
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
			if err != nil {
				close(first)
				return
			}
			defer f.Close() //nolint:errcheck // test mutation
			for i := 0; ; i++ {
				_, _ = f.Write([]byte("growth"))
				if i == 0 {
					close(first)
				}
				select {
				case <-done:
					return
				default:
				}
			}
		}()
		<-first
	}
	_, err := fingerprintStructurizrTree(dir)
	close(done)
	<-stopped
	if err == nil || !strings.Contains(err.Error(), "changed while hashing") {
		t.Fatalf("continuous appender was accepted: %v", err)
	}
}

func TestStructurizrClosureRecursivelyAcceptsBoundLocalIncludes(t *testing.T) {
	project := t.TempDir()
	design := filepath.Join(project, "design")
	entry := filepath.Join(design, "workspace.dsl")
	fragment := filepath.Join(project, "fragments", "model.dsl")
	view := filepath.Join(project, "fragments", "view.dsl")
	writeC4Fixture(t, entry, "workspace {\n  !include ../fragments/model.dsl\n}\n")
	writeC4Fixture(t, fragment, "!include view.dsl\nmodel {}\n")
	writeC4Fixture(t, view, "views {}\n")
	closure, err := validateStructurizrClosure(design, entry)
	if err != nil {
		t.Fatal(err)
	}
	if len(closure) != 3 || closure[0] != entry || closure[1] != fragment || closure[2] != view {
		t.Fatalf("local include closure = %v", closure)
	}
}

func TestStructurizrClosureRejectsOversizedSparseInput(t *testing.T) {
	design := t.TempDir()
	entry := filepath.Join(design, "workspace.dsl")
	file, err := os.Create(entry)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(c4ClosureFileMaxBytes + 1); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := validateStructurizrClosure(design, entry); err == nil || !strings.Contains(err.Error(), "byte limit") {
		t.Fatalf("Structurizr closure accepted oversized sparse input: %v", err)
	}
}

func TestStructurizrClosureRejectsExcessiveIncludeDepth(t *testing.T) {
	design := t.TempDir()
	for index := 0; index <= c4ClosureMaxDepth; index++ {
		body := "workspace {}\n"
		if index < c4ClosureMaxDepth {
			body = fmt.Sprintf("!include %03d.dsl\n", index+1)
		}
		writeC4Fixture(t, filepath.Join(design, fmt.Sprintf("%03d.dsl", index)), body)
	}
	if _, err := validateStructurizrClosure(design, filepath.Join(design, "000.dsl")); err == nil || !strings.Contains(err.Error(), "nesting") {
		t.Fatalf("Structurizr closure accepted excessive include depth: %v", err)
	}
}

func TestStructurizrClosureRejectsRemoteDynamicAndEscapingInputs(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"remote include case and whitespace", "  !InClUdE    HTTPS://example.invalid/model.dsl\n", "remote structurizr reference"},
		{"remote extends", "workspace extends \"https://example.invalid/base\" {}\n", "remote structurizr reference"},
		{"remote theme", "themes https://example.invalid/theme.json\n", "remote structurizr reference"},
		{"remote image", "image HTTP://example.invalid/image.png\n", "remote structurizr reference"},
		{"script", "!SCRIPT scripts/model.groovy\n", "!script"},
		{"plugin", "!plugin example.Plugin\n", "!plugin"},
		{"components", "!components src example.Scanner\n", "!components"},
		{"custom implied relationships", "!impliedRelationships example.Strategy\n", "custom !impliedRelationships"},
		{"docs importer", "!docs docs example.DocumentationImporter\n", "custom DocumentationImporter"},
		{"adrs importer", "!adrs adrs example.DecisionImporter\n", "custom importer"},
		{"environment substitution", "!include ${REMOTE_MODEL}\n", "substitution"},
		{"split variable URL", "!const SCHEME https\n!include ${SCHEME}://example.invalid/model.dsl\n", "!const"},
		{"local theme", "theme theme.json\n", "unbound local data"},
		{"local icon", "icon icons/user.png\n", "unbound local data"},
		{"local image", "image diagram.png\n", "unbound local data"},
		{"local plantuml", "plantuml diagram.puml\n", "unbound local data"},
		{"local mermaid", "mermaid diagram.mmd\n", "unbound local data"},
		{"local kroki", "kroki diagram.puml\n", "unbound local data"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			design := t.TempDir()
			entry := filepath.Join(design, "workspace.dsl")
			writeC4Fixture(t, entry, test.body)
			if _, err := validateStructurizrClosure(design, entry); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("unsafe DSL was accepted or wrong diagnostic: %v", err)
			}
		})
	}
}

func TestStructurizrClosureRejectsRemoteReferenceThroughLocalIndirection(t *testing.T) {
	design := t.TempDir()
	entry := filepath.Join(design, "workspace.dsl")
	writeC4Fixture(t, entry, "!include fragment.dsl\n")
	writeC4Fixture(t, filepath.Join(design, "fragment.dsl"), "themes https://example.invalid/theme.json\n")
	if _, err := validateStructurizrClosure(design, entry); err == nil || !strings.Contains(err.Error(), "remote structurizr reference") {
		t.Fatalf("remote indirection was accepted: %v", err)
	}
}

func TestStructurizrClosureRejectsSiblingRepositoryEscape(t *testing.T) {
	workspace := t.TempDir()
	design := filepath.Join(workspace, "project", "design")
	entry := filepath.Join(design, "workspace.dsl")
	writeC4Fixture(t, filepath.Join(workspace, "sibling", "model.dsl"), "model {}\n")
	writeC4Fixture(t, entry, "!include ../../sibling/model.dsl\n")
	if _, err := validateStructurizrClosure(design, entry); err == nil || !strings.Contains(err.Error(), "escapes the retained design workspace") {
		t.Fatalf("sibling repository include was accepted: %v", err)
	}
}

func TestVerifyC4UnsafePreflightStartsNoTool(t *testing.T) {
	design := t.TempDir()
	writeC4Fixture(t, filepath.Join(design, "workspace.dsl"), "!script model.groovy\n")
	started := filepath.Join(t.TempDir(), "started")
	launcher := filepath.Join(t.TempDir(), "structurizr-cli")
	writeC4Fixture(t, launcher, "#!/bin/sh\ntouch \""+started+"\"\n")
	if err := os.Chmod(launcher, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(structurizrEnv, launcher)
	_, stderr, codes := withCapturedIO(t)
	cmd := newVerifyC4Cmd()
	cmd.SetArgs([]string{design})
	if err := executeCapturedCommand(cmd); err != nil {
		t.Fatal(err)
	}
	if len(*codes) != 1 || (*codes)[0] != 1 || !strings.Contains(stderr.String(), "deterministic DSL preflight") {
		t.Fatalf("unsafe preflight did not block: codes=%v stderr=%s", *codes, stderr)
	}
	if _, err := os.Lstat(started); !os.IsNotExist(err) {
		t.Fatalf("Structurizr started before deterministic preflight: %v", err)
	}
}

func TestStructurizrClosureBindsConcurrentLocalIncludeMutation(t *testing.T) {
	project := t.TempDir()
	design := filepath.Join(project, "design")
	entry := filepath.Join(design, "workspace.dsl")
	fragment := filepath.Join(project, "fragment.dsl")
	writeC4Fixture(t, entry, "!include ../fragment.dsl\n")
	writeC4Fixture(t, fragment, "model {}\n")
	prior := designReaderAfterSnapshot
	designReaderAfterSnapshot = func() { writeC4Fixture(t, fragment, "model { changed = true }\n") }
	defer func() { designReaderAfterSnapshot = prior }()
	err := withDesignWorkspaceSnapshot(design, func(snapshot string) error {
		_, err := validateStructurizrClosure(snapshot, filepath.Join(snapshot, "workspace.dsl"))
		return err
	})
	if err == nil || !strings.Contains(err.Error(), "external tree changed") {
		t.Fatalf("concurrent local include mutation was not blocking: %v", err)
	}
}

func TestVerifyStructurizrVersionRejectsSuccessfulWarningNoise(t *testing.T) {
	launcher := filepath.Join(t.TempDir(), "structurizr-cli")
	writeC4Fixture(t, launcher, "#!/bin/sh\nprintf 'deprecated option\\nstructurizr-cli: 2025.11.09\\n'\n")
	if err := os.Chmod(launcher, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := verifyStructurizrVersion(context.Background(), launcher, "2025.11.09", nil); err == nil || !strings.Contains(err.Error(), "no warnings or other output") {
		t.Fatalf("warning-bearing version probe passed: %v", err)
	}
}

func TestVerifyStructurizrVersionAcceptsOnlyOfficialFourLineIdentity(t *testing.T) {
	canonical := "structurizr-cli: 2025.11.09\nstructurizr-java: 5.0.2\nJava: 21.0.12.1/Eclipse Adoptium (/fixture/java-home)\nOS: Mac OS X 26.7 (aarch64)\n"
	if err := validateStructurizrVersionOutput(canonical, "2025.11.09"); err != nil {
		t.Fatalf("official identity rejected: %v", err)
	}
	for _, bad := range []string{
		canonical + "WARNING deprecated\n",
		strings.Replace(canonical, "structurizr-java: 5.0.2", "structurizr-java: 5.0.3", 1),
		strings.Replace(canonical, "/fixture/java-home", "relative", 1),
		strings.Replace(canonical, "OS: Mac OS X 26.7 (aarch64)", "OS: ", 1),
	} {
		if err := validateStructurizrVersionOutput(bad, "2025.11.09"); err == nil {
			t.Fatalf("noncanonical identity accepted: %q", bad)
		}
	}
}

func TestVerifyC4RejectsSuccessfulExporterWarning(t *testing.T) {
	setSupportedJava(t)
	design := t.TempDir()
	writeC4Fixture(t, filepath.Join(design, "workspace.dsl"), "workspace \"W\" \"d\" {}\n")
	launcher := filepath.Join(t.TempDir(), "structurizr-cli")
	writeC4Fixture(t, launcher, "#!/bin/sh\n"+fakeStructurizrVersionBranch+"printf 'graph TD\\n' > \"$7/view.mmd\"\n"+fakeStructurizrExportProgress+"echo 'deprecated export feature'\nexit 0\n")
	if err := os.Chmod(launcher, 0o755); err != nil {
		t.Fatal(err)
	}
	setStructurizrOverride(t, launcher)
	_, stderr, codes := withCapturedIO(t)
	cmd := newVerifyC4Cmd()
	cmd.SetArgs([]string{design})
	if err := executeCapturedCommand(cmd); err != nil {
		t.Fatal(err)
	}
	if len(*codes) != 1 || (*codes)[0] != 1 || !strings.Contains(stderr.String(), "warnings are errors") || !strings.Contains(stderr.String(), "deprecated export feature") {
		t.Fatalf("successful exporter warning was hidden: codes=%v stderr=%s", *codes, stderr)
	}
}

func TestVerifyC4RejectsEmptyAndNonFileExportInventory(t *testing.T) {
	setSupportedJava(t)
	for _, tc := range []struct {
		name, export string
		want         string
	}{
		{"empty", "exit 0", "produced no Mermaid view files"},
		{"directory masquerading as view", `mkdir "$7/view.mmd"`, "regular non-symlink file"},
		{"symlink masquerading as view", `ln -s "$3" "$7/view.mmd"`, "regular non-symlink file"},
		{"unexpected extension", `printf x > "$7/view.svg"`, "permits only .mmd"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design := t.TempDir()
			writeC4Fixture(t, filepath.Join(design, "workspace.dsl"), "workspace \"W\" \"d\" {}\n")
			launcher := filepath.Join(t.TempDir(), "structurizr-cli")
			writeC4Fixture(t, launcher, "#!/bin/sh\n"+fakeStructurizrVersionBranch+tc.export+"\n")
			if err := os.Chmod(launcher, 0o755); err != nil {
				t.Fatal(err)
			}
			setStructurizrOverride(t, launcher)
			cmd := newVerifyC4Cmd()
			cmd.SetArgs([]string{design})
			if err := executeCapturedCommand(cmd); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("invalid successful export inventory passed: %v", err)
			}
		})
	}
}

func TestValidateC4ExportInventoryIsPortableClosedAndStable(t *testing.T) {
	dir := t.TempDir()
	writeC4Fixture(t, filepath.Join(dir, "System.mmd"), "graph TD\n")
	names, err := validateC4ExportInventory(dir)
	if err != nil || strings.Join(names, ",") != "System.mmd" {
		t.Fatalf("valid export inventory = %v, err=%v", names, err)
	}
	writeC4Fixture(t, filepath.Join(dir, "system.mmd"), "graph TD\n")
	upper, upperErr := os.Lstat(filepath.Join(dir, "System.mmd"))
	lower, lowerErr := os.Lstat(filepath.Join(dir, "system.mmd"))
	if upperErr == nil && lowerErr == nil && os.SameFile(upper, lower) {
		t.Skip("host filesystem cannot materialize case-aliased filenames")
	}
	if _, err := validateC4ExportInventory(dir); err == nil || !strings.Contains(err.Error(), "alias on case-insensitive") {
		t.Fatalf("case-aliased export inventory passed: %v", err)
	}
}

func TestReserveC4ExportBytesRejectsInvalidMetadataBeforeAllocation(t *testing.T) {
	for _, tc := range []struct {
		name        string
		size, total int64
		want        string
	}{
		{name: "negative file", size: -1, want: "negative size"},
		{name: "oversized file", size: structurizrExportMaxFileBytes + 1, want: "per-file bound"},
		{name: "negative accumulated", size: 0, total: -1, want: "invalid accumulated size"},
		{name: "oversized accumulated", size: 0, total: structurizrTreeMaxBytes + 1, want: "invalid accumulated size"},
		{name: "remaining aggregate", size: 2, total: structurizrTreeMaxBytes - 1, want: "remaining inventory bound"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := reserveC4ExportBytes("System.mmd", tc.size, tc.total); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("reserve size=%d total=%d error = %v", tc.size, tc.total, err)
			}
		})
	}
	if got, err := reserveC4ExportBytes("System.mmd", structurizrExportMaxFileBytes, structurizrTreeMaxBytes-structurizrExportMaxFileBytes); err != nil || got != structurizrTreeMaxBytes {
		t.Fatalf("exact remaining budget = %d, %v", got, err)
	}
}

func TestValidateC4ExportInventoryRejectsSparseOversizeBeforeRead(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files int
		size  int64
		want  string
	}{
		{name: "per-file", files: 1, size: structurizrExportMaxFileBytes + 1, want: "per-file bound"},
		{name: "aggregate", files: int(structurizrTreeMaxBytes/structurizrExportMaxFileBytes) + 1, size: structurizrExportMaxFileBytes, want: "remaining inventory bound"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for i := 0; i < tc.files; i++ {
				path := filepath.Join(dir, fmt.Sprintf("View-%03d.mmd", i))
				file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
				if err != nil {
					t.Fatal(err)
				}
				if err := file.Truncate(tc.size); err != nil {
					_ = file.Close()
					t.Fatal(err)
				}
				if err := file.Close(); err != nil {
					t.Fatal(err)
				}
			}
			read := false
			prior := verifyC4AfterExportRead
			verifyC4AfterExportRead = func(string) { read = true }
			t.Cleanup(func() { verifyC4AfterExportRead = prior })
			if _, err := validateC4ExportInventory(dir); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("sparse oversized export error = %v", err)
			}
			if read {
				t.Fatal("oversized sparse export reached content-read validation")
			}
		})
	}
}

func TestValidateC4ExportInventoryRejectsSameSizeContentRace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "System.mmd")
	writeC4Fixture(t, path, "graph AA\n")
	prior := verifyC4AfterExportRead
	verifyC4AfterExportRead = func(name string) {
		if name == "System.mmd" {
			writeC4Fixture(t, path, "graph BB\n")
		}
	}
	defer func() { verifyC4AfterExportRead = prior }()
	if _, err := validateC4ExportInventory(dir); err == nil || !strings.Contains(err.Error(), "changed content") {
		t.Fatalf("same-size in-place mutation passed: %v", err)
	}
}

func TestValidateC4ExportInventoryRejectsEntryAddedAfterInitialInventory(t *testing.T) {
	dir := t.TempDir()
	writeC4Fixture(t, filepath.Join(dir, "System.mmd"), "graph TD\n")
	prior := verifyC4AfterExportInventory
	verifyC4AfterExportInventory = func() {
		writeC4Fixture(t, filepath.Join(dir, "Late.mmd"), "graph LR\n")
	}
	defer func() { verifyC4AfterExportInventory = prior }()
	if _, err := validateC4ExportInventory(dir); err == nil || !strings.Contains(err.Error(), "inventory changed during validation") {
		t.Fatalf("entry added after initial inventory passed: %v", err)
	}
}

func TestValidateC4ExportInventoryReconcilesFinalIdentityAndNames(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(t *testing.T, dir, path string)
	}{
		{
			name: "replace validated file",
			mutate: func(t *testing.T, dir, path string) {
				replacement := filepath.Join(dir, "replacement")
				writeC4Fixture(t, replacement, "graph LR\n")
				if err := os.Rename(replacement, path); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "replace inventory name",
			mutate: func(t *testing.T, dir, path string) {
				if err := os.Rename(path, filepath.Join(dir, "Other.mmd")); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "System.mmd")
			writeC4Fixture(t, path, "graph TD\n")
			prior := verifyC4BeforeFinalExportInventory
			verifyC4BeforeFinalExportInventory = func() { tc.mutate(t, dir, path) }
			defer func() { verifyC4BeforeFinalExportInventory = prior }()
			if _, err := validateC4ExportInventory(dir); err == nil || !strings.Contains(err.Error(), "inventory") && !strings.Contains(err.Error(), "changed after validation") {
				t.Fatalf("final export mutation passed: %v", err)
			}
		})
	}
}

func TestStructurizrExportProgressMustMatchExactInventory(t *testing.T) {
	dir := t.TempDir()
	view := filepath.Join(dir, "System.mmd")
	writeC4Fixture(t, view, "graph TD\n")
	dsl := filepath.Join(t.TempDir(), "workspace.dsl")
	writeC4Fixture(t, dsl, "workspace {}\n")
	canonical := "Exporting workspace from " + dsl + "\n - exporting with MermaidDiagramExporter\n - writing " + view + "\n - finished\n"
	if err := validateStructurizrExportOutput(canonical, dsl, dir, []string{"System.mmd"}); err != nil {
		t.Fatalf("canonical progress rejected: %v", err)
	}
	defaultViews := strings.Replace(canonical, " - exporting with MermaidDiagramExporter\n", " - no views defined; creating default views\n - exporting with MermaidDiagramExporter\n", 1)
	if err := validateStructurizrExportOutput(defaultViews, dsl, dir, []string{"System.mmd"}); err != nil {
		t.Fatalf("canonical default-view progress rejected: %v", err)
	}
	for _, bad := range []string{
		strings.Replace(canonical, "System.mmd", "Other.mmd", 1),
		canonical + "WARNING deprecated\n",
		strings.Replace(canonical, " - finished\n", "", 1),
		strings.Replace(defaultViews, " - no views defined; creating default views\n", " - no views defined; creating default views\n - no views defined; creating default views\n", 1),
	} {
		if err := validateStructurizrExportOutput(bad, dsl, dir, []string{"System.mmd"}); err == nil {
			t.Fatalf("noncanonical progress accepted: %q", bad)
		}
	}
}
