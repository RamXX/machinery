// Command runtime-pins reports how every pinned native runtime compares with
// the host and with the latest upstream release, so the pins cannot age
// silently.
//
// The pinned column is read from internal/runtimeclosure, the single owner
// of every runtime pin. The host column runs the runtime found on PATH. The
// latest column is fetched from each runtime's own release index. When no
// index can be reached at all the column prints "offline"; when one index
// fails while others answer, its cell names the failure ("lookup failed:
// HTTP 403") and a warning is printed, because that is a broken lookup, not
// a missing network.
//
// Status is "ok" or "drift" (pin versus latest upstream) or "unknown" (no
// latest), plus "host behind" or "host ahead" when the runtime on PATH is
// not the pinned version, which is what fails the native suites locally.
//
// Exit codes: 1 when a pin is behind upstream; 2 under -strict when a latest
// release is unknown; 3 under -host-strict when a host runtime is not the
// pinned version; else 0. CI runs -strict (it has no host runtimes);
// -host-strict is for a contributor machine. `go run` reports every nonzero
// exit as 1; build the binary to see the distinct codes.
//
//	go run ./scripts/runtime-pins                 # report; exit 1 on drift
//	go run ./scripts/runtime-pins -offline        # pinned and host only
//	go run ./scripts/runtime-pins -strict         # also fail on an unknown latest
//	go run ./scripts/runtime-pins -host-strict    # also fail when the host is not the pin
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/RamXX/machinery/internal/runtimeclosure"
)

const (
	offline      = "offline"
	absent       = "absent"
	lookupFailed = "lookup failed: "
)

// sources names every upstream index. Tests point them at local servers.
type sources struct {
	OTPTable       string // erlang/otp otp_versions.table: every OTP release and its erts
	ElixirReleases string // GitHub releases API for elixir-lang/elixir
	ElixirBuilds   string // builds.hex.pm Elixir build list, the fallback
	NodeIndex      string // nodejs.org release index
	TypeScript     string // npm registry dist-tag latest
	PythonReleases string // python.org release API
	JavaReleases   string // Adoptium release names, with %d for the feature version
	JavaInfo       string // Adoptium available releases
	GoReleases     string // go.dev release list
}

var upstream = sources{
	OTPTable:       "https://raw.githubusercontent.com/erlang/otp/master/otp_versions.table",
	ElixirReleases: "https://api.github.com/repos/elixir-lang/elixir/releases?per_page=50",
	ElixirBuilds:   "https://builds.hex.pm/builds/elixir/builds.txt",
	NodeIndex:      "https://nodejs.org/dist/index.json",
	TypeScript:     "https://registry.npmjs.org/typescript/latest",
	PythonReleases: "https://www.python.org/api/v2/downloads/release/?is_published=true&pre_release=false",
	JavaReleases:   "https://api.adoptium.net/v3/info/release_names?release_type=ga&vendor=eclipse&image_type=jdk&page_size=50&sort_order=DESC&version=%%5B%d,%d%%29",
	JavaInfo:       "https://api.adoptium.net/v3/info/available_releases",
	GoReleases:     "https://go.dev/dl/?mode=json",
}

// row is one runtime line of the report.
type row struct {
	Runtime string
	Pinned  string
	Host    string
	Latest  string
}

// PinStatus is "ok" when the pin is the latest release, "drift" when it is
// not, and "unknown" when the latest release could not be fetched.
func (r row) PinStatus() string {
	switch {
	case r.Latest == offline || strings.HasPrefix(r.Latest, lookupFailed):
		return "unknown"
	case r.Latest == r.Pinned:
		return "ok"
	default:
		return "drift"
	}
}

// HostStatus is "host behind" or "host ahead" when the runtime on PATH is
// not the pinned version, else empty. Java is exempt: machinery provisions
// its pinned Temurin itself, so the PATH java never gates a suite.
func (r row) HostStatus() string {
	if r.Host == absent || r.Host == "" || r.Runtime == "Java" {
		return ""
	}
	switch c := compareVersions(r.Host, r.Pinned); {
	case c < 0:
		return "host behind"
	case c > 0:
		return "host ahead"
	default:
		return ""
	}
}

// Status is the pin verdict, plus the host verdict when there is one.
func (r row) Status() string {
	if host := r.HostStatus(); host != "" {
		return r.PinStatus() + ", " + host
	}
	return r.PinStatus()
}

