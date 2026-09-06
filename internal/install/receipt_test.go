package install

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
)

func TestReceiptReadModifyWriteIsSerialized(t *testing.T) {
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	root := t.TempDir()
	const count = 16
	for i := 0; i < count; i++ {
		seedHomeArtifactInventory(t, filepath.Join(root, fmt.Sprintf("home-%02d", i)))
	}
	var wg sync.WaitGroup
	errCh := make(chan error, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errCh <- recordHomeInstall([]string{filepath.Join(root, fmt.Sprintf("home-%02d", i))}, false)
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	receipt, _, err := loadReceipt()
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.HomeInstalls) != count {
		t.Fatalf("receipt retained %d/%d concurrent updates", len(receipt.HomeInstalls), count)
	}
}

func TestLoadReceiptRejectsSymlinkOversizeAndUnstableSwap(t *testing.T) {
	t.Run("symlink", func(t *testing.T) {
		config := privateConfigDir(t)
		t.Setenv("MACHINERY_CONFIG_DIR", config)
		outside := filepath.Join(t.TempDir(), "outside.json")
		if err := os.WriteFile(outside, []byte(`{"schema_version":2}`), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(config, "install.json")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		if _, exists, err := loadReceipt(); !exists || err == nil || !strings.Contains(err.Error(), "private regular file") {
			t.Fatalf("symlink receipt: exists=%v err=%v", exists, err)
		}
	})

	t.Run("oversize", func(t *testing.T) {
		config := privateConfigDir(t)
		t.Setenv("MACHINERY_CONFIG_DIR", config)
		if err := os.WriteFile(filepath.Join(config, "install.json"), make([]byte, receiptMaxBytes+1), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, exists, err := loadReceipt(); !exists || err == nil || !strings.Contains(err.Error(), "exceeds") {
			t.Fatalf("oversize receipt: exists=%v err=%v", exists, err)
		}
	})

	t.Run("config directory swap", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("fixture uses POSIX rename-over-open-directory semantics")
		}
		base := t.TempDir()
		config := filepath.Join(base, "config")
		if err := os.Mkdir(config, 0o700); err != nil {
			t.Fatal(err)
		}
		t.Setenv("MACHINERY_CONFIG_DIR", config)
		writeLegacyReceipt(t, installReceipt{})
		outside := t.TempDir()
		sentinel := filepath.Join(outside, "sentinel")
		write(t, sentinel, "outside")
		parked := filepath.Join(base, "config-parked")
		afterReceiptRootOpen = func() {
			afterReceiptRootOpen = nil
			if err := os.Rename(config, parked); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, config); err != nil {
				t.Fatal(err)
			}
		}
		t.Cleanup(func() {
			afterReceiptRootOpen = nil
			_ = os.Remove(config)
			_ = os.Rename(parked, config)
		})
		if _, exists, err := loadReceipt(); !exists || err == nil || !strings.Contains(err.Error(), "changed during read") {
			t.Fatalf("directory swap receipt: exists=%v err=%v", exists, err)
		}
		if got, err := os.ReadFile(sentinel); err != nil || string(got) != "outside" {
			t.Fatalf("outside sentinel changed: %q, %v", got, err)
		}
	})

	t.Run("entry swap", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("fixture uses POSIX rename-over-open-file semantics")
		}
		config := privateConfigDir(t)
		t.Setenv("MACHINERY_CONFIG_DIR", config)
		writeLegacyReceipt(t, installReceipt{})
		path := filepath.Join(config, "install.json")
		parked := path + ".parked"
		outside := filepath.Join(t.TempDir(), "outside.json")
		if err := os.WriteFile(outside, []byte(`{"schema_version":2}`), 0o600); err != nil {
			t.Fatal(err)
		}
		afterReceiptEntryLstat = func() {
			afterReceiptEntryLstat = nil
			if err := os.Rename(path, parked); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, path); err != nil {
				t.Fatal(err)
			}
		}
		t.Cleanup(func() {
			afterReceiptEntryLstat = nil
			_ = os.Remove(path)
			_ = os.Rename(parked, path)
		})
		if _, exists, err := loadReceipt(); !exists || err == nil {
			t.Fatalf("entry swap receipt: exists=%v err=%v", exists, err)
		}
		if got, err := os.ReadFile(outside); err != nil || string(got) != `{"schema_version":2}` {
			t.Fatalf("outside receipt changed: %q, %v", got, err)
		}
	})
}

