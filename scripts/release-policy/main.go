// Command release-policy verifies, fail closed, that a release tag may be
// published. It runs standalone and locally: it resolves the tag in a real
// repository, proves the release commit has permitted main ancestry, queries
// the GitHub Actions API for successful required-workflow runs on the EXACT
// release commit (workflow identity, run conclusion, and required job
// conclusions, including the required integration lane), and verifies the
// build artifact manifest binds the published artifacts to that same commit.
// Every operational failure (Git failure, API transport error, timeout,
// malformed response) denies publication; there is no fallback that allows.
//
// The verifier is standalone Machinery: it never requires pvg, nd, Paivot
// metadata, or commit conventions.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/gitcontrol"
	"github.com/RamXX/machinery/internal/processcontrol"
)

const (
	gitTimeout       = 30 * time.Second
	gitOutputLimit   = 1 << 20
	apiResponseLimit = 8 << 20
	apiPerPage       = 100
)

var (
	commitPattern     = regexp.MustCompile(`^[0-9a-f]{40}$`)
	repositoryPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*/[A-Za-z0-9_.-]+$`)
	manifestLine      = regexp.MustCompile(`^([0-9a-f]{64})  ([0-9A-Za-z._+-]+)$`)
	workflowPathShape = regexp.MustCompile(`^\.github/workflows/[A-Za-z0-9._-]+\.yml$`)
)

func main() {
	os.Exit(run(os.Args[1:], os.Environ(), os.Stdout, os.Stderr))
}

type options struct {
	root         string
	tag          string
	policyPath   string
	mainRef      string
	expectSHA    string
	apiBase      string
	repository   string
	manifest     string
	artifactsDir string
	timeout      time.Duration
}

func run(args, environ []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("release-policy", flag.ContinueOnError)
	flags.SetOutput(stderr)
	opts := options{}
	flags.StringVar(&opts.root, "root", "", "trusted repository root")
	flags.StringVar(&opts.tag, "tag", "", "release tag to verify")
	flags.StringVar(&opts.policyPath, "policy", "", "policy JSON path (default <root>/docs/release-policy.json)")
	flags.StringVar(&opts.mainRef, "main-ref", "", "main ref the release commit must descend from (default from policy)")
	flags.StringVar(&opts.expectSHA, "expect-sha", "", "commit the tag must resolve to (for example GITHUB_SHA)")
	flags.StringVar(&opts.apiBase, "api-base", "https://api.github.com", "GitHub API base URL")
	flags.StringVar(&opts.repository, "repository", "", "owner/name slug; required-check verification runs when set")
	flags.StringVar(&opts.manifest, "artifact-manifest", "", "build artifact manifest to verify")
	flags.StringVar(&opts.artifactsDir, "artifacts-dir", "", "directory to re-hash against the artifact manifest")
	flags.DurationVar(&opts.timeout, "timeout", 30*time.Second, "per-request API timeout")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if opts.root == "" || opts.tag == "" {
		fmt.Fprintln(stderr, "release-policy: require -root and -tag")
		return 2
	}
	if opts.repository != "" && !repositoryPattern.MatchString(opts.repository) {
		fmt.Fprintf(stderr, "release-policy: -repository must be an owner/name slug: %s\n", opts.repository)
		return 2
	}
	if opts.expectSHA != "" && !commitPattern.MatchString(strings.ToLower(opts.expectSHA)) {
		fmt.Fprintf(stderr, "release-policy: -expect-sha must be a 40-hex commit: %s\n", opts.expectSHA)
		return 2
	}
	if opts.timeout <= 0 || opts.timeout > 5*time.Minute {
		fmt.Fprintln(stderr, "release-policy: -timeout must be in (0, 5m]")
		return 2
	}
	absRoot, err := filepath.Abs(opts.root)
	if err != nil {
		fmt.Fprintf(stderr, "release-policy: resolve repository root: %v\n", err)
		return 2
	}
	if info, err := os.Stat(absRoot); err != nil || !info.IsDir() {
		fmt.Fprintf(stderr, "release-policy: repository root is not a real directory: %s\n", absRoot)
		return 2
	}
	opts.root = absRoot
	if opts.policyPath == "" {
		opts.policyPath = filepath.Join(opts.root, "docs", "release-policy.json")
	}

	v := verifier{opts: opts, environ: environ, stdout: stdout}
	denies := v.verify()
	if len(denies) > 0 {
		for _, deny := range denies {
			fmt.Fprintln(stderr, deny)
		}
		fmt.Fprintf(stderr, "release-policy: verdict DENY (%d blocking findings)\n", len(denies))
		return 1
	}
	for _, note := range v.notes {
		fmt.Fprintf(stdout, "release-policy: %s\n", note)
	}
	fmt.Fprintln(stdout, "release-policy: verdict ALLOW")
	return 0
}

type verifier struct {
	opts    options
	environ []string
	stdout  io.Writer
	notes   []string
}

func (v *verifier) deny(code, format string, args ...any) string {
	return fmt.Sprintf("release-policy: deny [%s] %s", code, fmt.Sprintf(format, args...))
}

func (v *verifier) note(format string, args ...any) {
	v.notes = append(v.notes, fmt.Sprintf(format, args...))
}

func (v *verifier) verify() []string {
	policy, err := loadPolicy(v.opts.policyPath)
	if err != nil {
		return []string{v.deny("R05", "policy %s invalid: %v", v.opts.policyPath, err)}
	}
	v.note("policy %s (version %d)", filepath.Base(v.opts.policyPath), policy.PolicyVersion)

	if !policy.tagPattern.MatchString(v.opts.tag) {
		return []string{v.deny("R01", "tag %s does not match policy tag pattern %s", v.opts.tag, policy.TagPattern)}
	}
	if strings.HasPrefix(v.opts.tag, "-") {
		return []string{v.deny("R01", "tag %s begins with a dash", v.opts.tag)}
	}

	commit, err := v.gitResolve("refs/tags/" + v.opts.tag + "^{commit}")
	if err != nil {
		return []string{v.deny("R20", "git unavailable, failing closed: %v", err)}
	}
	if commit == "" {
		return []string{v.deny("R02", "tag %s unresolved in repository %s", v.opts.tag, v.opts.root)}
	}
	v.note("tag %s resolves to commit %s", v.opts.tag, commit)
	var denies []string
	if v.opts.expectSHA != "" && v.opts.expectSHA != commit {
		denies = append(denies, v.deny("R03", "expect-sha %s does not match tag commit %s (stale SHA)", v.opts.expectSHA, commit))
	}

	mainRef := v.opts.mainRef
	if mainRef == "" {
		mainRef = policy.MainRef
	}
	mainTip, err := v.gitResolve(mainRef)
	if err != nil {
		return append(denies, v.deny("R20", "git unavailable, failing closed: %v", err))
	}
	if mainTip == "" {
		return append(denies, v.deny("R19", "main ref %s unresolved in repository %s, failing closed", mainRef, v.opts.root))
	}
	ancestor, err := v.gitIsAncestor(commit, mainTip)
	if err != nil {
		return append(denies, v.deny("R20", "git unavailable, failing closed: ancestry query: %v", err))
	}
	if !ancestor {
		denies = append(denies, v.deny("R04", "tag commit %s is not an ancestor of %s (%s)", commit, mainRef, mainTip))
	} else {
		v.note("tag commit descends from %s (%s)", mainRef, mainTip)
	}

	if v.opts.repository != "" {
		denies = append(denies, v.verifyRequiredChecks(policy, commit)...)
	}
	if v.opts.manifest != "" {
		denies = append(denies, v.verifyArtifactManifest(policy, commit)...)
	}
	return denies
}

// gitResolve returns the commit named by rev, "" when the repository holds no
// such revision, or an operational error. A leading dash never reaches Git.
func (v *verifier) gitResolve(rev string) (string, error) {
	if rev == "" || strings.HasPrefix(rev, "-") {
		return "", nil
	}
	out, exitCode, err := v.git("rev-parse", "--verify", "--quiet", rev)
	if err != nil {
		return "", err
	}
	if exitCode == 1 && strings.TrimSpace(out) == "" {
		return "", nil
	}
	if exitCode != 0 {
		return "", fmt.Errorf("git rev-parse %s failed with exit %d", rev, exitCode)
	}
	commit := strings.ToLower(strings.TrimSpace(out))
	if !commitPattern.MatchString(commit) {
		return "", fmt.Errorf("git rev-parse %s returned malformed commit %q", rev, commit)
	}
	return commit, nil
}

func (v *verifier) gitIsAncestor(ancestor, descendant string) (bool, error) {
	_, exitCode, err := v.git("merge-base", "--is-ancestor", ancestor, descendant)
	if err != nil {
		return false, err
	}
	switch exitCode {
	case 0:
		return true, nil
	case 1:
		return false, nil
	default:
		return false, fmt.Errorf("git merge-base --is-ancestor failed with exit %d", exitCode)
	}
}

// git runs one read-only Git query with a closed environment, bounded output,
// and warning-free success semantics, mirroring scripts/git-safe.
func (v *verifier) git(args ...string) (string, int, error) {
	gitPath, err := lookPathInEnviron(v.environ, "git")
	if err != nil {
		return "", -1, fmt.Errorf("resolve Git executable: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, gitPath, append([]string{"-C", v.opts.root}, args...)...)
	cmd.Dir = v.opts.root
	cmd.Env = gitcontrol.Environment(v.environ)
	stdout, stderr, runErr := processcontrol.RunCapturedStreams(ctx, cmd, gitOutputLimit)
	if ctx.Err() != nil {
		return "", -1, fmt.Errorf("git %s timed out after %s", strings.Join(args, " "), gitTimeout)
	}
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			return stdout, exitErr.ExitCode(), nil
		}
		return "", -1, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), runErr, strings.TrimSpace(stderr))
	}
	if strings.TrimSpace(stderr) != "" {
		return "", -1, fmt.Errorf("git %s emitted stderr on success: %s", strings.Join(args, " "), strings.TrimSpace(stderr))
	}
	return stdout, 0, nil
}

type workflowRun struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	HeadSHA    string `json:"head_sha"`
	Event      string `json:"event"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

type runsResponse struct {
	TotalCount   int           `json:"total_count"`
	WorkflowRuns []workflowRun `json:"workflow_runs"`
}

type runJob struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

type jobsResponse struct {
	TotalCount int      `json:"total_count"`
	Jobs       []runJob `json:"jobs"`
}

func (v *verifier) apiGet(pathname string, query url.Values, out any) error {
	endpoint := strings.TrimSuffix(v.opts.apiBase, "/") + pathname
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	ctx, cancel := context.WithTimeout(context.Background(), v.opts.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "machinery-release-policy")
	if token := environValue(v.environ, "RELEASE_POLICY_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("GET %s timed out after %s; fail closed", endpoint, v.opts.timeout)
		}
		return fmt.Errorf("GET %s: %w; fail closed", endpoint, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, apiResponseLimit+1))
	if err != nil {
		return fmt.Errorf("read response: %w; fail closed", err)
	}
	if len(body) > apiResponseLimit {
		return fmt.Errorf("response exceeds %d bytes; fail closed", apiResponseLimit)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("GET %s returned HTTP %d; fail closed", endpoint, resp.StatusCode)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return &malformedAPIError{err: fmt.Errorf("malformed response: %w", err)}
	}
	return nil
}