// pinned is the closed list of pinned runtimes, in report order, read from
// the one place each pin is defined.
func pinned() []row {
	return []row{
		{Runtime: "Erlang/OTP", Pinned: runtimeclosure.RequiredOTPVersion},
		{Runtime: "erts", Pinned: runtimeclosure.RequiredErtsVersion},
		{Runtime: "Elixir", Pinned: runtimeclosure.RequiredElixirVersion},
		{Runtime: "Node", Pinned: runtimeclosure.RequiredNodeRelease},
		{Runtime: "TypeScript", Pinned: runtimeclosure.RequiredTypeScriptVersion},
		{Runtime: "Python", Pinned: runtimeclosure.RequiredPythonVersion},
		{Runtime: "Java", Pinned: runtimeclosure.PinnedJavaRuntimeVersion},
		{Runtime: "Go", Pinned: runtimeclosure.RequiredGoVersion},
	}
}

func main() {
	offlineFlag := flag.Bool("offline", false, "do not contact upstream indexes")
	strict := flag.Bool("strict", false, "exit 2 when a latest upstream release is unknown")
	hostStrict := flag.Bool("host-strict", false, "exit 3 when a runtime on PATH is not the pinned version")
	timeout := flag.Duration("timeout", 20*time.Second, "per-request timeout")
	flag.Parse()
	ctx := context.Background()
	rows := pinned()
	fillHost(ctx, rows, hostProbes())
	var notes []string
	if *offlineFlag {
		for i := range rows {
			rows[i].Latest = offline
		}
	} else {
		client := &http.Client{Timeout: *timeout}
		notes = fillLatest(ctx, rows, fetcher{client: client, githubToken: os.Getenv("GITHUB_TOKEN")}, upstream)
	}
	fmt.Print(render(rows, notes))
	os.Exit(exitCode(rows, *strict, *hostStrict))
}

// exitCode is 1 on any drift, else 2 under strict when a latest release is
// unknown, else 3 under hostStrict when a host runtime is not the pin,
// else 0.
func exitCode(rows []row, strict, hostStrict bool) int {
	unknown, host := false, false
	for _, r := range rows {
		switch r.PinStatus() {
		case "drift":
			return 1
		case "unknown":
			unknown = true
		}
		host = host || r.HostStatus() != ""
	}
	switch {
	case strict && unknown:
		return 2
	case hostStrict && host:
		return 3
	default:
		return 0
	}
}

// render prints the report as a Markdown table, readable in a terminal and
// pasteable into an issue as is.
func render(rows []row, notes []string) string {
	var b strings.Builder
	b.WriteString("| runtime | pinned | host-installed | latest-upstream | status |\n")
	b.WriteString("|---|---|---|---|---|\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", r.Runtime, r.Pinned, r.Host, r.Latest, r.Status())
	}
	for _, note := range notes {
		fmt.Fprintf(&b, "\n%s\n", note)
	}
	return b.String()
}

// hostProbe runs one host command and extracts the version from its
// combined output with the first pattern that matches.
type hostProbe struct {
	argv    []string
	extract []*regexp.Regexp
}

func patterns(exprs ...string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, len(exprs))
	for i, expr := range exprs {
		out[i] = regexp.MustCompile(expr)
	}
	return out
}

func hostProbes() map[string]hostProbe {
	return map[string]hostProbe{
		"Erlang/OTP": {[]string{"erl", "-noshell", "-eval", otpProbe, "-s", "init", "stop"}, patterns(`OTP=(\S+)`)},
		"erts":       {[]string{"erl", "-noshell", "-eval", otpProbe, "-s", "init", "stop"}, patterns(`ERTS=(\S+)`)},
		"Elixir":     {[]string{"elixir", "--version"}, patterns(`Elixir (\d+\.\d+\.\d+)`)},
		"Node":       {[]string{"node", "--version"}, patterns(`^v(\d+\.\d+\.\d+)`)},
		"TypeScript": {[]string{"tsc", "--version"}, patterns(`Version (\d+\.\d+\.\d+)`)},
		"Python":     {[]string{"python3", "--version"}, patterns(`Python (\d+\.\d+\.\d+)`)},
		"Java":       {[]string{"java", "-version"}, patterns(`Temurin-(\d[\d.]*\+\d+)`, `version "([^"]+)"`)},
		"Go":         {[]string{"go", "env", "GOVERSION"}, patterns(`^go(\d+\.\d+(?:\.\d+)?)`)},
	}
}