func TestInstallRecordsCustomHomesAndNativeTargets(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME override does not steer os.UserHomeDir on Windows")
	}
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	home := t.TempDir()
	t.Setenv("HOME", home)
	src := fakeSource(t)
	customA := filepath.Join(home, "custom-a")
	customB := filepath.Join(home, "custom-b")

	if err := Install(Options{Homes: []string{customA, customB}, From: src, Record: true}); err != nil {
		t.Fatal(err)
	}
	if err := Install(Options{Targets: []string{"codex", "opencode"}, From: src, Copy: true, Record: true}); err != nil {
		t.Fatal(err)
	}
	receipt, exists, err := loadReceipt()
	if err != nil {
		t.Fatal(err)
	}
	if !exists || receipt.SchemaVersion != receiptSchema {
		t.Fatalf("receipt = %+v, exists=%v", receipt, exists)
	}
	if len(receipt.HomeInstalls) != 1 || len(receipt.HomeInstalls[0].Homes) != 2 || receipt.HomeInstalls[0].Copy {
		t.Fatalf("home receipt = %+v", receipt.HomeInstalls)
	}
	if len(receipt.Targets) != 2 || !receipt.Targets[0].Copy || !receipt.Targets[1].Copy {
		t.Fatalf("target receipt = %+v", receipt.Targets)
	}

	if err := ForgetTargetInstalls([]string{"opencode"}); err != nil {
		t.Fatal(err)
	}
	if err := ForgetHomeInstalls([]string{customB}); err != nil {
		t.Fatal(err)
	}
	receipt, _, err = loadReceipt()
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.Targets) != 1 || receipt.Targets[0].Target != "codex" {
		t.Fatalf("target receipt after removal = %+v", receipt.Targets)
	}
	if len(receipt.HomeInstalls) != 1 || len(receipt.HomeInstalls[0].Homes) != 1 || receipt.HomeInstalls[0].Homes[0] != customA {
		t.Fatalf("home receipt after secondary removal = %+v", receipt.HomeInstalls)
	}
}

func TestForgetReceiptUsesNativeCaseAliasIdentity(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("case-folded path identity is a Darwin/Windows contract")
	}
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	root := t.TempDir()
	recorded := filepath.Join(root, "Recorded", "Home")
	seedHomeArtifactInventory(t, recorded)
	if err := recordHomeInstall([]string{recorded}, false); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "recorded", "home")
	if err := ForgetHomeInstalls([]string{alias}); err != nil {
		t.Fatal(err)
	}
	_, exists, err := loadReceipt()
	if err != nil || exists {
		t.Fatalf("case-alias receipt was not removed: exists=%v err=%v", exists, err)
	}
}

func seedHomeArtifactInventory(t *testing.T, home string) {
	t.Helper()
	write(t, filepath.Join(home, "skills", "machinery", "SKILL.md"), "seed skill")
	for _, doc := range RoleDocs {
		write(t, filepath.Join(home, "agents", doc), "seed role")
	}
}