func environValue(environ []string, key string) string {
	for _, item := range environ {
		if value, ok := strings.CutPrefix(item, key+"="); ok {
			return value
		}
	}
	return ""
}

// lookPathInEnviron resolves name against the PATH carried by environ, so the
// verifier honors the caller's environment rather than the host process's.
func lookPathInEnviron(environ []string, name string) (string, error) {
	for _, dir := range filepath.SplitList(environValue(environ, "PATH")) {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, name)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("%s not found in the provided PATH", name)
}

// malformedAPIError marks a response that arrived but cannot be trusted as a
// well-formed GitHub API document. It is a distinct deny class (R14) from a
// response that never arrived (R13).
type malformedAPIError struct{ err error }

func (e *malformedAPIError) Error() string { return e.err.Error() }
func (e *malformedAPIError) Unwrap() error { return e.err }

func (v *verifier) fetchRuns(commit string) ([]workflowRun, error) {
	var doc runsResponse
	query := url.Values{}
	query.Set("head_sha", commit)
	query.Set("per_page", fmt.Sprint(apiPerPage))
	if err := v.apiGet("/repos/"+v.opts.repository+"/actions/runs", query, &doc); err != nil {
		return nil, err
	}
	if doc.TotalCount > apiPerPage {
		return nil, &malformedAPIError{err: fmt.Errorf("malformed response: page truncated (total_count %d exceeds one page of %d)", doc.TotalCount, apiPerPage)}
	}
	if doc.TotalCount != len(doc.WorkflowRuns) {
		return nil, &malformedAPIError{err: fmt.Errorf("malformed response: total_count %d does not match %d returned runs", doc.TotalCount, len(doc.WorkflowRuns))}
	}
	for _, run := range doc.WorkflowRuns {
		if run.ID <= 0 || run.Name == "" || run.Path == "" || run.Event == "" || run.Status == "" {
			return nil, &malformedAPIError{err: fmt.Errorf("malformed response: run %d carries an empty identity field", run.ID)}
		}
		if !commitPattern.MatchString(strings.ToLower(run.HeadSHA)) {
			return nil, &malformedAPIError{err: fmt.Errorf("malformed response: run %d carries malformed head_sha %q", run.ID, run.HeadSHA)}
		}
		if run.Status == "completed" && run.Conclusion == "" {
			return nil, &malformedAPIError{err: fmt.Errorf("malformed response: completed run %d has no conclusion", run.ID)}
		}
	}
	return doc.WorkflowRuns, nil
}

