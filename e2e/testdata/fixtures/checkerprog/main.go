// Command checkerprog is a fixture checker: it reads a projection and writes
// evidence. Flags make it misbehave in the ways the scripts need.
//
// Usage: checkerprog [flags] <projection path> <evidence path>
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

func main() {
	id := flag.String("id", "rules", "checker id written to the evidence")
	covers := flag.String("covers", "*", "comma list of ids or id prefixes ending in * to judge; empty judges nothing")
	finding := flag.String("finding", "", "emit one finding with this severity (error or warn)")
	subject := flag.String("subject", "system", "subject of the finding and of the obligation")
	obligation := flag.String("obligation", "", "comma list of obligation names to declare")
	wrongHash := flag.Bool("wrong-hash", false, "write an input_sha256 that is not the projection hash")
	invalidJSON := flag.Bool("invalid-json", false, "write a document that is not JSON")
	badVerdict := flag.Bool("bad-verdict", false, "write the verdict pass although a finding has severity error")
	extraKey := flag.Bool("extra-key", false, "add a key the evidence format does not have")
	extraCover := flag.String("extra-cover", "", "add this id to covers even when the projection has no such id")
	exit := flag.Int("exit", 0, "exit with this status before writing anything")
	sleep := flag.Int("sleep", 0, "sleep this many seconds before writing")
	noWrite := flag.Bool("no-write", false, "exit 0 without writing the evidence")
	flag.Parse()
	if flag.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "usage: checkerprog [flags] <projection> <evidence>")
		os.Exit(64)
	}
	projPath, evPath := flag.Arg(0), flag.Arg(1)
	if *sleep > 0 {
		time.Sleep(time.Duration(*sleep) * time.Second)
	}
	if *exit != 0 {
		fmt.Fprintln(os.Stderr, "checkerprog: refusing to run")
		os.Exit(*exit)
	}
	if *noWrite {
		return
	}
	data, err := os.ReadFile(projPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *invalidJSON {
		write(evPath, []byte("{not json"))
		return
	}
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	if *wrongHash {
		hash = strings.Repeat("0", 64)
	}
	ids := projectionIDs(data)
	var judged []string
	for _, id := range ids {
		for _, pat := range strings.Split(*covers, ",") {
			if pat == "" {
				continue
			}
			if pat == "*" || id == pat || (strings.HasSuffix(pat, "*") && strings.HasPrefix(id, strings.TrimSuffix(pat, "*"))) {
				judged = append(judged, id)
				break
			}
		}
	}
	if *extraCover != "" {
		judged = append(judged, *extraCover)
	}
	ev := map[string]any{
		"version": 1, "checker": *id, "input_sha256": hash, "verdict": "pass",
		"covers": nonNil(judged), "findings": []any{}, "details": map[string]any{"judged": len(judged)},
	}
	if *finding != "" {
		ev["findings"] = []any{map[string]string{"severity": *finding, "subject": *subject, "rule": "R-1", "message": "fixture finding"}}
		if *finding == "error" && !*badVerdict {
			ev["verdict"] = "fail"
		}
	}
	if *obligation != "" {
		var obs []any
		for _, name := range strings.Split(*obligation, ",") {
			obs = append(obs, map[string]string{"name": name, "title": "Fixture obligation " + name, "subject": *subject})
		}
		ev["obligations"] = obs
	}
	if *extraKey {
		ev["surprise"] = true
	}
	out, _ := json.MarshalIndent(ev, "", "  ")
	write(evPath, append(out, '\n'))
}

func write(path string, data []byte) {
	if err := os.WriteFile(path, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// projectionIDs lists the ids a checker may judge, sorted.
func projectionIDs(data []byte) []string {
	var p struct {
		Entities []struct {
			ID      string `json:"id"`
			Actions []struct {
				ID string `json:"id"`
			} `json:"actions"`
		} `json:"entities"`
		Invariants []struct {
			ID string `json:"id"`
		} `json:"invariants"`
		Scenarios []struct {
			ID string `json:"id"`
		} `json:"scenarios"`
		Machines []struct {
			Transitions []struct {
				ID string `json:"id"`
			} `json:"transitions"`
			Ignores []struct {
				ID string `json:"id"`
			} `json:"ignores"`
			Refusals []struct {
				ID string `json:"id"`
			} `json:"refusals"`
			Clauses []struct {
				ID string `json:"id"`
			} `json:"clauses"`
		} `json:"machines"`
		FailureModes []struct {
			ID string `json:"id"`
		} `json:"failure_modes"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
		fmt.Fprintln(os.Stderr, "checkerprog: projection:", err)
		os.Exit(1)
	}
	var ids []string
	for _, e := range p.Entities {
		ids = append(ids, e.ID)
		for _, a := range e.Actions {
			ids = append(ids, a.ID)
		}
	}
	for _, i := range p.Invariants {
		ids = append(ids, i.ID)
	}
	for _, s := range p.Scenarios {
		ids = append(ids, s.ID)
	}
	for _, m := range p.Machines {
		for _, t := range m.Transitions {
			ids = append(ids, t.ID)
		}
		for _, i := range m.Ignores {
			ids = append(ids, i.ID)
		}
		for _, r := range m.Refusals {
			ids = append(ids, r.ID)
		}
		for _, c := range m.Clauses {
			ids = append(ids, c.ID)
		}
	}
	for _, f := range p.FailureModes {
		ids = append(ids, f.ID)
	}
	sort.Strings(ids)
	return ids
}
