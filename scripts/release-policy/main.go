// Command release-policy verifies, fail closed, that a release tag may be
// published: the exact release commit carries successful required-workflow
// runs, the commit has permitted main ancestry, the tag matches the policy
// version pattern, and (when supplied) the build artifact manifest binds the
// published artifacts to that same commit.
//
// RED PHASE STUB: this file parses the complete command surface but performs
// none of the verification. The test suite in main_test.go defines the
// behavior; the stub exists so those tests compile and fail behaviorally
// (every deny expectation fails, every allow control passes) until the
// verifier is implemented.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

func main() {
	os.Exit(run(os.Args[1:], os.Environ(), os.Stdout, os.Stderr))
}

func run(args, environ []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("release-policy", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", "", "trusted repository root")
	tag := flags.String("tag", "", "release tag to verify")
	policyPath := flags.String("policy", "", "policy JSON path (default <root>/docs/release-policy.json)")
	mainRef := flags.String("main-ref", "", "main ref the release commit must descend from (default from policy)")
	expectSHA := flags.String("expect-sha", "", "commit the tag must resolve to (for example GITHUB_SHA)")
	apiBase := flags.String("api-base", "https://api.github.com", "GitHub API base URL")
	repository := flags.String("repository", "", "owner/name slug; required-check verification runs when set")
	manifest := flags.String("artifact-manifest", "", "build artifact manifest to verify")
	artifactsDir := flags.String("artifacts-dir", "", "directory to re-hash against the artifact manifest")
	timeout := flags.Duration("timeout", 30*time.Second, "per-request API timeout")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *root == "" || *tag == "" {
		fmt.Fprintln(stderr, "release-policy: require -root and -tag")
		return 2
	}
	_ = policyPath
	_ = mainRef
	_ = expectSHA
	_ = apiBase
	_ = repository
	_ = manifest
	_ = artifactsDir
	_ = timeout
	_ = environ
	fmt.Fprintf(stdout, "release-policy: stub (RED phase): no checks implemented for tag %s in %s\n", *tag, *root)
	fmt.Fprintln(stdout, "release-policy: verdict ALLOW")
	return 0
}

type requiredWorkflow struct {
	Path         string   `json:"path"`
	Name         string   `json:"name"`
	Events       []string `json:"events"`
	RequiredJobs []string `json:"required_jobs"`
}

type statusChecksPayload struct {
	Strict   bool     `json:"strict"`
	Contexts []string `json:"contexts"`
}

type requiredStatusChecks struct {
	Endpoint                 string              `json:"endpoint"`
	Method                   string              `json:"method"`
	Payload                  statusChecksPayload `json:"payload"`
	ApplyCommand             string              `json:"apply_command"`
	VerifyCommand            string              `json:"verify_command"`
	PreApplyNameConfirmation string              `json:"pre_apply_name_confirmation"`
}

type releasePolicy struct {
	PolicyVersion     int    `json:"policy_version"`
	Description       string `json:"description"`
	ApplicationStatus string `json:"application_status"`
	Target            struct {
		Repository string `json:"repository"`
		Branch     string `json:"branch"`
	} `json:"target"`
	RequiredWorkflows []requiredWorkflow `json:"required_workflows"`
	TagPattern        string             `json:"tag_pattern"`
	MainRef           string             `json:"main_ref"`
	ArtifactInventory struct {
		Names []string `json:"names"`
	} `json:"artifact_inventory"`
	GitHubRequiredStatusChecks requiredStatusChecks `json:"github_required_status_checks"`
}

// loadPolicy is the RED stub: the policy schema exists so tests compile, but
// no policy is ever loaded or validated until the verifier is implemented.
func loadPolicy(path string) (*releasePolicy, error) {
	return nil, fmt.Errorf("stub: policy loading not implemented for %s", path)
}
