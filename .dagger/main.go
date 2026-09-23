// Machinery CI as a Dagger module.
//
// Every function here is one hosted job from .github/workflows, expressed so
// it runs identically on a laptop and on a runner. The runtime pins are NOT
// duplicated: the polyglot toolchain is built from scripts/ci-linux.dockerfile,
// which remains the single owner of Go, Node, TypeScript, CPython and
// Elixir/OTP identities and is shared with scripts/ci-linux.sh.
//
// What this module does not cover: the macOS-only jobs (native-tests,
// golden-native) and the GitHub-native dependency-review job. Containers are
// Linux; a darwin runtime cannot be reproduced here and pretending otherwise
// would be worse than the honest gap.
package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"dagger/machinery/internal/dagger"
)

// Pinned tool identities that live outside the container image. Versions the
// repository already owns in a file are read from that file at run time
// instead of being repeated here.
const (
	dindImage       = "docker:29.7.2-dind"
	goTestTimeout   = "30m"
	sharedTmpPath   = "/shared"
	gitleaksImage   = "zricethezav/gitleaks:v8.30.0"
	ciUser          = "ci"
	shellcheckPlatf = dagger.Platform("linux/amd64")
)

type Machinery struct {
	// The repository worktree under test.
	Source *dagger.Directory
	// Platform every job container runs on. Empty means the host's own
	// architecture, which is what makes a local run fast.
	Platform string
	// Extra BEAM flags forwarded into every job container.
	ErlFlags string
}

func New(
	// The repository worktree. Only paths git already ignores are dropped:
	// .bin holds a host-built binary every job rebuilds for itself, and
	// .vault/issues is tracker state. The rest of .vault stays, because
	// tracked files under it are part of the worktree the formal gate proves
	// clean, and excluding them would report them as deletions. .claude holds
	// agent-session state (worktrees, the sidecar file) that is never tracked
	// and can hold whole extra checkouts, which would breach the
	// tree-inventory snapshot bound.
	// +defaultPath="/"
	// +ignore=[".bin", ".dagger/internal", ".vault/issues", ".claude"]
	source *dagger.Directory,
	// Target platform, for example linux/amd64 to mirror the hosted runners
	// exactly. Defaults to the host architecture.
	//
	// The assurance catalog pins linux/amd64 and darwin/arm64 as the only
	// native assurance platforms, so the integration lane fails closed
	// anywhere else. On an arm64 host that means Test and
	// IntegrationRequired need this set to linux/amd64, and then they run
	// under emulation.
	// +optional
	platform string,
	// Extra BEAM flags, forwarded the way scripts/ci-linux.sh forwards them.
	// Under emulation the BEAM's JIT traps, and "+JMsingle true" is what
	// makes an emulated Elixir suite run at all.
	// +optional
	erlFlags string,
) *Machinery {
	return &Machinery{Source: source, Platform: platform, ErlFlags: erlFlags}
}

func (m *Machinery) platform() dagger.Platform { return dagger.Platform(m.Platform) }

// Base is the pinned polyglot toolchain: the exact runtime identities the
// assurance catalog requires, built from the Dockerfile that already owns
// them. Cache volumes keep the Go module and build caches warm across runs.
func (m *Machinery) Base() *dagger.Container { return m.baseAs("") }

