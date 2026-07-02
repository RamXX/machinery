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

	"github.com/ramirosalas/machinery/internal/compose"
	"github.com/ramirosalas/machinery/internal/ir"
	"github.com/ramirosalas/machinery/internal/refine"
	"github.com/ramirosalas/machinery/internal/tla"
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
	jar := jarPath()
	if _, err := os.Stat(jar); err == nil {
		return jar, nil
	}
	if err := os.MkdirAll(filepath.Dir(jar), 0755); err != nil {
		return "", err
	}
	url := "https://github.com/tlaplus/tlaplus/releases/download/" + tlaVersion + "/tla2tools.jar"
	fmt.Fprintf(os.Stderr, "fetching tla2tools.jar %s into %s\n", tlaVersion, jar)
	tmp := jar + ".tmp"
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
	if got != tlaSHA256 {
		os.Remove(tmp)
		return "", fmt.Errorf("checksum mismatch for tla2tools.jar %s: got %s, want %s", tlaVersion, got, tlaSHA256)
	}
	if err := os.Rename(tmp, jar); err != nil {
		return "", err
	}
	return jar, nil
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

// VerifyFormal regenerates + TLC-checks the whole formal suite for a design.
// Mirrors verify_formal.sh line-for-line in its output.
func VerifyFormal(design string) int {
	mdir := filepath.Join(design, "machines")
	fdir := filepath.Join(design, "formal")
	if err := os.MkdirAll(fdir, 0755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	// regenerate (the sub-generators print their own status / exit on error)
	for _, mj := range globExt(mdir, ".machine.json") {
		_ = tla.Run(mj, fdir)
	}
	for _, sem := range globExt(fdir, ".semantics.yaml") {
		m := strings.TrimSuffix(filepath.Base(sem), ".semantics.yaml")
		_ = refine.Run(filepath.Join(mdir, m+".machine.json"), sem, fdir)
	}
	for _, comp := range globExt(fdir, ".composition.yaml") {
		data, _ := os.ReadFile(comp)
		compV, _ := ir.LoadYAML(data)
		coord := compV.AsObject().GetString("coordinator")
		_ = compose.Run(comp, filepath.Join(mdir, coord+".machine.json"), fdir)
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
	fmt.Fprintln(os.Stdout, "")
	fmt.Fprintf(os.Stdout, "%d passed, %d failed\n", pass, fail)
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