// otpProbe prints the full OTP version from the installation's own
// OTP_VERSION record and the running erts version.
const otpProbe = `{ok,V}=file:read_file(filename:join([code:root_dir(),"releases",erlang:system_info(otp_release),"OTP_VERSION"])),io:format("OTP=~s ERTS=~s~n",[string:trim(V),erlang:system_info(version)]).`

func fillHost(ctx context.Context, rows []row, probes map[string]hostProbe) {
	for i := range rows {
		rows[i].Host = absent
		probe, ok := probes[rows[i].Runtime]
		if !ok {
			continue
		}
		if version := runHostProbe(ctx, probe); version != "" {
			rows[i].Host = version
		}
	}
}

func runHostProbe(ctx context.Context, probe hostProbe) string {
	if _, err := exec.LookPath(probe.argv[0]); err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, probe.argv[0], probe.argv[1:]...).CombinedOutput()
	if err != nil {
		return ""
	}
	return extractVersion(probe.extract, out)
}

// extractVersion returns the first capture of the first pattern that
// matches any output line; patterns are tried in priority order.
func extractVersion(res []*regexp.Regexp, out []byte) string {
	for _, re := range res {
		scanner := bufio.NewScanner(bytes.NewReader(out))
		for scanner.Scan() {
			if match := re.FindStringSubmatch(strings.TrimSpace(scanner.Text())); len(match) > 1 && match[1] != "" {
				return match[1]
			}
		}
	}
	return ""
}

type fetcher struct {
	client *http.Client
	// githubToken authenticates GitHub API calls when set (the workflow
	// passes the job token); unauthenticated calls share a 60/hour limit.
	githubToken string
}

// httpError is a non-200 answer. rateLimited marks GitHub's exhausted
// quota (403 or 429 with X-RateLimit-Remaining: 0) so it is reported as
// such instead of as a generic denial.
type httpError struct {
	url         string
	code        int
	rateLimited bool
}

func (e *httpError) Error() string {
	if e.rateLimited {
		return fmt.Sprintf("GET %s: rate limited (HTTP %d)", e.url, e.code)
	}
	return fmt.Sprintf("GET %s: HTTP %d", e.url, e.code)
}

// networkError is a request that never got an HTTP answer (DNS, connect,
// timeout): the only failure that counts toward "offline".
type networkError struct{ err error }

func (e *networkError) Error() string { return e.err.Error() }
func (e *networkError) Unwrap() error { return e.err }

func (f fetcher) get(ctx context.Context, url string) ([]byte, error) {
	return f.getWith(ctx, url, nil)
}

func (f fetcher) getWith(ctx context.Context, url string, header http.Header) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	for key, values := range header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	req.Header.Set("User-Agent", "machinery-runtime-pins")
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, &networkError{err}
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		limited := (resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests) &&
			resp.Header.Get("X-RateLimit-Remaining") == "0"
		return nil, &httpError{url: url, code: resp.StatusCode, rateLimited: limited}
	}
	return io.ReadAll(io.LimitReader(resp.Body, 32<<20))
}

// failureClass names why a lookup failed, for the latest-upstream cell.
func failureClass(err error) string {
	var h *httpError
	var n *networkError
	switch {
	case errors.As(err, &h) && h.rateLimited:
		return fmt.Sprintf("rate limited (HTTP %d)", h.code)
	case errors.As(err, &h):
		return fmt.Sprintf("HTTP %d", h.code)
	case errors.As(err, &n):
		return "network"
	default:
		return "parse"
	}
}

