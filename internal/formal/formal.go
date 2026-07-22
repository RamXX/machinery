// Package formal is the Go port of verify_formal.sh + tlc.sh: regenerates the
// formal suite from source and runs the TLC model checker, shelling out to java.
package formal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/alloy"
	"github.com/RamXX/machinery/internal/compose"
	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/refine"
	"github.com/RamXX/machinery/internal/tla"
)

const (
	tlaVersion = "v1.7.4"
	tlaSHA256  = "936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88"
)

// jarPath resolves the pinned tla2tools.jar location (env override honored).
func jarPath() string {
	if j := os.Getenv("TLA_TOOLS_JAR"); j != "" {
		return j
	}
	cache, _ := os.UserCacheDir()
	if cache == "" {
		cache = os.TempDir()
	}
	return filepath.Join(cache, "machinery", "tla2tools-"+tlaVersion+".jar")
}

// ensureJar fetches+checksum-verifies the pinned jar on first use (like tlc.sh).
func ensureJar() (string, error) {
	return fetchJar(jarPath(),
		"https://github.com/tlaplus/tlaplus/releases/download/"+tlaVersion+"/tla2tools.jar",
		"tla2tools.jar "+tlaVersion, tlaSHA256)
}

// fetchJar downloads url into dest and verifies the pinned sha256; a jar
// already on disk is trusted (it was verified when it landed).
func fetchJar(dest, url, label, wantSHA string) (string, error) {
	if _, err := os.Stat(dest); err == nil {
		return dest, nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return "", err
	}
	fmt.Fprintf(os.Stderr, "fetching %s into %s\n", label, dest)
	tmp := dest + ".tmp"
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	out, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		return "", err
	}
	out.Close()
	data, _ := os.ReadFile(tmp)
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if got != wantSHA {
		os.Remove(tmp)
		return "", fmt.Errorf("checksum mismatch for %s: got %s, want %s", label, got, wantSHA)
	}
	if err := os.Rename(tmp, dest); err != nil {
		return "", err
	}
	return dest, nil
}

// runTLC mirrors tlc.sh: java -cp jar tlc2.TLC on a .tla/.cfg pair.
func runTLC(tlaPath, cfgPath string) (string, error) {
	jar, err := ensureJar()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(tlaPath)
	// TLC writes a states/ working directory; remove it on exit.
	defer os.RemoveAll(filepath.Join(dir, "states"))
	// TLC is exhaustive model-checking; give it a generous but bounded budget.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "java", "-XX:+UseParallelGC", "-cp", jar, "tlc2.TLC", "-cleanup",
		"-config", filepath.Base(cfgPath), filepath.Base(tlaPath))
	cmd.Dir = dir
	var buf strings.Builder
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err = cmd.Run()
	return buf.String(), err
}