func (v *verifier) fetchJobs(runID int64) ([]runJob, error) {
	var doc jobsResponse
	if err := v.apiGet(fmt.Sprintf("/repos/%s/actions/runs/%d/jobs", v.opts.repository, runID), nil, &doc); err != nil {
		return nil, err
	}
	if doc.TotalCount != len(doc.Jobs) {
		return nil, &malformedAPIError{err: fmt.Errorf("malformed response: total_count %d does not match %d returned jobs", doc.TotalCount, len(doc.Jobs))}
	}
	for _, job := range doc.Jobs {
		if job.ID <= 0 || job.Name == "" || job.Status == "" {
			return nil, &malformedAPIError{err: fmt.Errorf("malformed response: job %d carries an empty identity field", job.ID)}
		}
		if job.Status == "completed" && job.Conclusion == "" {
			return nil, &malformedAPIError{err: fmt.Errorf("malformed response: completed job %d has no conclusion", job.ID)}
		}
	}
	return doc.Jobs, nil
}

func (v *verifier) verifyRequiredChecks(policy *releasePolicy, commit string) []string {
	runs, err := v.fetchRuns(commit)
	if err != nil {
		denies := make([]string, 0, len(policy.RequiredWorkflows))
		var malformed *malformedAPIError
		for range policy.RequiredWorkflows {
			if errors.As(err, &malformed) {
				denies = append(denies, v.deny("R14", "API response %v", err))
			} else {
				denies = append(denies, v.deny("R13", "API unavailable, failing closed: %v", err))
			}
		}
		return denies
	}
	exact := make([]workflowRun, 0, len(runs))
	for _, run := range runs {
		if strings.ToLower(run.HeadSHA) == commit {
			exact = append(exact, run)
		}
	}
	jobsCache := map[int64][]runJob{}
	var denies []string
	for _, wf := range policy.RequiredWorkflows {
		allowedEvents := map[string]bool{}
		for _, event := range wf.Events {
			allowedEvents[event] = true
		}
		var disallowedEvent, candidates []workflowRun
		nearIdentity := false
		for _, run := range exact {
			pathMatch := run.Path == wf.Path
			nameMatch := run.Name == wf.Name
			switch {
			case pathMatch && nameMatch && allowedEvents[run.Event]:
				candidates = append(candidates, run)
			case pathMatch && nameMatch:
				disallowedEvent = append(disallowedEvent, run)
			case pathMatch || nameMatch:
				nearIdentity = true
			}
		}
		if len(candidates) == 0 && len(disallowedEvent) == 0 {
			if nearIdentity {
				denies = append(denies, v.deny("R09", "workflow identity mismatch: a run for the exact commit matches %s only in name or only in path", wf.Path))
			} else {
				denies = append(denies, v.deny("R06", "required workflow run missing for exact commit %s: no run of %s (%s)", commit, wf.Path, wf.Name))
			}
			continue
		}
		if len(candidates) == 0 {
			for _, run := range disallowedEvent {
				denies = append(denies, v.deny("R12", "event %s not permitted by policy for %s (run %d); permitted: %s", run.Event, wf.Path, run.ID, strings.Join(wf.Events, ", ")))
			}
			continue
		}
		for _, run := range candidates {
			if run.Status != "completed" {
				denies = append(denies, v.deny("R07", "run %d of %s not completed (status %s)", run.ID, wf.Path, run.Status))
				continue
			}
			if run.Conclusion != "success" {
				denies = append(denies, v.deny("R08", "run %d of %s completed with conclusion %s, not success", run.ID, wf.Path, run.Conclusion))
				continue
			}
			jobs, cached := jobsCache[run.ID]
			if !cached {
				fetched, err := v.fetchJobs(run.ID)
				if err != nil {
					var malformed *malformedAPIError
					if errors.As(err, &malformed) {
						denies = append(denies, v.deny("R14", "API response for jobs of run %d: %v", run.ID, err))
					} else {
						denies = append(denies, v.deny("R13", "API unavailable, failing closed: jobs of run %d: %v", run.ID, err))
					}
					continue
				}
				jobs = fetched
				jobsCache[run.ID] = jobs
			}
			runPassing := true
			for _, required := range wf.RequiredJobs {
				present, success := false, false
				badStatus, badConclusion := "", ""
				for _, job := range jobs {
					if job.Name != required {
						continue
					}
					present = true
					if job.Status == "completed" && job.Conclusion == "success" {
						success = true
					} else {
						badStatus, badConclusion = job.Status, job.Conclusion
					}
				}
				switch {
				case !present:
					runPassing = false
					denies = append(denies, v.deny("R10", "required job missing from run %d of %s: %s", run.ID, wf.Path, required))
				case !success:
					runPassing = false
					denies = append(denies, v.deny("R11", "required job not successful in run %d of %s: %s status=%s conclusion=%s", run.ID, wf.Path, required, badStatus, badConclusion))
				}
			}
			if runPassing {
				v.note("workflow %s run %d (%s) success; required jobs green: %s", wf.Path, run.ID, run.Event, strings.Join(wf.RequiredJobs, ", "))
			}
		}
	}
	return denies
}