// baseAs builds the toolchain container owned by user, or by root when user
// is empty.
//
// Root is wrong for the test sweep: the hosted runner executes as an
// unprivileged user, and root bypasses permission bits, so custody tests that
// assert an unwritable path is refused silently pass instead
// (TestStorePermissionCustody is the one that catches it). Running the sweep
// as a real unprivileged user keeps that coverage rather than skipping it.
func (m *Machinery) baseAs(user string) *dagger.Container {
	img := m.Source.
		DockerBuild(dagger.DirectoryDockerBuildOpts{
			Dockerfile: "scripts/ci-linux.dockerfile",
			Platform:   m.platform(),
		})
	cacheOpts := dagger.ContainerWithMountedCacheOpts{}
	dirOpts := dagger.ContainerWithMountedDirectoryOpts{}
	if user != "" {
		img = img.WithExec([]string{"bash", "-c",
			"id -u " + user + " >/dev/null 2>&1 || useradd -u 1000 -m -s /bin/bash " + user})
		cacheOpts.Owner = user
		dirOpts.Owner = user
	}
	c := img.
		WithEnvVariable("GOCACHE", "/gocache").
		WithEnvVariable("GOMODCACHE", "/gomodcache").
		WithMountedCache("/gocache", dag.CacheVolume("machinery-gocache"), cacheOpts).
		WithMountedCache("/gomodcache", dag.CacheVolume("machinery-gomodcache"), cacheOpts).
		WithMountedDirectory("/src", m.Source, dirOpts).
		WithWorkdir("/src")
	if m.ErlFlags != "" {
		c = c.WithEnvVariable("ERL_FLAGS", m.ErlFlags)
	}
	if user != "" {
		c = c.WithUser(user)
	}
	return c
}

// sharedTmp is one directory mounted at the same absolute path in both the
// job container and the daemon.
//
// Checkers and the integration lane bind-mount their own scratch directories
// into checker containers, and those mounts are resolved by the DAEMON. A
// temp path that exists only inside the job container cannot be mounted
// ("bind source path does not exist"). scripts/ci-linux.sh solves this the
// same way for the same reason.
func (m *Machinery) sharedTmp() *dagger.CacheVolume {
	return dag.CacheVolume("machinery-shared-tmp")
}

// dockerd is a throwaway Docker daemon for the jobs that provision a real OCI
// runtime. The hosted runners get a daemon from the image; here it is an
// explicit service binding, which is the same contract stated out loud.
func (m *Machinery) dockerd() *dagger.Service {
	return dag.Container().
		From(dindImage).
		WithMountedCache("/var/lib/docker", dag.CacheVolume("machinery-dind"),
			dagger.ContainerWithMountedCacheOpts{Sharing: dagger.CacheSharingModePrivate}).
		// Same owner as the job side mounts it with: both sides must see the
		// identical directory, or a scratch path the job creates is not the
		// path the daemon resolves for a bind mount.
		WithMountedCache(sharedTmpPath, m.sharedTmp(),
			dagger.ContainerWithMountedCacheOpts{Owner: "1000:1000"}).
		// Empty means "serve plain TCP"; the entrypoint otherwise generates
		// a CA and refuses the unencrypted port.
		WithEnvVariable("DOCKER_TLS_CERTDIR", "").
		WithExposedPort(2375).
		AsService(dagger.ContainerAsServiceOpts{
			// Start through the image's own entrypoint rather than exec'ing
			// dockerd directly: it prepares the nested cgroup the daemon
			// needs. Without it every checker container dies on "cannot
			// enter cgroupv2 ... it is in threaded mode".
			//
			// --feature containerd-snapshotter=false keeps the classic image
			// store, where `docker image inspect` reports .Os and
			// .Architecture at the top level the way the hosted runners do.
			// The containerd store leaves both empty and would fail
			// machinery's OCI platform check for a reason that has nothing
			// to do with the code under test.
			UseEntrypoint: true,
			Args: []string{"--host=tcp://0.0.0.0:2375", "--tls=false",
				"--feature", "containerd-snapshotter=false"},
			InsecureRootCapabilities: true,
		})
}

