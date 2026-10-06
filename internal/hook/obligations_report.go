package hook

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/dirscan"
)

// reportProjectObligations lists, per project, the outstanding state that a
// Stop cannot clear on its own: in-flight tool tokens no live completion is
// known for, and obligations armed under a routing configuration that differs
// from the project's current one. Each line names the exact recovery. Neither
// condition blocks a Stop any more, so neither fails doctor; both keep the
// gates running at every Stop until resolved. An unreadable ledger fails
// every governed event in its project, so it fails doctor.
func reportProjectObligations(w io.Writer, dir string) bool {
	entries, err := dirscan.Read(dir, hookStateDirRepairMaxEntries)
	if err != nil {
		return true // the count line above already reported enumeration
	}
	healthy := true
	type line struct{ root, text string }
	var lines []line
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".state") || !validHookHexDigest(strings.TrimSuffix(name, ".state")) {
			continue
		}
		witness, err := readBoundedHookStateFile(filepath.Join(dir, name), "hook state", hookStateMaxBytes)
		if err != nil || witness == nil {
			continue
		}
		record, err := parseHookStateRecord(witness.body)
		if err != nil {
			healthy = false
			lines = append(lines, line{name, fmt.Sprintf("  ERROR    governance hook ledger %s is unreadable: %v", name, err)})
			continue
		}
		root := record.root
		label := root
		if label == "" {
			label = "ledger " + name + " (project root unrecorded)"
		}
		if len(record.pending) > 0 {
			unowned := 0
			for _, token := range record.pending {
				if !record.pendingOwners[token].known() {
					unowned++
				}
			}
			command := "machinery hook-state release --root <root> --orphaned"
			if root != "" {
				command = releaseCommand(root)
			}
			if unowned > 0 {
				lines = append(lines, line{label, fmt.Sprintf("  present  governance hook obligation for %s holds %d in-flight tool token(s) with no recorded owning session (%s); they block every Stop in the project. If no agent session in the project is running a tool, release them from your own terminal: %s",
					label, unowned, shortTokens(record.pending), command)})
			}
			if owned := len(record.pending) - unowned; owned > 0 {
				lines = append(lines, line{label, fmt.Sprintf("  present  governance hook obligation for %s holds %d in-flight tool token(s) owned by agent sessions. Only the session that armed a token waits for it; every other Stop runs the gates and keeps the obligation armed. If no agent session in the project is running a tool, release them from your own terminal: %s",
					label, owned, command)})
			}
		}
		if root == "" || len(record.routes) == 0 {
			continue
		}
		if info, err := os.Stat(root); err != nil || !info.IsDir() {
			continue
		}
		cfg, ok, _ := Load(root)
		if !ok || cfg.loadError != "" {
			lines = append(lines, line{label, fmt.Sprintf("  present  governance hook obligation for %s is outstanding but the project has no usable %s; restore the operator configuration so the next Stop can run the gates", label, ConfigName)})
			continue
		}
		body, err := routeSnapshotBody(cfg)
		if err != nil {
			continue
		}
		want := routeSnapshotDigest(body)
		foreign := 0
		for _, route := range record.routes {
			if route != want {
				foreign++
			}
		}
		if foreign > 0 {
			check := "machinery check " + shellQuote(filepath.Join(root, filepath.FromSlash(cfg.Design)))
			if cfg.Impl != "" {
				check += " --impl " + shellQuote(filepath.Join(root, filepath.FromSlash(cfg.Impl)))
			}
			narrowed, err := narrowedRoutes(root, record.routes, want, cfg)
			if err != nil {
				healthy = false
				lines = append(lines, line{label, fmt.Sprintf("  ERROR    governance hook obligation for %s has an unreadable route snapshot: %v", label, err)})
				continue
			}
			if len(narrowed) > 0 {
				lines = append(lines, line{label, fmt.Sprintf("  present  governance hook obligation for %s was armed under a routing configuration that checked more than the current %s (%s); Stops run the gates under the current one but keep the obligation armed. If an operator made this change deliberately, accept it: machinery hook-state release --root %s --routes",
					label, ConfigName, strings.Join(narrowed, "; "), shellQuote(root))})
				continue
			}
			lines = append(lines, line{label, fmt.Sprintf("  present  governance hook obligation for %s was armed under %d routing configuration(s) that differ from the current %s; the next Stop in the project re-runs the gates under the current configuration (to see the result now: %s)",
				label, foreign, ConfigName, check)})
		}
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].root < lines[j].root })
	for _, l := range lines {
		fmt.Fprintln(w, l.text)
	}
	return healthy
}