// fillLatest fetches every index concurrently and fills the latest column;
// any failure leaves that row "offline". It returns informational notes.
func fillLatest(ctx context.Context, rows []row, f fetcher, src sources) []string {
	latest := map[string]string{}
	failures := map[string]error{}
	var notes []string
	var mu sync.Mutex
	set := func(runtime, version string, err error) {
		mu.Lock()
		defer mu.Unlock()
		if err == nil && version == "" {
			err = errors.New("empty version")
		}
		if err != nil {
			failures[runtime] = err
			return
		}
		latest[runtime] = version
	}
	javaMajor := javaFeature(runtimeclosure.PinnedJavaRuntimeVersion)
	jobs := []func(){
		func() {
			otp, erts, err := latestOTP(ctx, f, src.OTPTable)
			set("Erlang/OTP", otp, err)
			set("erts", erts, err)
		},
		func() { v, err := latestElixir(ctx, f, src.ElixirReleases, src.ElixirBuilds); set("Elixir", v, err) },
		func() { v, err := latestNode(ctx, f, src.NodeIndex); set("Node", v, err) },
		func() { v, err := latestTypeScript(ctx, f, src.TypeScript); set("TypeScript", v, err) },
		func() { v, err := latestPython(ctx, f, src.PythonReleases); set("Python", v, err) },
		func() {
			v, err := latestJava(ctx, f, src.JavaReleases, javaMajor)
			set("Java", v, err)
			if lts, err := newestJavaLTS(ctx, f, src.JavaInfo); err == nil && lts > javaMajor {
				mu.Lock()
				notes = append(notes, fmt.Sprintf("Java: the pin tracks the Temurin %d LTS line; the newest Temurin LTS is %d. Moving to it is a separate, deliberate change (see CONTRIBUTING.md, Runtime pins).", javaMajor, lts))
				mu.Unlock()
			}
		},
		func() { v, err := latestGo(ctx, f, src.GoReleases); set("Go", v, err) },
	}
	var wg sync.WaitGroup
	for _, job := range jobs {
		wg.Add(1)
		go func() { defer wg.Done(); job() }()
	}
	wg.Wait()
	// "offline" only when nothing answered and every failure is a network
	// failure; otherwise each failed cell names its own failure.
	allOffline := len(latest) == 0
	for _, err := range failures {
		allOffline = allOffline && failureClass(err) == "network"
	}
	for i := range rows {
		runtime := rows[i].Runtime
		if v, ok := latest[runtime]; ok {
			rows[i].Latest = v
			continue
		}
		if allOffline {
			rows[i].Latest = offline
			continue
		}
		err := failures[runtime]
		if err == nil {
			err = errors.New("no result")
		}
		rows[i].Latest = lookupFailed + failureClass(err)
		notes = append(notes, fmt.Sprintf("warning: %s latest-upstream lookup failed: %v", runtime, err))
	}
	return notes
}

var dotted = regexp.MustCompile(`^\d+(\.\d+)*$`)

// compareVersions orders dotted numeric versions; a longer version with an
// equal prefix is newer (29.1.1 > 29.1).
func compareVersions(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) || i < len(bs); i++ {
		if i >= len(as) {
			return -1
		}
		if i >= len(bs) {
			return 1
		}
		x, _ := strconv.Atoi(as[i])
		y, _ := strconv.Atoi(bs[i])
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}

// maxVersion returns the newest dotted numeric version, ignoring anything
// that is not one (release candidates, tags).
func maxVersion(candidates []string) (string, error) {
	best := ""
	for _, c := range candidates {
		if !dotted.MatchString(c) {
			continue
		}
		if best == "" || compareVersions(c, best) > 0 {
			best = c
		}
	}
	if best == "" {
		return "", errors.New("no release version in index")
	}
	return best, nil
}

var otpLine = regexp.MustCompile(`^OTP-(\d+(?:\.\d+)*) :.*\berts-(\d+(?:\.\d+)*)\b`)

// latestOTP reads otp_versions.table: one line per OTP release naming every
// application version, erts included, so the release and its erts come
// from the same authoritative record.
func latestOTP(ctx context.Context, f fetcher, url string) (string, string, error) {
	body, err := f.get(ctx, url)
	if err != nil {
		return "", "", err
	}
	erts := map[string]string{}
	var versions []string
	for _, line := range strings.Split(string(body), "\n") {
		if m := otpLine.FindStringSubmatch(line); m != nil {
			versions = append(versions, m[1])
			erts[m[1]] = m[2]
		}
	}
	otp, err := maxVersion(versions)
	if err != nil {
		return "", "", err
	}
	return otp, erts[otp], nil
}

// latestElixir asks the GitHub releases API first and falls back to the
// hex.pm build list when GitHub fails (most often the unauthenticated rate
// limit). Both failing returns both errors; the GitHub one classifies the
// cell unless it was only a network error.
func latestElixir(ctx context.Context, f fetcher, releasesURL, buildsURL string) (string, error) {
	version, githubErr := latestElixirGitHub(ctx, f, releasesURL)
	if githubErr == nil {
		return version, nil
	}
	version, buildsErr := latestElixirBuilds(ctx, f, buildsURL)
	if buildsErr == nil {
		return version, nil
	}
	var n *networkError
	if errors.As(githubErr, &n) {
		return "", errors.Join(buildsErr, githubErr)
	}
	return "", errors.Join(githubErr, buildsErr)
}