// withDocker binds the daemon a job needs to pull and run pinned images, and
// puts the job's temp directory on the one path both sides can resolve.
//
// user is the account the job runs as ("" keeps root). The job directory is
// created as root and handed to uid 1000, because the cache volume persists
// across runs and its parent directory may already belong to root from an
// earlier job or daemon; an unprivileged mkdir there fails with EACCES and
// the job dies before its first test.
func (m *Machinery) withDocker(c *dagger.Container, job, user string) *dagger.Container {
	dir := sharedTmpPath + "/tmp/" + job
	c = c.
		WithServiceBinding("docker", m.dockerd()).
		WithEnvVariable("DOCKER_HOST", "tcp://docker:2375").
		// uid 1000 so an unprivileged job can write here; root is not
		// blocked by a 1000-owned directory.
		WithMountedCache(sharedTmpPath, m.sharedTmp(),
			dagger.ContainerWithMountedCacheOpts{Owner: "1000:1000"}).
		WithUser("root").
		WithExec([]string{"bash", "-euo", "pipefail", "-c",
			"mkdir -p " + dir + " && chown 1000:1000 " + sharedTmpPath + "/tmp " + dir}).
		WithEnvVariable("TMPDIR", dir)
	if user != "" {
		c = c.WithUser(user)
	}
	return c
}

// shInShared runs a script from a copy of the worktree on the shared path.
//
// Some scratch directories are created inside the repository root rather than
// under TMPDIR (the OCI isolation probe is one), and those are bind-mounted
// into checker containers too. Working from a copy on the shared volume makes
// every path under the tree resolvable by the daemon, which is what a hosted
// runner gets for free by owning one filesystem. The per-job subdirectory
// keeps concurrent jobs from sharing a tree.
func (m *Machinery) shInShared(ctx context.Context, c *dagger.Container, job, script string) (string, error) {
	dir := sharedTmpPath + "/work/" + job
	return sh(ctx, c, "rm -rf "+dir+"\nmkdir -p "+dir+"\ncp -a /src/. "+dir+"/\ncd "+dir+"\n"+script)
}

// sh runs one bash script under the strict shell settings every hosted step
// uses, and returns its combined output.
func sh(ctx context.Context, c *dagger.Container, script string) (string, error) {
	return c.
		WithExec([]string{"bash", "-euo", "pipefail", "-c", script}).
		Stdout(ctx)
}

// shInCopy runs a script against a runtime copy of the worktree.
//
// Overlayfs refuses renameat2(RENAME_NOREPLACE) on a directory that still
// lives in a lower layer, and that is exactly what the Modelith and formal
// transactions do when they park a corpus before publishing a new one. A
// hosted runner works on a real filesystem where the atomic rename succeeds.
// Copying the tree inside the exec puts it in the container's own writable
// layer and restores that behavior, instead of weakening the transaction to
// suit the sandbox.
func shInCopy(ctx context.Context, c *dagger.Container, script string) (string, error) {
	return sh(ctx, c, "cp -a /src /work\ncd /work\n"+script)
}

// ---------------------------------------------------------------- build ---

// Build cross-compiles every release target, statically and with CGO off, and
// smoke-tests the one target that can execute here.
func (m *Machinery) Build(ctx context.Context) (string, error) {
	targets := []struct{ goos, goarch string }{
		{"linux", "amd64"}, {"linux", "arm64"},
		{"darwin", "arm64"}, {"windows", "amd64"},
	}
	c := m.Base()
	var b strings.Builder
	for _, t := range targets {
		out, err := sh(ctx, c.
			WithEnvVariable("GOOS", t.goos).
			WithEnvVariable("GOARCH", t.goarch).
			WithEnvVariable("CGO_ENABLED", "0"),
			`go build -trimpath -ldflags "-s -w -X main.version=ci" -o /tmp/machinery ./cmd/machinery && echo "built `+t.goos+`/`+t.goarch+`"`)
		b.WriteString(out)
		if err != nil {
			return b.String(), fmt.Errorf("build %s/%s: %w", t.goos, t.goarch, err)
		}
	}
	out, err := sh(ctx, c, `go build -trimpath -ldflags "-s -w -X main.version=ci" -o /tmp/machinery ./cmd/machinery && /tmp/machinery version`)
	b.WriteString(out)
	return b.String(), err
}

// ----------------------------------------------------------------- lint ---