func writeLegacyReceipt(t *testing.T, receipt installReceipt) {
	t.Helper()
	receipt.SchemaVersion = 1
	receipt.Artifacts = nil
	path, err := installationReceiptPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestBuildRefreshPlanMigratesLegacyStandardTopology(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink topology test is POSIX-specific")
	}
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	home := t.TempDir()
	t.Setenv("HOME", home)
	agentsSkill := filepath.Join(home, ".agents", "skills", "machinery")
	if err := os.MkdirAll(agentsSkill, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(agentsSkill, "SKILL.md"), "skill")
	claudeSkills := filepath.Join(home, ".claude", "skills")
	if err := os.MkdirAll(claudeSkills, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(agentsSkill, filepath.Join(claudeSkills, "machinery")); err != nil {
		t.Fatal(err)
	}

	plan, err := buildRefreshPlan()
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.HomeInstalls) != 1 || plan.HomeInstalls[0].Copy || len(plan.HomeInstalls[0].Homes) != 2 {
		t.Fatalf("legacy plan = %+v", plan.HomeInstalls)
	}
}

func TestBuildRefreshPlanRecognizesNativeTargetTopology(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME override does not steer os.UserHomeDir on Windows")
	}
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	home := t.TempDir()
	t.Setenv("HOME", home)
	write(t, filepath.Join(home, ".agents", "skills", "machinery", "SKILL.md"), "skill")
	write(t, filepath.Join(home, ".codex", "agents", "machinery-fsm-author.toml"), "agent")

	plan, err := buildRefreshPlan()
	if err != nil {
		t.Fatal(err)
	}
	if !plan.hasTarget(TargetCodex) {
		t.Fatalf("Codex target not discovered: %+v", plan.Targets)
	}
	if len(plan.HomeInstalls) != 0 {
		t.Fatalf("shared target skill must not be misclassified as a legacy home: %+v", plan.HomeInstalls)
	}
}

func TestCorruptReceiptFailsLoudly(t *testing.T) {
	err := assertPrivateReceiptRejected(t, `{"schema_version":1}`, `{not-json`, "parse", "invalid character")
	var syntax *json.SyntaxError
	if !errors.As(err, &syntax) {
		t.Fatalf("malformed JSON did not reach syntax validation: %v", err)
	}
}

func TestReceiptRejectsUnknownDuplicateAndWrongTypedTopology(t *testing.T) {
	home, err := json.Marshal(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, valid, raw, diagnostic, typeField, typeName string
	}{
		{"unknown root", `{"schema_version":1}`, `{"schema_version":1,"home_installz":[]}`, `unknown field "home_installz"`, "", ""},
		{"unknown home field", fmt.Sprintf(`{"schema_version":1,"home_installs":[{"homes":[%s]}]}`, home), fmt.Sprintf(`{"schema_version":1,"home_installs":[{"homes":[%s],"copies":true}]}`, home), `unknown field "copies"`, "", ""},
		{"unknown target field", `{"schema_version":1,"targets":[{"target":"codex"}]}`, `{"schema_version":1,"targets":[{"target":"codex","copied":true}]}`, `unknown field "copied"`, "", ""},
		{"duplicate root", `{"schema_version":1}`, `{"schema_version":1,"schema_version":1}`, `duplicate JSON field "schema_version"`, "", ""},
		{"duplicate nested", `{"schema_version":1,"targets":[{"target":"codex"}]}`, `{"schema_version":1,"targets":[{"target":"codex","target":"opencode"}]}`, `duplicate JSON field "target"`, "", ""},
		{"wrong homes type", fmt.Sprintf(`{"schema_version":1,"home_installs":[{"homes":[%s]}]}`, home), fmt.Sprintf(`{"schema_version":1,"home_installs":[{"homes":%s}]}`, home), "cannot unmarshal string", "homes", "[]string"},
		{"wrong copy type", `{"schema_version":1,"targets":[{"target":"codex","copy":true}]}`, `{"schema_version":1,"targets":[{"target":"codex","copy":"yes"}]}`, "cannot unmarshal string", "copy", "bool"},
		{"trailing value", `{"schema_version":1}`, `{"schema_version":1} {"schema_version":1}`, "trailing JSON value", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := assertPrivateReceiptRejected(t, tc.valid, tc.raw, "parse", tc.diagnostic)
			if tc.typeField != "" {
				var typed *json.UnmarshalTypeError
				if !errors.As(err, &typed) || typed.Value != "string" || !strings.HasSuffix(typed.Field, "."+tc.typeField) || typed.Type.String() != tc.typeName {
					t.Fatalf("wrong typed-field diagnosis: %v", err)
				}
			}
		})
	}
}

