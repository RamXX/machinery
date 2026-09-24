package datalog

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// FuzzParse checks that the parser never panics, that every rejection is a
// positioned *Error, and that every accepted program runs (over a few
// generated input tuples, under small limits) without panicking. The seed
// corpus is every parity program.
func FuzzParse(f *testing.F) {
	for _, dir := range programDirs(f) {
		src, err := os.ReadFile(filepath.Join(dir, "program.dl"))
		if err != nil {
			f.Fatal(err)
		}
		f.Add(string(src))
	}
	f.Fuzz(func(t *testing.T, src string) {
		p, err := Parse(src, "fuzz.dl")
		if err != nil {
			var pe *Error
			if !errors.As(err, &pe) {
				t.Fatalf("rejection is not a *Error: %v", err)
			}
			if pe.Pos.Line < 1 || pe.Pos.Col < 1 {
				t.Fatalf("rejection without a position: %v", err)
			}
			return
		}
		in := Inputs{}
		for _, ri := range p.inputs {
			arity := len(p.decls[ri].Attrs)
			var rows [][]string
			for _, pat := range []string{"aa", "ab", "ba", "bb"} {
				row := make([]string, arity)
				for j := range row {
					row[j] = string(pat[j%2])
				}
				rows = append(rows, row)
			}
			in[p.decls[ri].Name] = rows
		}
		res, err := p.Run(in, Options{Explain: true, MaxTuples: 1000, MaxIterations: 100})
		if err != nil {
			if !errors.Is(err, ErrLimit) {
				t.Fatalf("run failed: %v", err)
			}
			return
		}
		for _, name := range res.OutputRelations() {
			_ = res.FormatCSV(name)
		}
	})
}