// Lint runs the formatting gate, the stdlib analyzer, actionlint, ShellCheck
// over the closed shell corpus, and the pinned golangci-lint set.
func (m *Machinery) Lint(ctx context.Context) (string, error) {
	var b strings.Builder

	out, err := sh(ctx, m.Base(), `
unformatted=$(gofmt -l cmd/ internal/)
if [ -n "$unformatted" ]; then echo "not gofmt-clean:"; echo "$unformatted"; exit 1; fi
echo "gofmt clean"
go vet ./...
echo "go vet clean"
version=$(cat .actionlint-version)
go install github.com/rhysd/actionlint/cmd/actionlint@"$version"
"$(go env GOPATH)"/bin/actionlint .github/workflows/*.yml
echo "actionlint clean"
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@"$(cat .golangci-version)"
"$(go env GOPATH)"/bin/golangci-lint run --config .golangci.yml --timeout 5m
echo "golangci-lint clean"`)
	b.WriteString(out)
	if err != nil {
		return b.String(), err
	}

	// The closed shell corpus is produced by a Go helper, so the inventory is
	// taken in the toolchain container and only the resulting list crosses
	// into the ShellCheck container.
	inventory, err := sh(ctx, m.Base(), `scripts/shellcheck-inventory.sh`)
	if err != nil {
		b.WriteString(inventory)
		return b.String(), err
	}

	// ShellCheck ships one checksum in this repository, for linux x86_64.
	// Run that exact pinned binary on that exact platform rather than
	// widening the trust root to a second architecture.
	out, err = sh(ctx, m.shellcheckBase().
		WithNewFile("/tmp/shellcheck-files", inventory), `
version=$(cat .shellcheck-version)
checksum=$(cat .shellcheck-linux-x86_64.sha256)
install_dir=/tmp/shellcheck
curl -fsSL "https://github.com/koalaman/shellcheck/releases/download/v${version}/shellcheck-v${version}.linux.x86_64.tar.xz" -o /tmp/shellcheck.tar.xz
echo "$checksum  /tmp/shellcheck.tar.xz" | sha256sum -c -
mkdir -p "$install_dir"
tar -xJf /tmp/shellcheck.tar.xz --strip-components=1 -C "$install_dir"
test "$("$install_dir/shellcheck" --version | awk '$1 == "version:" {print $2}')" = "$version"
# The same Bash 3.2-safe read loop scripts/preflight-fast.sh uses: the
# array-reading builtin it replaces does not exist in the stock macOS Bash,
# and the two mirrors keep one shape.
shell_files=()
while IFS= read -r shell_file; do
  shell_files+=("$shell_file")
done </tmp/shellcheck-files
test "${#shell_files[@]}" -gt 0
"$install_dir/shellcheck" "${shell_files[@]}"
echo "shellcheck clean (${#shell_files[@]} files)"`)
	b.WriteString(out)
	return b.String(), err
}

// shellcheckBase is a minimal amd64 container for the one pinned x86_64 tool.
func (m *Machinery) shellcheckBase() *dagger.Container {
	return dag.Container(dagger.ContainerOpts{Platform: shellcheckPlatf}).
		From("debian:trixie-slim").
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "-y", "--no-install-recommends",
			"ca-certificates", "curl", "xz-utils", "coreutils", "gawk", "git"}).
		WithMountedDirectory("/src", m.Source).
		WithWorkdir("/src")
}

// ----------------------------------------------------------------- tidy ---

// Tidy proves go.mod and go.sum are tidy and in sync.
func (m *Machinery) Tidy(ctx context.Context) (string, error) {
	return sh(ctx, m.Base(), `
go mod tidy
go run ./scripts/git-safe -root . -- diff --exit-code -- go.mod go.sum
echo "go.mod and go.sum tidy"`)
}

// ----------------------------------------------------------------- test ---

// Test is the full race sweep over every package: the hosted ci test job.
func (m *Machinery) Test(ctx context.Context) (string, error) {
	return m.shInShared(ctx, m.withDocker(m.baseAs(ciUser), "test", ciUser), "test",
		`go test -race -count=1 ./... -timeout=`+goTestTimeout)
}

