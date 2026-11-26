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
	V    int    `json:"v"`
	File string `json:"file,omitempty"` // sha256 of the finished file; absent until the run ends
}

// Binding states of a receipts file.
const (
	bound      = ""
	notRun     = "not produced by machinery run"
	changed    = "changed after the run"
	unfinished = "unfinished" // machinery run started the file and never ended it
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

// finishEntry records the digest of the finished receipts file.
func finishEntry(p *project.Project, file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return writeEntry(p, file, ledgerEntry{File: digest(data)})
}

// binding compares a parsed receipts file with its ledger entry.
func binding(p *project.Project, fd *fileData) (string, error) {
	path, err := ledgerPath(p, fd.Rel)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return notRun, nil
	}
	if err != nil {
		return "", err
	}
	var e ledgerEntry
	if err := json.Unmarshal(data, &e); err != nil || e.V != 1 {
		return notRun, nil // not an entry machinery run wrote
	}
	switch {
	case e.File == "" && fd.End == nil:
		return unfinished, nil
	case e.File != fd.Sum:
		return changed, nil
	}
	return bound, nil
}