func TestSemanticallyInvalidReceiptFailsBeforeUpdate(t *testing.T) {
	// This is loader rejection before planning, not an observed binary swap.
	assertPrivateReceiptRejected(t, `{"schema_version":1,"targets":[{"target":"codex"}]}`, `{"schema_version":1,"targets":[{"target":"cursor"}]}`, "invalid", `unknown target "cursor"`)
}

func readPrivateReceiptFixture(t *testing.T, raw string) (installReceipt, bool, error) {
	t.Helper()
	path, err := installationReceiptPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	dir, err := os.Lstat(filepath.Dir(path))
	if err != nil || !dir.IsDir() || dir.Mode().Perm() != 0o700 {
		t.Fatalf("receipt fixture config is not an actual private directory: %v", err)
	}
	file, err := os.Lstat(path)
	if err != nil || !file.Mode().IsRegular() || file.Mode().Perm() != 0o600 {
		t.Fatalf("receipt fixture is not an actual 0600 regular file: %v", err)
	}
	return loadReceipt()
}

func assertPrivateReceiptRejected(t *testing.T, valid, invalid, category string, details ...string) error {
	t.Helper()
	t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
	if _, exists, err := readPrivateReceiptFixture(t, valid); err != nil || !exists {
		t.Fatalf("matched private valid receipt was not accepted: exists=%v err=%v", exists, err)
	}
	_, exists, err := readPrivateReceiptFixture(t, invalid)
	if !exists || err == nil {
		t.Fatalf("intended invalid receipt accepted/missing: exists=%v err=%v", exists, err)
	}
	path, pathErr := installationReceiptPath()
	if pathErr != nil {
		t.Fatal(pathErr)
	}
	// Strip only the exact known wrapper: directory/test names cannot satisfy details.
	prefix := category + " installation receipt " + path + ": "
	if !strings.HasPrefix(err.Error(), prefix) {
		t.Fatalf("did not reach intended %s validation: %v", category, err)
	}
	diagnostic := strings.TrimPrefix(err.Error(), prefix)
	for _, detail := range details {
		if !strings.Contains(diagnostic, detail) {
			t.Fatalf("intended diagnostic %q missing from %q", detail, diagnostic)
		}
	}
	t.Logf("reached %s validation: %s", category, diagnostic)
	return err
}

func realSchemaTwoReceiptFixture(t *testing.T) installReceipt {
	t.Helper()
	root := t.TempDir()
	receipt := installReceipt{SchemaVersion: 2, HostPlugins: []string{"claude", "codex"}}
	for i, name := range []string{"home-a", "home-b"} {
		home := filepath.Join(root, name)
		seedHomeArtifactInventory(t, home)
		receipt.HomeInstalls = append(receipt.HomeInstalls, homeInstall{Homes: []string{home}, Copy: i == 1})
		// Independently enumerate all three shipped artifact roots per home.
		for _, relative := range []string{"skills/machinery", "agents/machinery-fsm-author.md", "agents/machinery-build-writer.md"} {
			path := filepath.Join(home, filepath.FromSlash(relative))
			digest, err := artifactTreeDigest(path)
			if err != nil {
				t.Fatal(err)
			}
			receipt.Artifacts = append(receipt.Artifacts, receiptArtifact{Path: path, Digest: digest})
		}
	}
	normalizeReceipt(&receipt)
	return receipt
}

