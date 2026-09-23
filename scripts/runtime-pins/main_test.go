package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/runtimeclosure"
)

// Unit fixtures are trimmed copies of each real index's shape.
var indexFixtures = map[string]string{
	"/otp": "OTP-29.1.1 : asn1-5.5.2 compiler-10.0.6 # common_test-1.31.2 erts-17.1 kernel-11.0.4 :\n" +
		"OTP-28.5.0.7 : ssl-11.4.1 # erts-16.3.0.2 kernel-10.4 :\n" +
		"OTP-29.1 : erts-17.1 kernel-11.0.4 # asn1-5.5.1 :\n" +
		"OTP-29.0.6 : compiler-10.0.4 erts-17.0.6 # asn1-5.5 :\n",
	"/elixir": "v1.19.6 aaa 2026-08-28T10:41:36Z sha\n" +
		"v1.20.4 bbb 2026-08-28T10:41:23Z sha\n" +
		"v1.21.0-rc.0 ccc 2026-09-01T00:00:00Z sha\n" +
		"main ddd 2026-09-20T00:00:00Z sha\n",
	"/node":       `[{"version":"v26.10.0","lts":false},{"version":"v26.9.0","lts":false},{"version":"v24.12.0","lts":"Krypton"}]`,
	"/typescript": `{"name":"typescript","version":"7.0.2"}`,
	"/python":     `[{"name":"Python 3.14.7"},{"name":"Python 3.13.12"},{"name":"Python 3.14.10"},{"name":"Python 3.15.0rc2"}]`,
	"/java/21":    `{"releases":["jdk-21.0.12.1+1","jdk-21.0.12+8","jdk-21.0.11+10"]}`,
	"/java/info":  `{"available_lts_releases":[8,11,17,21,25],"most_recent_lts":25}`,
	"/go":         `[{"version":"go1.27.1","stable":true},{"version":"go1.28rc1","stable":false},{"version":"go1.26.8","stable":true}]`,
}

func fixtureServer(t *testing.T) (*httptest.Server, sources) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/java" {
			path = "/java/" + strings.TrimPrefix(strings.Split(r.URL.Query().Get("version"), ",")[0], "[")
		}
		body, ok := indexFixtures[path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server, sources{
		OTPTable:       server.URL + "/otp",
		ElixirBuilds:   server.URL + "/elixir",
		NodeIndex:      server.URL + "/node",
		TypeScript:     server.URL + "/typescript",
		PythonReleases: server.URL + "/python",
		JavaReleases:   server.URL + "/java?version=%%5B%d,%d%%29",
		JavaInfo:       server.URL + "/java/info",
		GoReleases:     server.URL + "/go",
	}
}

func testFetcher() fetcher { return fetcher{client: &http.Client{Timeout: 5 * time.Second}} }

func TestLatestParsersPickNewestRelease(t *testing.T) {
	_, src := fixtureServer(t)
	ctx, f := context.Background(), testFetcher()
	otp, erts, err := latestOTP(ctx, f, src.OTPTable)
	if err != nil || otp != "29.1.1" || erts != "17.1" {
		t.Fatalf("OTP = %s/%s (%v), want 29.1.1/17.1", otp, erts, err)
	}
	checks := []struct {
		name string
		got  func() (string, error)
		want string
	}{
		{"elixir skips rc and branch builds", func() (string, error) { return latestElixir(ctx, f, src.ElixirBuilds) }, "1.20.4"},
		{"node", func() (string, error) { return latestNode(ctx, f, src.NodeIndex) }, "26.10.0"},
		{"typescript", func() (string, error) { return latestTypeScript(ctx, f, src.TypeScript) }, "7.0.2"},
		{"python compares numerically and skips rc", func() (string, error) { return latestPython(ctx, f, src.PythonReleases) }, "3.14.10"},
		{"java keeps the pin spelling", func() (string, error) { return latestJava(ctx, f, src.JavaReleases, 21) }, "21.0.12.1+1"},
		{"go skips unstable", func() (string, error) { return latestGo(ctx, f, src.GoReleases) }, "1.27.1"},
	}
	for _, check := range checks {
		got, err := check.got()
		if err != nil || got != check.want {
			t.Errorf("%s = %q (%v), want %q", check.name, got, err, check.want)
		}
	}
	if lts, err := newestJavaLTS(ctx, f, src.JavaInfo); err != nil || lts != 25 {
		t.Errorf("newest LTS = %d (%v), want 25", lts, err)
	}
}