// VerifyFormal regenerates the whole formal suite for a design from source
// and, unless genOnly is set, TLC-checks every .tla/.cfg pair. Mirrors
// verify_formal.sh line-for-line in its full-mode output. genOnly exists so
// Java-free environments (the nightly regen gate, adopter CI) can assert
// freshness through the same code path that the checked run uses, instead of
// re-implementing the generator orchestration in shell.
func VerifyFormal(design string, genOnly bool) int {
	mdir := filepath.Join(design, "machines")
	fdir := filepath.Join(design, "formal")
	if err := os.MkdirAll(fdir, 0755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	// regenerate; a generator that cannot produce its spec is a verification
	// failure, never a silent skip (a stale committed spec must not pass as fresh)
	genFail := 0
	genErr := func(err error) {
		fmt.Fprintln(os.Stderr, err)
		genFail++
	}
	// written tracks the basenames generated THIS run, so the report can
	// distinguish fresh pairs from committed orphans no source produces
	written := map[string]bool{}
	record := func(names []string) {
		for _, n := range names {
			written[n] = true
		}
	}
	machineSrcs := globExt(mdir, ".machine.json")
	semSrcs := globExt(fdir, ".semantics.yaml")
	compSrcs := globExt(fdir, ".composition.yaml")
	for _, mj := range machineSrcs {
		names, err := tla.RunWritten(mj, fdir)
		record(names)
		if err != nil {
			genErr(err)
		}
	}
	for _, sem := range semSrcs {
		m := strings.TrimSuffix(filepath.Base(sem), ".semantics.yaml")
		names, err := refine.RunWritten(filepath.Join(mdir, m+".machine.json"), sem, fdir)
		record(names)
		if err != nil {
			genErr(err)
		}
	}
	for _, comp := range compSrcs {
		data, err := os.ReadFile(comp)
		if err != nil {
			genErr(fmt.Errorf("compose_gen: %w", err))
			continue
		}
		compV, err := ir.LoadYAML(data)
		if err != nil || compV.Kind != ir.KindObject {
			genErr(fmt.Errorf("compose_gen: %s is not a composition mapping", comp))
			continue
		}
		coord := compV.AsObject().GetString("coordinator")
		if coord == "" {
			genErr(fmt.Errorf("compose_gen: %s declares no coordinator", comp))
			continue
		}
		names, err := compose.RunWritten(comp, filepath.Join(mdir, coord+".machine.json"), fdir)
		record(names)
		if err != nil {
			genErr(err)
		}
	}

	// static relational policy layer (opt-in: present only when the design
	// carries a policy annotation)
	policyAnn := filepath.Join(fdir, alloy.AnnotationName)
	havePolicy := false
	var policyCommands []alloy.Command
	if _, err := os.Stat(policyAnn); err == nil {
		havePolicy = true
		domainPath, _, perr := alloy.Paths(design)
		if perr != nil {
			genErr(perr)
			havePolicy = false
		} else if als, oracleMD, stats, aerr := alloy.GenerateAll(domainPath, policyAnn); aerr != nil {
			genErr(aerr)
			havePolicy = false
		} else if werr := os.WriteFile(filepath.Join(fdir, alloy.OutputName), []byte(als), 0644); werr != nil {
			genErr(werr)
			havePolicy = false
		} else if werr := os.WriteFile(filepath.Join(fdir, alloy.OracleName), []byte(oracleMD), 0644); werr != nil {
			genErr(werr)
			havePolicy = false
		} else {
			policyCommands = stats.Commands
		}
	}

	// static relational integrity model (opt-in: present only when the design
	// carries an integrity annotation)
	integrityAnn := filepath.Join(fdir, alloy.IntegrityAnnotationName)
	haveIntegrity := false
	var integrityCommands []alloy.Command
	if _, err := os.Stat(integrityAnn); err == nil {
		haveIntegrity = true
		domainPath, _, perr := alloy.Paths(design)
		if perr != nil {
			genErr(perr)
			haveIntegrity = false
		} else if als, stats, aerr := alloy.GenerateIntegrity(domainPath, integrityAnn); aerr != nil {
			genErr(aerr)
			haveIntegrity = false
		} else if werr := os.WriteFile(filepath.Join(fdir, alloy.IntegrityOutputName), []byte(als), 0644); werr != nil {
			genErr(werr)
			haveIntegrity = false
		} else {
			integrityCommands = stats.Commands
		}
	}

	// static relational isolation model (opt-in: present only when the design
	// carries an isolation annotation)
	isolationAnn := filepath.Join(fdir, alloy.IsolationAnnotationName)
	haveIsolation := false
	var isolationCommands []alloy.Command
	if _, err := os.Stat(isolationAnn); err == nil {
		haveIsolation = true
		domainPath, _, perr := alloy.Paths(design)
		if perr != nil {
			genErr(perr)
			haveIsolation = false
		} else if als, oracleMD, stats, aerr := alloy.GenerateIsolation(domainPath, isolationAnn); aerr != nil {
			genErr(aerr)
			haveIsolation = false
		} else if werr := os.WriteFile(filepath.Join(fdir, alloy.IsolationOutputName), []byte(als), 0644); werr != nil {
			genErr(werr)
			haveIsolation = false
		} else if werr := os.WriteFile(filepath.Join(fdir, alloy.IsolationOracleName), []byte(oracleMD), 0644); werr != nil {
			genErr(werr)
			haveIsolation = false
		} else {
			isolationCommands = stats.Commands
		}
	}

	// zero machines AND zero relational annotations: nothing can be generated,
	// so nothing can be verified as fresh; committed orphan pairs must not
	// masquerade as a regenerated (or checkable) suite
	if len(machineSrcs)+len(semSrcs)+len(compSrcs) == 0 && !havePolicy && !haveIntegrity && !haveIsolation && genFail == 0 {
		fmt.Fprintf(os.Stderr, "verify-formal: nothing to generate: no machines under %s and no semantics/composition/relational annotations under %s\n", mdir, fdir)
		return 1
	}

	// committed .tla/.cfg pairs a fresh generation did not produce: report
	// them, never count them as regenerated (reviewer fixture exp-c: an orphan
	// pair inflated the gen-only count and a zero-machine design exited 0)
	pairs := 0
	var orphans []string
	for _, tlaF := range globExt(fdir, ".tla") {
		if _, err := os.Stat(strings.TrimSuffix(tlaF, ".tla") + ".cfg"); err != nil {
			continue
		}
		if written[filepath.Base(tlaF)] {
			pairs++
		} else {
			orphans = append(orphans, strings.TrimSuffix(filepath.Base(tlaF), ".tla"))
		}
	}
	for _, o := range orphans {
		fmt.Fprintf(os.Stderr, "verify-formal: WARN: %s.tla/.cfg committed but not regenerated (no source); a fresh generation would not produce it\n", o)
	}

	if genOnly {
		fmt.Fprintf(os.Stdout, "%d spec pair(s) regenerated from source; TLC skipped (--gen-only)\n", pairs)
		if havePolicy {
			fmt.Fprintf(os.Stdout, "relational policy model + authz oracle regenerated (%s, %d commands; %s); Alloy skipped (--gen-only)\n", alloy.OutputName, len(policyCommands), alloy.OracleName)
		}
		if haveIntegrity {
			fmt.Fprintf(os.Stdout, "relational integrity model regenerated (%s, %d commands); Alloy skipped (--gen-only)\n", alloy.IntegrityOutputName, len(integrityCommands))
		}
		if haveIsolation {
			fmt.Fprintf(os.Stdout, "relational isolation model + tenant oracle regenerated (%s, %d commands; %s); Alloy skipped (--gen-only)\n", alloy.IsolationOutputName, len(isolationCommands), alloy.IsolationOracleName)
		}
		if genFail > 0 {
			fmt.Fprintf(os.Stderr, "verify-formal: %d generator failure(s); the committed specs above were NOT regenerated from source\n", genFail)
			return 1
		}
		if pairs == 0 && !havePolicy && !haveIntegrity && !haveIsolation {
			fmt.Fprintln(os.Stderr, "verify-formal: no .tla/.cfg pairs under "+fdir+": nothing to generate is a failure, not a pass")
			return 1
		}
		return 0
	}

	pass, fail := 0, 0
	for _, tlaF := range globExt(fdir, ".tla") {
		base := strings.TrimSuffix(tlaF, ".tla")
		cfgF := base + ".cfg"
		if _, err := os.Stat(cfgF); err != nil {
			continue
		}
		name := filepath.Base(base)
		out, err := runTLC(tlaF, cfgF)
		if err == nil && strings.Contains(out, "No error has been found") {
			fmt.Fprintf(os.Stdout, "  PASS  %s\n", name)
			pass++
		} else {
			fmt.Fprintf(os.Stdout, "  FAIL  %s\n", name)
			// an infrastructure failure (missing jar, missing java, timeout)
			// produces little or no TLC output; the error object is the only
			// diagnostic and must never be discarded
			if err != nil {
				fmt.Fprintf(os.Stdout, "        error: %v\n", err)
			}
			lines := strings.Split(out, "\n")
			start := 0
			if len(lines) > 40 {
				start = len(lines) - 40
			}
			for _, l := range lines[start:] {
				fmt.Fprintf(os.Stdout, "        %s\n", l)
			}
			fail++
		}
	}
	runLayer := func(present bool, alsName, prefix string, commands []alloy.Command) {
		if !present {
			return
		}
		vs, aerr := runAlloy(filepath.Join(fdir, alsName), commands)
		if aerr != nil {
			fmt.Fprintln(os.Stderr, aerr)
			fail++
			return
		}
		for _, v := range vs {
			name := prefix + "/" + v.Command.Name
			if v.Pass {
				fmt.Fprintf(os.Stdout, "  PASS  %s\n", name)
				pass++
			} else {
				fmt.Fprintf(os.Stdout, "  FAIL  %s\n", name)
				if v.Detail != "" {
					fmt.Fprintf(os.Stdout, "        %s\n", v.Detail)
				}
				fail++
			}
		}
	}
	runLayer(havePolicy, alloy.OutputName, "Policy", policyCommands)
	runLayer(haveIntegrity, alloy.IntegrityOutputName, "Integrity", integrityCommands)
	runLayer(haveIsolation, alloy.IsolationOutputName, "Isolation", isolationCommands)
	fmt.Fprintln(os.Stdout, "")
	fmt.Fprintf(os.Stdout, "%d passed, %d failed\n", pass, fail)
	if genFail > 0 {
		fmt.Fprintf(os.Stderr, "verify-formal: %d generator failure(s); the committed specs above were NOT regenerated from source\n", genFail)
		return 1
	}
	if pass+fail == 0 {
		fmt.Fprintln(os.Stderr, "verify-formal: no .tla/.cfg pairs under "+fdir+": nothing to check is a failure, not a pass")
		return 1
	}
	if fail > 0 {
		return 1
	}
	return 0
}

func globExt(dir, ext string) []string {
	entries, _ := os.ReadDir(dir)
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ext) {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	return out
}