// Golden runs the byte-for-byte corpus and the adversarial gate-experiment
// suite.
func (m *Machinery) Golden(ctx context.Context) (string, error) {
	return sh(ctx, m.Base(), `
go test -count=1 -run TestGolden ./cmd/machinery
go test -count=1 ./internal/experiments/`)
}

// Gates runs the deterministic gate suite over every registered example: the
// dogfood job, where machinery checks its own reference designs.
func (m *Machinery) Gates(ctx context.Context) (string, error) {
	return sh(ctx, m.Base(), `
make build
scripts/example-gates.sh .bin/machinery`)
}

// ExampleImpls runs the hermetic suite of every registered implementation
// module.
func (m *Machinery) ExampleImpls(ctx context.Context) (string, error) {
	return sh(ctx, m.Base(), `
scripts/example-inventory.sh impl-modules | while IFS=$'\t' read -r design impl module; do
  case "$module" in
    go) (cd "$impl" && go test ./... -count=1) ;;
    *) echo "unsupported implementation module type $module: $design" >&2; exit 1 ;;
  esac
done
echo "every registered implementation module passed"`)
}

// ----------------------------------------------------------------- docs ---

// Docs is the documentation surface gate: no whitespace errors in the
// aggregate change, no stale toolchain references, and no em dashes.
func (m *Machinery) Docs(
	ctx context.Context,
	// Base revision the aggregate change is measured against.
	// +optional
	// +default="HEAD~1"
	baseRef string,
) (string, error) {
	return sh(ctx, m.Base(), `
# github.event.before is all-zeros on the first push to a new branch, and can
# name a commit this checkout does not have after a force push. Either way the
# aggregate change is measured against the parent.
base="`+baseRef+`"
if [ -z "$base" ] || [ "$base" = "0000000000000000000000000000000000000000" ] \
   || ! git cat-file -e "${base}^{commit}" 2>/dev/null; then
  base="HEAD~1"
fi
go run ./scripts/git-safe -root . -- diff --check "$base"..HEAD
echo "no whitespace errors"

grep_status=0
grep -rnE "PyYAML|pyyaml|uv run|oracle_gen\.py|machine_lint\.py|machinery_check\.py|tla_gen\.py|refine_gen\.py|compose_gen\.py|diff-all\.sh|capture-golden\.sh" \
  README.md skills/ agents/ docs/ Makefile || grep_status=$?
case "$grep_status" in
  0) echo "stale Python-toolchain reference found" >&2; exit 1 ;;
  1) ;;
  *) echo "stale Python-toolchain scan failed with status $grep_status" >&2; exit 1 ;;
esac

grep_status=0
grep -rnEi "Souffl(e|é).*(external.checker|checker engine|CI pin|required)|external.checker.*Souffl(e|é)" \
  README.md docs/ examples/pii-flow/README.md || grep_status=$?
case "$grep_status" in
  0) echo "stale host checker-runtime contract found" >&2; exit 1 ;;
  1) ;;
  *) echo "host checker-runtime scan failed with status $grep_status" >&2; exit 1 ;;
esac
echo "doc surface clean"

grep_status=0
em_dash=$(printf '\342\200\224')
grep -rn "$em_dash" README.md CONTRIBUTING.md install.sh skills/ agents/ docs/ examples/ commands/ adapters/ hooks/ Makefile .github/ || grep_status=$?
case "$grep_status" in
  0) echo "em dash found; house style forbids it" >&2; exit 1 ;;
  1) ;;
  *) echo "em-dash scan failed with status $grep_status" >&2; exit 1 ;;
esac
echo "no em dashes"`)
}

// ModelithRender regenerates every committed Modelith render with the pinned
// engine and rejects byte drift.
func (m *Machinery) ModelithRender(ctx context.Context) (string, error) {
	return shInCopy(ctx, m.Base(), `
go install github.com/stacklok/modelith/cmd/modelith@v0.4.0
export PATH="$(go env GOPATH)/bin:$PATH"
make modelith-render-check`)
}