func TestReceiptPrivateSchemaControls(t *testing.T) {
	for _, schema := range []int{1, 2} {
		t.Run(fmt.Sprintf("schema_%d", schema), func(t *testing.T) {
			t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
			want := realSchemaTwoReceiptFixture(t)
			want.SchemaVersion = schema
			if schema == 1 {
				want.HostPlugins, want.Artifacts = nil, nil
			}
			raw, err := json.Marshal(want)
			if err != nil {
				t.Fatal(err)
			}
			got, exists, err := readPrivateReceiptFixture(t, string(raw))
			encoded, encodeErr := json.Marshal(got)
			if err != nil || !exists || encodeErr != nil || string(encoded) != string(raw) {
				t.Fatalf("valid private schema%d topology/inventory changed: %v %v", schema, err, encodeErr)
			}
		})
	}
}

func TestReceiptSchemaTwoInventoryValidation(t *testing.T) {
	want := realSchemaTwoReceiptFixture(t)
	valid, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, diagnostic string
		mutate           func(*installReceipt)
	}{
		{"missing entry", "artifact inventory has 5 entries, want 6", func(r *installReceipt) { r.Artifacts = r.Artifacts[:5] }},
		{"extra entry", "artifact inventory has 7 entries, want 6", func(r *installReceipt) {
			r.Artifacts = append(r.Artifacts, receiptArtifact{Path: filepath.Join(t.TempDir(), "extra"), Digest: r.Artifacts[0].Digest})
		}},
		{"duplicate path", "artifact inventory path", func(r *installReceipt) { r.Artifacts[1].Path = r.Artifacts[0].Path }},
		{"substituted path", "artifact inventory path", func(r *installReceipt) {
			r.Artifacts[0].Path = filepath.Join(r.HomeInstalls[0].Homes[0], "agents", "substitute.md")
		}},
		{"relative path", "artifact inventory path", func(r *installReceipt) { r.Artifacts[0].Path = "relative/artifact" }},
		{"non-clean path", "artifact inventory path", func(r *installReceipt) {
			r.Artifacts[0].Path = filepath.Dir(r.Artifacts[0].Path) + string(os.PathSeparator) + "." + string(os.PathSeparator) + filepath.Base(r.Artifacts[0].Path)
		}},
		{"unexpected absolute path", "artifact inventory path", func(r *installReceipt) { r.Artifacts[0].Path = filepath.Join(t.TempDir(), "outside") }},
		{"digest prefix", "artifact inventory digest for", func(r *installReceipt) {
			r.Artifacts[0].Digest = "sha512:" + strings.TrimPrefix(r.Artifacts[0].Digest, "sha256:")
		}},
		{"digest length", "artifact inventory digest for", func(r *installReceipt) { r.Artifacts[0].Digest = r.Artifacts[0].Digest[:len(r.Artifacts[0].Digest)-1] }},
		{"digest nonhex", "artifact inventory digest for", func(r *installReceipt) { r.Artifacts[0].Digest = "sha256:" + strings.Repeat("g", 64) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var invalid installReceipt
			if err := json.Unmarshal(valid, &invalid); err != nil {
				t.Fatal(err)
			}
			tc.mutate(&invalid)
			raw, err := json.Marshal(invalid)
			if err != nil {
				t.Fatal(err)
			}
			details := []string{tc.diagnostic}
			if strings.Contains(tc.name, "path") {
				details = append(details, "does not match recorded topology path")
			}
			if strings.HasPrefix(tc.name, "digest") {
				details = append(details, "is malformed", invalid.Artifacts[0].Path)
			}
			assertPrivateReceiptRejected(t, string(valid), string(raw), "invalid", details...)
		})
	}
	t.Run("valid reversed ordering", func(t *testing.T) {
		t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
		slices.Reverse(want.HomeInstalls)
		slices.Reverse(want.HostPlugins)
		slices.Reverse(want.Artifacts)
		raw, err := json.Marshal(want)
		if err != nil {
			t.Fatal(err)
		}
		got, exists, err := readPrivateReceiptFixture(t, string(raw))
		encoded, encodeErr := json.Marshal(got)
		if err != nil || !exists || encodeErr != nil || string(encoded) != string(valid) {
			t.Fatalf("valid reversed inventory/topology not normalized: %v %v", err, encodeErr)
		}
	})
}