func (v *verifier) verifyArtifactManifest(policy *releasePolicy, commit string) []string {
	raw, err := os.ReadFile(v.opts.manifest)
	if err != nil {
		return []string{v.deny("R15", "artifact manifest unreadable: %v", err)}
	}
	text := strings.TrimSuffix(string(raw), "\n")
	lines := strings.Split(text, "\n")
	if len(lines) < 2 {
		return []string{v.deny("R17", "manifest malformed: expected a commit header and at least one entry")}
	}
	header, ok := strings.CutPrefix(lines[0], "commit ")
	if !ok || !commitPattern.MatchString(header) {
		return []string{v.deny("R17", "manifest malformed: first line must be 'commit <40-hex>', got %q", lines[0])}
	}
	if header != commit {
		return []string{v.deny("R16", "manifest header commit %s is not the release commit %s", header, commit)}
	}
	digests := map[string]string{}
	names := make([]string, 0, len(lines)-1)
	for _, line := range lines[1:] {
		match := manifestLine.FindStringSubmatch(line)
		if match == nil {
			return []string{v.deny("R17", "manifest malformed: entry %q is not '<64-hex>  <name>'", line)}
		}
		digest, name := match[1], match[2]
		if _, dup := digests[name]; dup {
			return []string{v.deny("R17", "manifest malformed: duplicate artifact name %s", name)}
		}
		digests[name] = digest
		names = append(names, name)
	}
	if !sort.StringsAreSorted(names) {
		return []string{v.deny("R17", "manifest malformed: entries are not sorted by artifact name")}
	}
	plain := strings.TrimPrefix(v.opts.tag, "v")
	expected := map[string]bool{}
	for _, name := range policy.ArtifactInventory.Names {
		expected[strings.ReplaceAll(name, "{version}", plain)] = true
	}
	if len(expected) != len(digests) {
		return []string{v.deny("R18", "artifact identity: manifest carries %d names, policy inventory expects %d", len(digests), len(expected))}
	}
	for name := range digests {
		if !expected[name] {
			return []string{v.deny("R18", "artifact identity: %s is outside the policy artifact inventory", name)}
		}
	}
	v.note("artifact manifest binds %d artifacts to commit %s", len(digests), commit)
	if v.opts.artifactsDir == "" {
		return nil
	}
	for _, name := range names {
		path := filepath.Join(v.opts.artifactsDir, name)
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			return []string{v.deny("R18", "artifact identity: %s is not a regular file in %s", name, v.opts.artifactsDir)}
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return []string{v.deny("R18", "artifact identity: reading %s: %v", name, err)}
		}
		sum := sha256.Sum256(body)
		if hex.EncodeToString(sum[:]) != digests[name] {
			return []string{v.deny("R18", "artifact identity: %s content digest does not match the manifest", name)}
		}
	}
	v.note("artifact contents re-hashed against the manifest in %s", v.opts.artifactsDir)
	return nil
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
	PolicyVersion     int                `json:"policy_version"`
	Description       string             `json:"description"`
	ApplicationStatus string             `json:"application_status"`
	Target            target             `json:"target"`
	RequiredWorkflows []requiredWorkflow `json:"required_workflows"`
	TagPattern        string             `json:"tag_pattern"`
	MainRef           string             `json:"main_ref"`
	ArtifactInventory struct {
		Names []string `json:"names"`
	} `json:"artifact_inventory"`
	GitHubRequiredStatusChecks requiredStatusChecks `json:"github_required_status_checks"`

	tagPattern *regexp.Regexp `json:"-"`
}