// ------------------------------------------------------- external engines ---

// DesignEngines exercises the two explicit engine halves: C4 grammar
// compilation and the reference external checker, including the real pinned
// OCI checker runtime.
func (m *Machinery) DesignEngines(ctx context.Context) (string, error) {
	return m.shInShared(ctx, m.withDocker(m.Base(), "design-engines", ""), "design-engines", `
make build
test -s .java-runtime-pin
test -s .structurizr-pin

c4_inventory=$(mktemp)
trap 'rm -f "$c4_inventory"' EXIT
scripts/c4-inventory.sh examples >"$c4_inventory"
test -s "$c4_inventory"
while IFS= read -r dsl; do
  .bin/machinery verify-c4 "$(dirname "$dsl")"
done <"$c4_inventory"
echo "every example workspace.dsl compiled"

runner=$(mktemp -d)/run-safe
go build -o "$runner" ./scripts/run-safe
image=python@sha256:c6ead215bfd31f1e433d968853b7a769989117115b728874824e6c0a27cb96fc
platform=linux/amd64
pull_receipt=$("$runner" -timeout 10m -stdout-limit 4096 -stderr-limit 65536 -- \
  docker pull --quiet --platform "$platform" "$image")
case "$pull_receipt" in
  "$image"|"docker.io/library/$image") ;;
  *) echo "Docker pull returned a non-canonical image receipt: $pull_receipt" >&2; exit 1 ;;
esac
inspect=$("$runner" -timeout 30s -stdout-limit 65536 -stderr-limit 4096 -- \
  docker image inspect --format '{{json .RepoDigests}} {{.Os}}/{{.Architecture}}' "$image")
case "$inspect" in
  *\"$image\"*" $platform") ;;
  *) echo "local OCI identity does not match $image on $platform: $inspect" >&2; exit 1 ;;
esac
"$runner" -timeout 2m -stdout-limit 4096 -stderr-limit 4096 -- \
  docker run --rm --pull=never --platform "$platform" --network=none --read-only "$image" python3 --version
echo "pinned external-checker OCI runtime provisioned"

# The pii-flow reference runs its rules under Souffle: rebuild that image
# reproducibly and refuse any digest other than the registry's pin.
scripts/pii-flow-image.sh

MACHINERY_REQUIRE_OCI_GOLDEN=1 go test -count=1 -run '^(TestVerifyCheckersPiiFlowEngineGolden|TestPiiFlowSouffleVerdicts)$' ./cmd/machinery

scripts/example-inventory.sh checkers | while IFS=$'\t' read -r design registry; do
  .bin/machinery verify-checkers "$design" --registry "$registry"
done
echo "every registered external checker re-ran"`)
}

// IntegrationEvidence runs the required native runtime integration lane and
// returns its evidence directory whatever the outcome, with the lane's exit
// status in `exit` and its console output in `console.txt`.
//
// A lane failure that only a runner reproduces is undiagnosable from the log
// alone, which is why the hosted job retains this directory. A plain failing
// exec would abort the pipeline and take the evidence with it, so the lane
// runs without -e here and the verdict travels inside the directory.
func (m *Machinery) IntegrationEvidence() *dagger.Directory {
	return m.withDocker(m.Base(), "integration-required", "").
		WithEnvVariable("MACHINERY_INTEGRATION_REPORT_DIR", "/evidence").
		WithExec([]string{"bash", "-c", `
set +e
mkdir -p /evidence
rm -rf ` + sharedTmpPath + `/work/integration-required
mkdir -p ` + sharedTmpPath + `/work/integration-required
cp -a /src/. ` + sharedTmpPath + `/work/integration-required/
cd ` + sharedTmpPath + `/work/integration-required
go run ./scripts/integration-lane --lane required 2>&1 | tee /evidence/console.txt
echo "${PIPESTATUS[0]}" > /evidence/exit
exit 0`}).
		Directory("/evidence")
}

