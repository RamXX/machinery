package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/designlock"
	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/lint"
	"github.com/RamXX/machinery/internal/pack"
)

func withDesignSnapshot(design string, fn func(snapshot string) error) (retErr error) {
	lock, err := designlock.AcquireReader(design)
	if err != nil {
		return err
	}
	snapshot := lock.SourceRoot()
	defer func() {
		retErr = errors.Join(retErr, lock.CheckUnchanged(), lock.Release())
		retErr = remapSnapshotError(retErr, snapshot, design)
	}()
	designReaderAfterSnapshot()
	return fn(snapshot)
}

func remapSnapshotText(value, snapshot, design string) string {
	value = strings.ReplaceAll(value, filepath.Clean(snapshot), filepath.Clean(design))
	return strings.ReplaceAll(value, filepath.ToSlash(filepath.Clean(snapshot)), filepath.ToSlash(filepath.Clean(design)))
}

func remapSnapshotError(err error, snapshot, design string) error {
	if err == nil {
		return nil
	}
	code, remaining := commandResult(err)
	if code == 0 {
		return errors.New(remapSnapshotText(err.Error(), snapshot, design))
	}
	if remaining == nil {
		return commandExit(code)
	}
	return errors.Join(commandExit(code), errors.New(remapSnapshotText(remaining.Error(), snapshot, design)))
}

var designReaderAfterSnapshot = func() {}
var stableRegularAfterInitialRead = func(string) {}

type stableRegularFile struct {
	path string
	info os.FileInfo
	file *os.File
}

func openStableRegular(path string) (*stableRegularFile, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s must be a regular non-symlink file", path)
	}
	f, err := os.Open(abs)
	if err != nil {
		return nil, err
	}
	opened, err := f.Stat()
	if err != nil {
		return nil, errors.Join(err, f.Close())
	}
	if !os.SameFile(info, opened) {
		return nil, errors.Join(fmt.Errorf("%s changed while opening", path), f.Close())
	}
	return &stableRegularFile{path: abs, info: info, file: f}, nil
}

func (s *stableRegularFile) read() ([]byte, error) {
	if _, err := s.file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	return io.ReadAll(s.file)
}

func (s *stableRegularFile) revalidate(body []byte) error {
	pathInfo, err := os.Lstat(s.path)
	if err != nil {
		return err
	}
	opened, err := s.file.Stat()
	if err != nil {
		return err
	}
	metadataStable := func(info os.FileInfo) bool {
		return info.Size() == s.info.Size() && info.Mode() == s.info.Mode() && info.ModTime().Equal(s.info.ModTime())
	}
	if pathInfo.Mode()&os.ModeSymlink != 0 || !os.SameFile(s.info, pathInfo) || !os.SameFile(s.info, opened) || !metadataStable(pathInfo) || !metadataStable(opened) {
		return fmt.Errorf("%s changed identity while reading", s.path)
	}
	again, err := s.read()
	if err != nil {
		return err
	}
	if !bytes.Equal(body, again) {
		return fmt.Errorf("%s changed content while reading", s.path)
	}
	return nil
}

func (s *stableRegularFile) close() error { return s.file.Close() }

// Thresholds for the decomposition recommendation. Deliberately conservative
// defaults; they are advisory (the report recommends, the human decides), and
// they should be recalibrated as real runs accumulate evidence about where
// synthesis quality actually degrades.
const (
	scaleShardMachines  = 10      // SKILL's sharding rule
	scaleRecurseTokens  = 100_000 // estimated synthesis input beyond one context
	scaleRecurseEntWide = 25      // a domain model this wide rarely has one language
)

func newScaleCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "scale <design>",
		Short: "Measure a design's size and recommend sharding or recursive decomposition",
		Args:  cobra.ExactArgs(1),
	}
	c.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		stdoutW, stderrW := output.stdout, output.stderr
		displayDesign := args[0]
		return withDesignSnapshot(displayDesign, func(design string) error {
			if err := checkIsDir(design); err != nil {
				fmt.Fprintln(stderrW, err)
				return commandExitBecause(1, err)
			}
			// refuse to measure a directory that is not a design: an empty dir
			// once produced a confident "single-run design" recommendation
			if !pack.LooksLikeDesignDir(design) {
				fmt.Fprintf(stderrW, "scale: %s contains no *.modelith.yaml, no machines/, and no decomposition.yaml; not a machinery design directory, nothing to measure\n", displayDesign)
				return commandExit(1)
			}
			out := stdoutW
			fmt.Fprintln(out, "== scale  design size and decomposition signals ==")

			machines := 0
			states, transitions := 0, 0
			mdir := filepath.Join(design, "machines")
			if entries, err := os.ReadDir(mdir); err == nil {
				for _, e := range entries {
					if !strings.HasSuffix(e.Name(), ".machine.json") {
						continue
					}
					machines++
					m, err := ir.LoadMachineJSON(filepath.Join(mdir, e.Name()))
					if err != nil {
						return fmt.Errorf("scale: read machine %s: %w", e.Name(), err)
					}
					lintErrs, _, _, counts := lint.LintMachine(m, e.Name())
					if len(lintErrs) > 0 {
						return fmt.Errorf("scale: machine %s fails lint: %s", e.Name(), strings.Join(lintErrs, "; "))
					}
					states += counts.States
					transitions += counts.Transitions
				}
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("scale: read machines directory: %w", err)
			}
			entities, invariants := 0, 0
			var modelithBytes int
			entries, err := os.ReadDir(design)
			if err != nil {
				return fmt.Errorf("scale: read design directory: %w", err)
			}
			for _, e := range entries {
				if strings.HasSuffix(e.Name(), ".modelith.yaml") {
					data, err := os.ReadFile(filepath.Join(design, e.Name()))
					if err != nil {
						return fmt.Errorf("scale: read model %s: %w", e.Name(), err)
					}
					modelithBytes += len(data)
					v, err := ir.LoadYAML(data)
					if err != nil || v.AsObject() == nil {
						if err == nil {
							err = fmt.Errorf("root is not an object")
						}
						return fmt.Errorf("scale: parse model %s: %w", e.Name(), err)
					}
					ents := v.AsObject().GetObject("entities")
					entities += ents.Len()
					invariants += len(v.AsObject().Get2("invariants").AsArray())
					for _, en := range ents.Keys() {
						invariants += len(ents.Get2(en).AsObject().Get2("invariants").AsArray())
					}
				}
			}

			eventRowsRaw, err := pack.EventRowsStrict(design)
			if err != nil {
				return fmt.Errorf("scale: read event contracts: %w", err)
			}
			eventRows := len(eventRowsRaw)
			inputBytes := modelithBytes
			for _, f := range []string{"ARCHITECTURE.md", "workspace.dsl"} {
				if data, err := os.ReadFile(filepath.Join(design, f)); err == nil {
					inputBytes += len(data)
				} else if !os.IsNotExist(err) {
					return fmt.Errorf("scale: read %s: %w", f, err)
				}
			}
			if entries, err := os.ReadDir(mdir); err == nil {
				for _, e := range entries {
					if strings.HasSuffix(e.Name(), ".matrix.md") {
						data, err := os.ReadFile(filepath.Join(mdir, e.Name()))
						if err != nil {
							return fmt.Errorf("scale: read matrix %s: %w", e.Name(), err)
						}
						inputBytes += len(data)
					}
				}
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("scale: read machines directory: %w", err)
			}
			estTokens := inputBytes / 4

			fmt.Fprintf(out, "  machines: %d (states %d, transitions %d)\n", machines, states, transitions)
			fmt.Fprintf(out, "  entities: %d, invariants: %d, boundary event rows: %d\n", entities, invariants, eventRows)
			fmt.Fprintf(out, "  estimated synthesis input: ~%dk tokens (modelith + architecture + matrices)\n", estTokens/1000)
			if pack.HasDecomposition(design) {
				fmt.Fprintln(out, "  decomposition: PARENT (decomposition.yaml present)")
			}
			if pack.HasPack(design) {
				fmt.Fprintln(out, "  decomposition: CHILD (pack/ present)")
			}

			var recs []string
			if machines > scaleShardMachines {
				recs = append(recs, fmt.Sprintf("shard the synthesis: %d stateful components exceeds the ~%d-per-run rule (SKILL: Sharding large designs)", machines, scaleShardMachines))
			}
			if estTokens > scaleRecurseTokens {
				recs = append(recs, fmt.Sprintf("consider recursion: the synthesis input (~%dk tokens) exceeds a single-context budget; if the ubiquitous language forks across areas, split into subsystems with contract packs (machinery pack)", estTokens/1000))
			}
			if entities > scaleRecurseEntWide {
				recs = append(recs, fmt.Sprintf("consider recursion: %d entities rarely share one ubiquitous language; probe for context boundaries (the same word meaning different things is the split signal)", entities))
			}
			if len(recs) == 0 {
				fmt.Fprintln(out, "  recommendation: single-run design; neither sharding nor recursion is indicated")
			} else {
				for _, r := range recs {
					fmt.Fprintln(out, "  recommendation: "+r)
				}
				fmt.Fprintln(out, "  note: shard first (cheap: one design tree); recurse only when the domain model itself no longer fits one conversation. Recursion is the contract-pack protocol: machinery pack generate / pack refine / check --gate g5.")
			}
			return nil
		})
	}
	return c
}
