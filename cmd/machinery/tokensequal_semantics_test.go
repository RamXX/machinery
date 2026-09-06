package main

import (
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// AC3: both inputs are valid source. Their evaluated results differ even though
// whitespace tokenization discards the only difference between the inputs.
func TestTokensEqualSemanticCounterexamples(t *testing.T) {
	cases := []struct {
		name, before, after   string
		meaning               func(*testing.T, string) int64
		wantBefore, wantAfter int64
	}{
		{
			name:   "quoted_literal_spacing",
			before: `len("pay  now")`, after: `len("pay now")`,
			meaning: func(t *testing.T, source string) int64 {
				t.Helper()
				value, err := types.Eval(token.NewFileSet(), nil, token.NoPos, source)
				if err != nil {
					t.Fatalf("evaluate valid Go expression %q: %v", source, err)
				}
				result, exact := constant.Int64Val(value.Value)
				if !exact {
					t.Fatalf("expression did not evaluate to an exact integer: %v", value)
				}
				return result
			},
			wantBefore: 8, wantAfter: 7,
		},
		{
			name:   "indentation_changes_permission_owner",
			before: "user:\n  can_delete: true\n", after: "user:\ncan_delete: true\n",
			meaning: func(t *testing.T, source string) int64 {
				t.Helper()
				var policy struct {
					User struct {
						CanDelete bool `yaml:"can_delete"`
					} `yaml:"user"`
					CanDelete bool `yaml:"can_delete"`
				}
				if err := yaml.Unmarshal([]byte(source), &policy); err != nil {
					t.Fatalf("parse valid indentation-sensitive YAML: %v", err)
				}
				// Pin the complete permission ownership distinction, not a parse error.
				if policy.User.CanDelete == policy.CanDelete {
					t.Fatalf("expected exactly one permission owner, got %+v", policy)
				}
				if policy.User.CanDelete {
					return 1
				}
				return 0
			},
			wantBefore: 1, wantAfter: 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("executable_semantic_control", func(t *testing.T) {
				before, after := tc.meaning(t, tc.before), tc.meaning(t, tc.after)
				if before != tc.wantBefore || after != tc.wantAfter || before == after {
					t.Fatalf("semantic outcomes = %d -> %d, want %d -> %d", before, after, tc.wantBefore, tc.wantAfter)
				}
				if !slices.Equal(strings.Fields(tc.before), strings.Fields(tc.after)) {
					t.Fatal("counterexample must differ only in whitespace discarded by tokenization")
				}
			})
			oldPath, newPath := writeTokensEqualPair(t, tc.before, tc.after)
			out, errOut, code := runBin(t, "tokens-equal", oldPath, newPath)
			t.Run("whitespace_token_utility", func(t *testing.T) {
				assertTokensEqualSuccess(t, out, errOut, code, len(strings.Fields(tc.before)))
			})
			t.Run("no_semantic_or_frozen_edit_assurance", func(t *testing.T) {
				assertTokensEqualHonesty(t, out+errOut)
			})
		})
	}
}

// AC2/AC5: run the actual locally built CLI, with the shared harness's private
// configuration; these are public command paths, without a Paivot dependency.
func TestTokensEqualHelpDescribesOnlyWhitespaceTokens(t *testing.T) {
	for _, args := range [][]string{{"tokens-equal", "--help"}, {"--help"}} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			out, errOut, code := runBin(t, args...)
			if code != 0 || errOut != "" {
				t.Fatalf("help exit=%d stderr=%q", code, errOut)
			}
			text := normalizeTokensEqualProse(out)
			if !strings.Contains(text, "whitespace") || !strings.Contains(text, "token") {
				t.Errorf("help must describe whitespace-token comparison: %q", out)
			}
			assertTokensEqualHonesty(t, out)
		})
	}
}

// AC4: utility compatibility stays independently green at RED. Honest output
// is tested separately so a useful comparison is never confused with approval.
func TestTokensEqualRealCLIUtilityControls(t *testing.T) {
	for _, tc := range []struct {
		name, before, after string
		code                int
		prefix              string
	}{
		{"whitespace_reflow", "one two\nthree four\n", "\t one\ntwo  three\r\nfour ", 0, "token-identical: 4 tokens"},
		{"empty_whitespace", "", " \t\r\n", 0, "token-identical: 0 tokens"},
		{"token_value_changed", "one two", "one six", 1, "NOT token-identical:"},
		{"token_added", "one two", "one two three", 1, "NOT token-identical:"},
		{"token_removed", "one two three", "one two", 1, "NOT token-identical:"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			oldPath, newPath := writeTokensEqualPair(t, tc.before, tc.after)
			out, errOut, code := runBin(t, "tokens-equal", oldPath, newPath)
			if code != tc.code || !strings.HasPrefix(out, tc.prefix) {
				t.Fatalf("exit=%d stdout=%q stderr=%q; want exit=%d prefix=%q", code, out, errOut, tc.code, tc.prefix)
			}
			if tc.code == 0 && errOut != "" {
				t.Fatalf("successful comparison stderr=%q", errOut)
			}
		})
	}
	t.Run("missing_input", func(t *testing.T) {
		oldPath, newPath := writeTokensEqualPair(t, "one two", "one two")
		missingPath := filepath.Join(filepath.Dir(newPath), "absent.txt")
		out, errOut, code := runBin(t, "tokens-equal", oldPath, missingPath)
		if code != 1 || strings.HasPrefix(out, "token-identical:") || !strings.Contains(errOut, "tokens-equal:") || !strings.Contains(errOut, "absent.txt") {
			t.Fatalf("real missing input must fail without equality: exit=%d stdout=%q stderr=%q", code, out, errOut)
		}
	})
}