// IntegrationRequired runs the required native runtime integration lane: the
// four-language assurance probes against the exact pinned runtime identities.
func (m *Machinery) IntegrationRequired(ctx context.Context) (string, error) {
	evidence := m.IntegrationEvidence()
	console, err := evidence.File("console.txt").Contents(ctx)
	if err != nil {
		return "", err
	}
	status, err := evidence.File("exit").Contents(ctx)
	if err != nil {
		return console, err
	}
	if strings.TrimSpace(status) != "0" {
		return console, fmt.Errorf("required integration lane failed (exit %s)", strings.TrimSpace(status))
	}
	return console, nil
}

// VerifyFormal regenerates and TLC-checks every registered formal example,
// then proves generation left no diff.
func (m *Machinery) VerifyFormal(ctx context.Context) (string, error) {
	return shInCopy(ctx, m.Base(), `
make build
scripts/example-inventory.sh formal | while IFS= read -r design; do
  echo "== $design =="
  .bin/machinery verify-formal "$design"
done
status=$(go run ./scripts/git-safe -root . -- status --porcelain --untracked-files=all)
if [ -n "$status" ]; then
  echo "$status"
  echo "formal verification regenerated tracked content or emitted an untracked artifact" >&2
  exit 1
fi
echo "formal suite clean, no diff"`)
}

// soufflePlatform is the one platform the upstream Soufflé release is built
// for; on an arm64 host the parity lane runs under emulation.
const soufflePlatform = dagger.Platform("linux/amd64")

// DatalogParity is the hosted datalog-parity job: the Soufflé parity tests of
// internal/datalog and of the shipped Gy-rules rule files, run in the image
// scripts/souffle.dockerfile builds from pinned inputs (base and Go images by
// digest, the Soufflé release asset by sha256, dependencies from a fixed
// Ubuntu snapshot). scripts/datalog-parity.sh fails when a test skipped its
// Soufflé half.
func (m *Machinery) DatalogParity(ctx context.Context) (string, error) {
	c := m.Source.
		DockerBuild(dagger.DirectoryDockerBuildOpts{
			Dockerfile: "scripts/souffle.dockerfile",
			Platform:   soufflePlatform,
		}).
		WithEnvVariable("GOCACHE", "/gocache").
		WithEnvVariable("GOMODCACHE", "/gomodcache").
		WithMountedCache("/gocache", dag.CacheVolume("machinery-gocache-souffle")).
		WithMountedCache("/gomodcache", dag.CacheVolume("machinery-gomodcache")).
		WithMountedDirectory("/src", m.Source).
		WithWorkdir("/src")
	return sh(ctx, c, "scripts/datalog-parity.sh")
}

// Govulncheck scans the toolchain and every registered example module.
func (m *Machinery) Govulncheck(ctx context.Context) (string, error) {
	return sh(ctx, m.Base(), `
go install golang.org/x/vuln/cmd/govulncheck@v1.5.0
export PATH="$(go env GOPATH)/bin:$PATH"
govulncheck ./...
scripts/example-inventory.sh security | while IFS=$'\t' read -r impl module; do
  case "$module" in
    go) (cd "$impl" && govulncheck ./...) ;;
    *) echo "unsupported security module type $module: $impl" >&2; exit 1 ;;
  esac
done
echo "govulncheck clean"`)
}

// Gitleaks scans the full history and the working tree for secrets. The
// hosted job uses the vendor's GitHub action; this is the same scanner from
// the vendor's own pinned image.
func (m *Machinery) Gitleaks(ctx context.Context) (string, error) {
	return sh(ctx, dag.Container(dagger.ContainerOpts{Platform: m.platform()}).
		From(gitleaksImage).
		WithMountedDirectory("/src", m.Source).
		WithWorkdir("/src"),
		`gitleaks detect --source . --redact --verbose --no-banner`)
}

// ------------------------------------------------------------- nightly ---

