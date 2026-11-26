package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/project"
)

// A ledger entry binds one receipts file to the machinery run that wrote it
// (section 10.2). It lives outside the working tree, in
// <git common dir>/machinery/runs/<receipts file name>.
type ledgerEntry struct {
	V      int    `json:"v"`
	Commit string `json:"commit"`         // the commit the run was made at
	Dir    string `json:"dir"`            // the receipts directory the file was written in, absolute
	File   string `json:"file,omitempty"` // sha256 of the finished file; absent until the run ends
}

// Binding states of a receipts file.
const (
	bound      = ""
	notRun     = "not produced by machinery run"
	changed    = "changed after the run"
	unfinished = "unfinished" // machinery run started the file and never ended it
	gone       = "recorded by machinery run and no longer present"
)

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// ledgerPath returns the entry of a receipts file, by its base name.
func ledgerPath(p *project.Project, file string) (string, error) {
	dir, err := gitx.CommonDir(p.Root)
	if err != nil {
		return "", fmt.Errorf("cannot find the git directory for the run ledger: %v", err)
	}
	return filepath.Join(dir, "machinery", "runs", filepath.Base(file)), nil
}

func writeEntry(p *project.Project, file string, e ledgerEntry) error {
	path, err := ledgerPath(p, file)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	e.V = 1
	return os.WriteFile(path, jsonLine(e), 0o644)
}

// forget removes the entry of a receipts file that was deleted.
func forget(p *project.Project, file string) {
	if path, err := ledgerPath(p, file); err == nil {
		os.Remove(path)
	}
}

// finishEntry adds the digest of the finished receipts file to its entry.
func finishEntry(p *project.Project, file string, e ledgerEntry) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	e.File = digest(data)
	return writeEntry(p, file, e)
}

// ledgerDir is how an entry names the receipts directory of this project.
func ledgerDir(p *project.Project) string { return gitx.RealPath(p.Abs(Dir(p))) }

// ledgerEntries reads every entry of the ledger, by receipts file name. A
// file that is not an entry machinery run wrote is left out.
func ledgerEntries(p *project.Project) (map[string]ledgerEntry, error) {
	probe, err := ledgerPath(p, "x")
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(probe)
	list, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	res := map[string]ledgerEntry{}
	for _, de := range list {
		data, err := os.ReadFile(filepath.Join(dir, de.Name()))
		if err != nil {
			continue
		}
		var e ledgerEntry
		if json.Unmarshal(data, &e) == nil && e.V == 1 {
			res[de.Name()] = e
		}
	}
	return res, nil
}

// forgetGone removes the entries of this receipts directory whose file is no
// longer there: a new run replaces what was deleted (step 2 of section 10).
func forgetGone(p *project.Project) error {
	entries, err := ledgerEntries(p)
	if err != nil {
		return err
	}
	mine := ledgerDir(p)
	for name, e := range entries {
		if e.Dir != mine {
			continue
		}
		if _, err := os.Stat(filepath.Join(p.Abs(Dir(p)), name)); os.IsNotExist(err) {
			forget(p, name)
		}
	}
	return nil
}