type target struct {
	Repository string `json:"repository"`
	Branch     string `json:"branch"`
}

// loadPolicy reads and schema-validates a policy document. Unknown fields are
// rejected so a drifting policy file cannot silently change meaning.
func loadPolicy(path string) (*releasePolicy, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	policy := &releasePolicy{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(policy); err != nil {
		return nil, err
	}
	if err := policy.validate(); err != nil {
		return nil, err
	}
	return policy, nil
}

func (policy *releasePolicy) validate() error {
	if policy.PolicyVersion != 1 {
		return fmt.Errorf("unsupported policy_version %d", policy.PolicyVersion)
	}
	if policy.Target.Repository == "" || policy.Target.Branch == "" {
		return fmt.Errorf("target repository and branch are required")
	}
	pattern, err := regexp.Compile(policy.TagPattern)
	if err != nil {
		return fmt.Errorf("tag_pattern: %w", err)
	}
	policy.tagPattern = pattern
	if !strings.HasPrefix(policy.MainRef, "refs/") {
		return fmt.Errorf("main_ref %q must be a refs/ name", policy.MainRef)
	}
	if len(policy.RequiredWorkflows) == 0 {
		return fmt.Errorf("at least one required workflow is required")
	}
	seenPaths := map[string]bool{}
	for _, wf := range policy.RequiredWorkflows {
		if !workflowPathShape.MatchString(wf.Path) {
			return fmt.Errorf("workflow path %q is not a .github/workflows/*.yml name", wf.Path)
		}
		if seenPaths[wf.Path] {
			return fmt.Errorf("workflow %s listed twice", wf.Path)
		}
		seenPaths[wf.Path] = true
		if wf.Name == "" {
			return fmt.Errorf("workflow %s has no name", wf.Path)
		}
		if err := uniqueNonEmpty(wf.Events, fmt.Sprintf("events of %s", wf.Path)); err != nil {
			return err
		}
		if err := uniqueNonEmpty(wf.RequiredJobs, fmt.Sprintf("required_jobs of %s", wf.Path)); err != nil {
			return err
		}
	}
	names := policy.ArtifactInventory.Names
	if err := uniqueNonEmpty(names, "artifact inventory names"); err != nil {
		return err
	}
	versioned := 0
	for _, name := range names {
		if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
			return fmt.Errorf("artifact name %q must be a plain file name", name)
		}
		switch strings.Count(name, "{version}") {
		case 0:
		case 1:
			versioned++
		default:
			return fmt.Errorf("artifact name %q carries more than one {version}", name)
		}
	}
	if versioned == 0 {
		return fmt.Errorf("artifact inventory carries no {version} name; it cannot bind a release")
	}
	github := policy.GitHubRequiredStatusChecks
	if !github.Payload.Strict {
		return fmt.Errorf("branch-protection payload must be strict")
	}
	if err := uniqueNonEmpty(github.Payload.Contexts, "branch-protection contexts"); err != nil {
		return err
	}
	for _, command := range []string{github.ApplyCommand, github.VerifyCommand, github.PreApplyNameConfirmation} {
		if strings.TrimSpace(command) == "" {
			return fmt.Errorf("policy must document apply, verify, and pre-apply name-confirmation commands")
		}
	}
	return nil
}

func uniqueNonEmpty(values []string, what string) error {
	if len(values) == 0 {
		return fmt.Errorf("%s must not be empty", what)
	}
	seen := map[string]bool{}
	for _, value := range values {
		if value == "" {
			return fmt.Errorf("%s carries an empty entry", what)
		}
		if seen[value] {
			return fmt.Errorf("%s lists %q twice", what, value)
		}
		seen[value] = true
	}
	return nil
}