// RegenCleanTree regenerates every committed artifact with the pinned
// generators and proves the tree is unchanged: the canary for a generator
// whose output drifted from what is committed.
func (m *Machinery) RegenCleanTree(ctx context.Context) (string, error) {
	return shInCopy(ctx, m.Base(), `
go install github.com/stacklok/modelith/cmd/modelith@v0.4.0
export PATH="$(go env GOPATH)/bin:$PATH"
make modelith-render
make build
scripts/example-inventory.sh formal | while IFS= read -r design; do
  .bin/machinery oracle "$design/machines"
done
scripts/example-inventory.sh pack-parents | while IFS= read -r design; do
  .bin/machinery pack generate "$design"
done
# verify-formal --gen-only runs the exact generator orchestration the checked
# run uses (tla + refine + compose, coordinator parsed as YAML), with TLC
# skipped so this stays Java-free.
scripts/example-inventory.sh formal | while IFS= read -r d; do
  .bin/machinery verify-formal --gen-only "$d" || exit 1
done
scripts/example-inventory.sh pack-children | while IFS=$'\t' read -r design parent; do
  test -n "$parent"
  .bin/machinery pack refine "$design"
done
status=$(go run ./scripts/git-safe -root . -- status --porcelain)
if [ -n "$status" ]; then
  echo "$status"
  echo "regeneration dirtied the tree (a stale tracked artifact or a newly emitted file)" >&2
  exit 1
fi
echo "regeneration left the tree clean"`)
}

// GoldenNightly runs the full non-race suite and the byte corpus on a cadence:
// the canary for environment-driven divergence, such as a Go toolchain update
// changing generated output.
func (m *Machinery) GoldenNightly(ctx context.Context) (string, error) {
	return m.shInShared(ctx, m.withDocker(m.baseAs(ciUser), "golden-nightly", ciUser), "golden-nightly", `
go test -count=1 ./... -timeout=20m
go test -count=1 -run TestGolden ./cmd/machinery`)
}

// ------------------------------------------------------------------ all ---

type jobResult struct {
	name string
	took time.Duration
	err  error
}

// Ci runs every containerizable job concurrently and reports one summary. A
// single failure fails the run, and no job hides another: every verdict is
// reported.
func (m *Machinery) Ci(ctx context.Context) (string, error) {
	jobs := []struct {
		name string
		run  func(context.Context) (string, error)
	}{
		{"build", m.Build},
		{"lint", m.Lint},
		{"tidy", m.Tidy},
		{"golden", m.Golden},
		{"gates", m.Gates},
		{"example-impls", m.ExampleImpls},
		{"docs", func(ctx context.Context) (string, error) { return m.Docs(ctx, "HEAD~1") }},
		{"modelith-render", m.ModelithRender},
		{"design-engines", m.DesignEngines},
		{"datalog-parity", m.DatalogParity},
		{"verify-formal", m.VerifyFormal},
		{"govulncheck", m.Govulncheck},
		{"gitleaks", m.Gitleaks},
		{"integration-required", m.IntegrationRequired},
		{"test", m.Test},
	}

	results := make([]jobResult, len(jobs))
	var wg sync.WaitGroup
	for i, job := range jobs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			_, err := job.run(ctx)
			results[i] = jobResult{name: job.name, took: time.Since(start), err: err}
		}()
	}
	wg.Wait()

	var b strings.Builder
	failed := 0
	for _, r := range results {
		verdict := "ok"
		if r.err != nil {
			verdict = "FAILED"
			failed++
		}
		fmt.Fprintf(&b, "%-22s %-7s %s\n", r.name, verdict, r.took.Round(time.Second))
	}
	for _, r := range results {
		if r.err != nil {
			fmt.Fprintf(&b, "\n--- %s ---\n%v\n", r.name, r.err)
		}
	}
	if failed > 0 {
		return b.String(), fmt.Errorf("%d of %d jobs failed", failed, len(jobs))
	}
	fmt.Fprintf(&b, "\nall %d containerized jobs passed\n", len(jobs))
	return b.String(), nil
}