func TestTokensEqualWhitespaceSuccessDoesNotAuthorizeFrozenEdits(t *testing.T) {
	oldPath, newPath := writeTokensEqualPair(t, "one two\nthree four", " one\ttwo three\nfour\n")
	out, errOut, code := runBin(t, "tokens-equal", oldPath, newPath)
	assertTokensEqualSuccess(t, out, errOut, code, 4)
	assertTokensEqualHonesty(t, out+errOut)
}

// AC1/AC2/AC3/AC5: inspect every actual shipped policy. A missing surface is a
// failure, and the observed affirmative exemption is checked before new wording
// so baseline RED is attributable to real unsafe authorization on each surface.
func TestFrozenGuidanceRequiresExactIdentityAndEvidenceReplay(t *testing.T) {
	paths := []string{
		"skills/machinery/references/build-md-template.md",
		"agents/machinery-build-writer.md",
		"docs/brownfield-team-guide.md",
		"examples/checkout-split/orders/design/BUILD.md",
		"examples/checkout-split/payments/design/BUILD.md",
		"examples/fulfillment/design/BUILD.md",
		"examples/go-crm/design/BUILD.md",
		"examples/portfolio-engine/design/BUILD.md",
		"examples/surreal-crm/design/BUILD.md",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join(repoRootDir(t), filepath.FromSlash(path)))
			if err != nil {
				t.Fatalf("required shipped guidance is unreadable: %v", err)
			}
			text := normalizeTokensEqualProse(string(body))
			unsafe := regexp.MustCompile(`an owner-sanctioned (?:formatting-only amendment (?:carrying|with) a token-identity proof|amendment that changes formatting only and carries a token-identity proof)`)
			if exemption := unsafe.FindString(text); exemption != "" {
				t.Fatalf("shipped guidance still authorizes a frozen RED amendment through token identity: %q", exemption)
			}
			for _, requirement := range []struct{ name, pattern string }{
				{"exact bytes and inventory define frozen identity", `(?:frozen|locked)[^.]{0,180}(?:exact bytes|byte-for-byte|byte identity)[^.]{0,180}inventory`},
				{"amendments require explicit new evidence revision and replay", `(?:amendment|amending|edit|change)[^.]{0,180}(?:require|must|need)[^.]{0,180}explicit[^.]{0,180}new evidence revision[^.]{0,180}replay`},
				{"formatting or token equality grants no editing exemption", `(?:no|neither|never)[^.]{0,180}(?:formatting|token)[^.]{0,180}(?:exempt|authoriz|permit)|(?:formatting|token)[^.]{0,180}(?:does not|do not|never|cannot)[^.]{0,120}(?:authoriz|permit|exempt)`},
			} {
				if !regexp.MustCompile(requirement.pattern).MatchString(text) {
					t.Errorf("shipped frozen-test policy must state: %s", requirement.name)
				}
			}
		})
	}
}

func writeTokensEqualPair(t *testing.T, before, after string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	paths := []string{filepath.Join(dir, "before.txt"), filepath.Join(dir, "after.txt")}
	for i, body := range []string{before, after} {
		if err := os.WriteFile(paths[i], []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return paths[0], paths[1]
}

func assertTokensEqualSuccess(t *testing.T, out, errOut string, code, count int) {
	t.Helper()
	prefix := fmt.Sprintf("token-identical: %d tokens", count)
	if code != 0 || errOut != "" || !strings.HasPrefix(out, prefix) {
		t.Fatalf("whitespace-token equality must remain useful: exit=%d stdout=%q stderr=%q; want %q", code, out, errOut, prefix)
	}
}

func normalizeTokensEqualProse(text string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.NewReplacer("`", "", "**", "").Replace(text))), " ")
}

func assertTokensEqualHonesty(t *testing.T, output string) {
	t.Helper()
	text := normalizeTokensEqualProse(output)
	// Match affirmative assurances, not isolated terminology that an honest
	// explanation may use to say what token equality does NOT establish.
	for _, claim := range []string{
		`(?:^|[.;:] )the change is formatting-only`,
		`(?:^|[.;:] |tokens-equal )prove two files are formatting-only`,
		`(?:^|[.;:] )(?:this |the comparison |token equality )?(?:proves semantic equivalence|preserves program meaning|authorizes (?:a |an )?frozen[- ]test edit)`,
	} {
		if match := regexp.MustCompile(claim).FindString(text); match != "" {
			t.Errorf("whitespace-token equality must not claim semantic preservation, formatting-only proof, or frozen-edit authority: %q", match)
		}
	}
}