func latestElixirGitHub(ctx context.Context, f fetcher, url string) (string, error) {
	header := http.Header{"Accept": {"application/vnd.github+json"}, "X-Github-Api-Version": {"2022-11-28"}}
	if f.githubToken != "" {
		header.Set("Authorization", "Bearer "+f.githubToken)
	}
	body, err := f.getWith(ctx, url, header)
	if err != nil {
		return "", err
	}
	var releases []struct {
		TagName    string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	if err := json.Unmarshal(body, &releases); err != nil {
		return "", err
	}
	var versions []string
	for _, release := range releases {
		if !release.Draft && !release.Prerelease {
			versions = append(versions, strings.TrimPrefix(release.TagName, "v"))
		}
	}
	return maxVersion(versions)
}

// latestElixirBuilds reads the hex.pm build list: "v<version> <sha> <date> ...".
func latestElixirBuilds(ctx context.Context, f fetcher, url string) (string, error) {
	body, err := f.get(ctx, url)
	if err != nil {
		return "", err
	}
	var versions []string
	for _, line := range strings.Split(string(body), "\n") {
		if field, _, _ := strings.Cut(line, " "); strings.HasPrefix(field, "v") {
			versions = append(versions, strings.TrimPrefix(field, "v"))
		}
	}
	return maxVersion(versions)
}

func latestNode(ctx context.Context, f fetcher, url string) (string, error) {
	body, err := f.get(ctx, url)
	if err != nil {
		return "", err
	}
	var index []struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(body, &index); err != nil {
		return "", err
	}
	versions := make([]string, 0, len(index))
	for _, release := range index {
		versions = append(versions, strings.TrimPrefix(release.Version, "v"))
	}
	return maxVersion(versions)
}

func latestTypeScript(ctx context.Context, f fetcher, url string) (string, error) {
	body, err := f.get(ctx, url)
	if err != nil {
		return "", err
	}
	var doc struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return "", err
	}
	return maxVersion([]string{doc.Version})
}

func latestPython(ctx context.Context, f fetcher, url string) (string, error) {
	body, err := f.get(ctx, url)
	if err != nil {
		return "", err
	}
	var releases []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &releases); err != nil {
		return "", err
	}
	versions := make([]string, 0, len(releases))
	for _, release := range releases {
		versions = append(versions, strings.TrimPrefix(release.Name, "Python "))
	}
	return maxVersion(versions)
}

// javaFeature is the feature (major) version of a Temurin version string.
func javaFeature(version string) int {
	major, _, _ := strings.Cut(version, ".")
	n, _ := strconv.Atoi(major)
	return n
}

// latestJava returns the newest GA Temurin build of one feature line, in
// the pin's spelling ("21.0.12.1+1").
func latestJava(ctx context.Context, f fetcher, urlPattern string, feature int) (string, error) {
	body, err := f.get(ctx, fmt.Sprintf(urlPattern, feature, feature+1))
	if err != nil {
		return "", err
	}
	var doc struct {
		Releases []string `json:"releases"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return "", err
	}
	best, bestBuild := "", 0
	for _, name := range doc.Releases {
		version, build, ok := strings.Cut(strings.TrimPrefix(name, "jdk-"), "+")
		n, err := strconv.Atoi(build)
		if !ok || err != nil || !dotted.MatchString(version) || javaFeature(version) != feature {
			continue
		}
		if best == "" || compareVersions(version, best) > 0 || (version == best && n > bestBuild) {
			best, bestBuild = version, n
		}
	}
	if best == "" {
		return "", fmt.Errorf("no GA Temurin %d release in index", feature)
	}
	return fmt.Sprintf("%s+%d", best, bestBuild), nil
}

func newestJavaLTS(ctx context.Context, f fetcher, url string) (int, error) {
	body, err := f.get(ctx, url)
	if err != nil {
		return 0, err
	}
	var doc struct {
		MostRecentLTS int `json:"most_recent_lts"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return 0, err
	}
	return doc.MostRecentLTS, nil
}

func latestGo(ctx context.Context, f fetcher, url string) (string, error) {
	body, err := f.get(ctx, url)
	if err != nil {
		return "", err
	}
	var releases []struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
	}
	if err := json.Unmarshal(body, &releases); err != nil {
		return "", err
	}
	var versions []string
	for _, release := range releases {
		if release.Stable {
			versions = append(versions, strings.TrimPrefix(release.Version, "go"))
		}
	}
	return maxVersion(versions)
}