func TestFillLatestReportsDriftAndNewerLTS(t *testing.T) {
	_, src := fixtureServer(t)
	rows := []row{{Runtime: "Node", Pinned: "26.9.0"}, {Runtime: "Python", Pinned: "3.14.10"}, {Runtime: "Java", Pinned: runtimeclosure.PinnedJavaRuntimeVersion}}
	notes := fillLatest(context.Background(), rows, testFetcher(), src)
	if rows[0].Latest != "26.10.0" || rows[0].Status() != "drift" {
		t.Errorf("Node row %+v is not drift against 26.10.0", rows[0])
	}
	if rows[1].Status() != "ok" {
		t.Errorf("Python row %+v is not ok", rows[1])
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "newest Temurin LTS is 25") {
		t.Errorf("notes %q do not report the newer Java LTS", notes)
	}
	if exitCode(rows, false) != 1 {
		t.Error("drift must exit 1")
	}
}

// TestUnreachableIndexIsOffline proves the report still renders, row by
// row, when no index answers.
func TestUnreachableIndexIsOffline(t *testing.T) {
	server, src := fixtureServer(t)
	server.Close()
	rows := pinned()
	fillLatest(context.Background(), rows, testFetcher(), src)
	for _, r := range rows {
		if r.Latest != offline || r.Status() != "unknown" {
			t.Errorf("row %+v is not offline", r)
		}
	}
	if exitCode(rows, false) != 0 {
		t.Error("offline without -strict must exit 0")
	}
	if exitCode(rows, true) != 2 {
		t.Error("offline under -strict must exit 2")
	}
	table := render(rows, nil)
	if !strings.Contains(table, "| Node | "+runtimeclosure.RequiredNodeRelease+" |  | offline | unknown |") {
		t.Errorf("offline table does not carry the offline column:\n%s", table)
	}
}

func TestPinnedRowsCoverEveryRuntime(t *testing.T) {
	want := []string{"Erlang/OTP", "erts", "Elixir", "Node", "TypeScript", "Python", "Java", "Go"}
	rows := pinned()
	if len(rows) != len(want) {
		t.Fatalf("rows %+v do not cover %v", rows, want)
	}
	probes := hostProbes()
	for i, r := range rows {
		if r.Runtime != want[i] || r.Pinned == "" {
			t.Errorf("row %d = %+v, want runtime %s with a pin", i, r, want[i])
		}
		if _, ok := probes[r.Runtime]; !ok {
			t.Errorf("runtime %s has no host probe", r.Runtime)
		}
	}
}

func TestExtractVersionFromHostOutput(t *testing.T) {
	probes := hostProbes()
	cases := map[string]struct{ runtime, out, want string }{
		"otp":           {"Erlang/OTP", "OTP=29.1.1 ERTS=17.1\n", "29.1.1"},
		"erts":          {"erts", "OTP=29.1.1 ERTS=17.1\n", "17.1"},
		"elixir":        {"Elixir", "Erlang/OTP 29 [erts-17.1]\n\nElixir 1.20.4 (compiled with Erlang/OTP 29)\n", "1.20.4"},
		"node":          {"Node", "v26.9.0\n", "26.9.0"},
		"tsc":           {"TypeScript", "Version 7.0.2\n", "7.0.2"},
		"python":        {"Python", "Python 3.14.7\n", "3.14.7"},
		"temurin":       {"Java", "openjdk version \"21.0.12.1\" 2026-08-18 LTS\nOpenJDK Runtime Environment Temurin-21.0.12.1+1 (build 21.0.12.1+1-LTS)\n", "21.0.12.1+1"},
		"other openjdk": {"Java", "openjdk version \"21.0.12.1\" 2026-08-18\nOpenJDK Runtime Environment Homebrew (build 21.0.12.1)\n", "21.0.12.1"},
		"go":            {"Go", "go1.27.1\n", "1.27.1"},
		"garbage":       {"Node", "command not found\n", ""},
	}
	for name, c := range cases {
		if got := extractVersion(probes[c.runtime].extract, []byte(c.out)); got != c.want {
			t.Errorf("%s: extracted %q, want %q", name, got, c.want)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
	}{
		{"29.1.1", "29.1", 1}, {"3.14.10", "3.14.7", 1}, {"26.9.0", "26.10.0", -1}, {"1.20.4", "1.20.4", 0},
	} {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Errorf("compare(%s, %s) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
