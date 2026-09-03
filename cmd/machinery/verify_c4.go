// verify-c4: the C4 engine phase. G2 parses workspace.dsl for identifiers and
// tags; whether the DSL actually COMPILES under the Structurizr grammar was an
// attested checklist line ("run structurizr-cli export; fix syntax errors")
// that is literally a shell command. This subcommand is that command, made a
// first-class engine phase like verify-formal (which needs Java) and
// verify-checkers (which needs the registry): pure gates stay dependency-free,
// engine phases shell out and fail loudly when the engine is absent.

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/portablepath"
	"github.com/RamXX/machinery/internal/processcontrol"
	"github.com/RamXX/machinery/internal/runtimeclosure"
	machversion "github.com/RamXX/machinery/internal/version"
)

// structurizrEnv overrides the binary lookup (a pinned path, a wrapper
// script); PATH lookup of "structurizr-cli" is the default.
const structurizrEnv = "MACHINERY_STRUCTURIZR_CLI"
const structurizrClosureSHAEnv = "MACHINERY_STRUCTURIZR_CLI_CLOSURE_SHA256"

const verifyC4OutputLimit = 1 << 20

const (
	structurizrTreeMaxFiles       = 10_000
	structurizrTreeMaxBytes       = int64(1 << 30)
	structurizrExportMaxFileBytes = int64(64 << 20)
	// This limit is deliberately independent of filesystem size metadata. A
	// raced or corrupt size must never control how much ReadAll may allocate.
	structurizrExportReadLimit = structurizrExportMaxFileBytes + 1
)

var verifyC4Timeout = 5 * time.Minute
var verifyC4AfterVersion = func(string) {}
var verifyC4AfterJavaProbe = func(string) {}
var verifyC4AfterExportInventory = func() {}
var verifyC4AfterExportRead = func(string) {}
var verifyC4BeforeFinalExportInventory = func() {}

type validatedC4Export struct {
	info   os.FileInfo
	digest [sha256.Size]byte
}

func reserveC4ExportBytes(name string, size, total int64) (int64, error) {
	if size < 0 {
		return 0, fmt.Errorf("exported view %q reports negative size %d", name, size)
	}
	if size > structurizrExportMaxFileBytes {
		return 0, fmt.Errorf("exported view %q exceeds per-file bound (%d bytes, maximum %d)", name, size, structurizrExportMaxFileBytes)
	}
	if total < 0 || total > structurizrTreeMaxBytes {
		return 0, fmt.Errorf("structurizr export has invalid accumulated size %d", total)
	}
	remaining := structurizrTreeMaxBytes - total
	if size > remaining {
		return 0, fmt.Errorf("structurizr export exceeds remaining inventory bound (%d-byte view, %d bytes remaining)", size, remaining)
	}
	return total + size, nil
}

func readC4ExportBody(file io.Reader) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(file, structurizrExportReadLimit))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > structurizrExportMaxFileBytes {
		return nil, fmt.Errorf("exported view content exceeds fixed read bound of %d bytes", structurizrExportMaxFileBytes)
	}
	return body, nil
}

func readC4ExportEntries(dir *os.File) ([]fs.DirEntry, error) {
	entries, err := dir.ReadDir(structurizrTreeMaxFiles + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if len(entries) > structurizrTreeMaxFiles {
		return nil, fmt.Errorf("structurizr export exceeds inventory bound (%d files, maximum %d)", len(entries), structurizrTreeMaxFiles)
	}
	return entries, nil
}

func validateC4ExportInventory(output string) (_ []string, retErr error) {
	root, err := os.OpenRoot(output)
	if err != nil {
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	dir, err := root.Open(".")
	if err != nil {
		return nil, err
	}
	entries, readErr := readC4ExportEntries(dir)
	closeErr := dir.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	if len(entries) == 0 {
		return nil, fmt.Errorf("structurizr export produced no Mermaid view files")
	}
	verifyC4AfterExportInventory()
	seen := make(map[string]string)
	initial := make(map[string]os.FileInfo, len(entries))
	validated := make(map[string]validatedC4Export, len(entries))
	names := make([]string, 0, len(entries))
	var total int64
	for _, entry := range entries {
		name := entry.Name()
		if err := portablepath.ValidateBase(name); err != nil {
			return nil, fmt.Errorf("exported view filename %q is not portable: %w", name, err)
		}
		if filepath.Ext(name) != ".mmd" || strings.TrimSuffix(name, ".mmd") == "" {
			return nil, fmt.Errorf("unexpected structurizr export entry %q; deterministic Mermaid export permits only .mmd view files", name)
		}
		folded := strings.ToLower(name)
		if prior := seen[folded]; prior != "" {
			return nil, fmt.Errorf("exported view filenames %q and %q alias on case-insensitive filesystems", prior, name)
		}
		seen[folded] = name
		before, err := root.Lstat(name)
		if err != nil {
			return nil, err
		}
		if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
			return nil, fmt.Errorf("exported view %q must be a regular non-symlink file", name)
		}
		nextTotal, err := reserveC4ExportBytes(name, before.Size(), total)
		if err != nil {
			return nil, err
		}
		initial[name] = before
		total = nextTotal
		names = append(names, name)
	}
	for _, name := range names {
		before := initial[name]
		file, err := root.Open(name)
		if err != nil {
			return nil, err
		}
		opened, statErr := file.Stat()
		if statErr != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) || opened.Size() != before.Size() {
			return nil, errors.Join(statErr, fmt.Errorf("exported view %q changed identity while opening", name), file.Close())
		}
		body, readErr := readC4ExportBody(file)
		closeErr := file.Close()
		after, afterErr := root.Lstat(name)
		if err := errors.Join(readErr, closeErr, afterErr); err != nil {
			return nil, err
		}
		if int64(len(body)) != before.Size() || !os.SameFile(before, after) || after.Mode() != before.Mode() || after.Size() != before.Size() {
			return nil, fmt.Errorf("exported view %q changed while validating", name)
		}
		verifyC4AfterExportRead(name)
		again, err := root.Open(name)
		if err != nil {
			return nil, fmt.Errorf("reopen exported view %q: %w", name, err)
		}
		againInfo, statErr := again.Stat()
		againBody, againReadErr := readC4ExportBody(again)
		againCloseErr := again.Close()
		final, finalErr := root.Lstat(name)
		if err := errors.Join(statErr, againReadErr, againCloseErr, finalErr); err != nil {
			return nil, err
		}
		if !againInfo.Mode().IsRegular() || !os.SameFile(before, againInfo) || !os.SameFile(before, final) ||
			againInfo.Mode() != before.Mode() || final.Mode() != before.Mode() || againInfo.Size() != before.Size() ||
			final.Size() != before.Size() || int64(len(againBody)) != before.Size() || !bytes.Equal(body, againBody) {
			return nil, fmt.Errorf("exported view %q changed content, identity, or mode while validating", name)
		}
		validated[name] = validatedC4Export{info: final, digest: sha256.Sum256(againBody)}
	}
	verifyC4BeforeFinalExportInventory()
	finalDir, err := root.Open(".")
	if err != nil {
		return nil, fmt.Errorf("reopen structurizr export directory: %w", err)
	}
	finalEntries, finalReadErr := readC4ExportEntries(finalDir)
	finalCloseErr := finalDir.Close()
	if err := errors.Join(finalReadErr, finalCloseErr); err != nil {
		return nil, fmt.Errorf("re-enumerate structurizr export directory: %w", err)
	}
	sort.Slice(finalEntries, func(i, j int) bool { return finalEntries[i].Name() < finalEntries[j].Name() })
	if len(finalEntries) != len(names) {
		return nil, fmt.Errorf("structurizr export inventory changed during validation: final entry count is %d, want %d", len(finalEntries), len(names))
	}
	for i, entry := range finalEntries {
		name := entry.Name()
		if name != names[i] {
			return nil, fmt.Errorf("structurizr export inventory changed during validation: final entry %q replaces %q", name, names[i])
		}
		want := validated[name]
		info, err := root.Lstat(name)
		if err != nil {
			return nil, fmt.Errorf("revalidate final exported view %q: %w", name, err)
		}
		file, err := root.Open(name)
		if err != nil {
			return nil, fmt.Errorf("reopen final exported view %q: %w", name, err)
		}
		opened, statErr := file.Stat()
		body, readErr := readC4ExportBody(file)
		closeErr := file.Close()
		after, afterErr := root.Lstat(name)
		if err := errors.Join(statErr, readErr, closeErr, afterErr); err != nil {
			return nil, fmt.Errorf("revalidate final exported view %q: %w", name, err)
		}
		if !opened.Mode().IsRegular() || !os.SameFile(want.info, opened) || !os.SameFile(want.info, info) || !os.SameFile(want.info, after) ||
			opened.Mode() != want.info.Mode() || info.Mode() != want.info.Mode() || after.Mode() != want.info.Mode() ||
			opened.Size() != want.info.Size() || info.Size() != want.info.Size() || after.Size() != want.info.Size() ||
			int64(len(body)) != want.info.Size() || sha256.Sum256(body) != want.digest {
			return nil, fmt.Errorf("exported view %q changed after validation during final inventory reconciliation", name)
		}
	}
	return names, nil
}

type c4BoundedOutput struct {
	data      []byte
	truncated bool
}

func (b *c4BoundedOutput) Write(p []byte) (int, error) {
	overflow := len(b.data)+len(p) > verifyC4OutputLimit
	if room := verifyC4OutputLimit - len(b.data); room > 0 {
		take := len(p)
		if take > room {
			take = room
		}
		b.data = append(b.data, p[:take]...)
	}
	b.truncated = b.truncated || overflow
	return len(p), nil
}

func runC4Process(ctx context.Context, cmd *exec.Cmd, timeout time.Duration) (string, error) {
	buf := &c4BoundedOutput{}
	cmd.Stdout, cmd.Stderr = buf, buf
	err := processcontrol.Run(ctx, cmd)
	out := string(buf.data)
	if buf.truncated {
		out += fmt.Sprintf("\n[output truncated at %d bytes]\n", verifyC4OutputLimit)
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		err = errors.Join(err, fmt.Errorf("process timed out after %s", timeout))
	}
	return out, err
}

func openC4Java(workdir string) (*runtimeclosure.Java, error) {
	java, err := runtimeclosure.OpenJava()
	if err != nil {
		return nil, err
	}
	const probeTimeout = 10 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	cmd := exec.CommandContext(ctx, java.Path(), "-XshowSettings:properties", "-version")
	cmd.Env = runtimeclosure.Environment(workdir, workdir, java.Path())
	out, probeErr := runC4Process(ctx, cmd, probeTimeout)
	cancel()
	if err := errors.Join(probeErr, java.BindIdentity(out), java.Validate()); err != nil {
		return nil, errors.Join(fmt.Errorf("verify supported Java runtime: %w", err), java.Close())
	}
	verifyC4AfterJavaProbe(java.Path())
	if err := java.Validate(); err != nil {
		return nil, errors.Join(fmt.Errorf("revalidate %s before Structurizr execution: %w", java.Identity(), err), java.Close())
	}
	return java, nil
}

func snapshotStructurizrExecutable(bin string) (path string, cleanup func() error, retErr error) {
	abs, err := filepath.Abs(bin)
	if err != nil {
		return "", nil, err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", nil, err
	}
	f, err := openStableRegular(real)
	if err != nil {
		return "", nil, err
	}
	body, readErr := f.read()
	revalidateErr := f.revalidate(body)
	closeErr := f.close()
	if err := errors.Join(readErr, revalidateErr, closeErr); err != nil {
		return "", nil, err
	}
	dir, err := os.MkdirTemp("", "machinery-structurizr-tool-")
	if err != nil {
		return "", nil, err
	}
	path = filepath.Join(dir, filepath.Base(real))
	if bytes.HasPrefix(body, []byte("#!")) {
		if err := snapshotStructurizrTree(filepath.Dir(real), dir); err != nil {
			return "", nil, errors.Join(err, os.RemoveAll(dir))
		}
		if final, err := filepath.EvalSymlinks(abs); err != nil || final != real {
			return "", nil, errors.Join(err, fmt.Errorf("structurizr launcher symlink chain changed while snapshotting"), os.RemoveAll(dir))
		}
		return path, func() error { return os.RemoveAll(dir) }, nil
	}
	if err := os.WriteFile(path, body, 0o700); err != nil {
		return "", nil, errors.Join(err, os.RemoveAll(dir))
	}
	if final, err := filepath.EvalSymlinks(abs); err != nil || final != real {
		return "", nil, errors.Join(err, fmt.Errorf("structurizr executable symlink chain changed while snapshotting"), os.RemoveAll(dir))
	}
	return path, func() error { return os.RemoveAll(dir) }, nil
}

func snapshotStructurizrTree(source, destination string) (retErr error) {
	root, err := os.OpenRoot(source)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	files := 0
	var total int64
	var copyDir func(string) error
	copyDir = func(rel string) error {
		dir, err := root.Open(rel)
		if err != nil {
			return err
		}
		entries, readErr := dir.ReadDir(-1)
		closeErr := dir.Close()
		if err := errors.Join(readErr, closeErr); err != nil {
			return err
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, entry := range entries {
			child := entry.Name()
			if rel != "." {
				child = filepath.Join(rel, child)
			}
			info, err := root.Lstat(child)
			if err != nil {
				return err
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("structurizr distribution contains symlink %s", child)
			}
			dst := filepath.Join(destination, child)
			if info.IsDir() {
				if err := os.Mkdir(dst, 0o755); err != nil {
					return err
				}
				if err := copyDir(child); err != nil {
					return err
				}
				continue
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("structurizr distribution contains special file %s", child)
			}
			files++
			total += info.Size()
			if files > structurizrTreeMaxFiles || total > structurizrTreeMaxBytes {
				return fmt.Errorf("structurizr distribution exceeds snapshot bound (%d files, %d bytes)", structurizrTreeMaxFiles, structurizrTreeMaxBytes)
			}
			src, err := root.Open(child)
			if err != nil {
				return err
			}
			opened, statErr := src.Stat()
			if statErr != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
				return errors.Join(statErr, fmt.Errorf("structurizr distribution file %s changed identity while opening", child), src.Close())
			}
			body, readErr := io.ReadAll(io.LimitReader(src, info.Size()+1))
			closeErr := src.Close()
			after, finalErr := root.Lstat(child)
			if err := errors.Join(readErr, closeErr, finalErr); err != nil {
				return err
			}
			if int64(len(body)) != info.Size() || !os.SameFile(info, after) {
				return fmt.Errorf("structurizr distribution file %s changed while snapshotting", child)
			}
			mode := os.FileMode(0o600)
			if info.Mode()&0o111 != 0 {
				mode = 0o700
			}
			if err := os.WriteFile(dst, body, mode); err != nil {
				return err
			}
		}
		return nil
	}
	return copyDir(".")
}

func fingerprintStructurizrTree(rootPath string) (result [sha256.Size]byte, retErr error) {
	var zero [sha256.Size]byte
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return zero, err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	var names []string
	var collect func(string) error
	collect = func(rel string) error {
		dir, err := root.Open(rel)
		if err != nil {
			return err
		}
		entries, readErr := dir.ReadDir(-1)
		closeErr := dir.Close()
		if err := errors.Join(readErr, closeErr); err != nil {
			return err
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, entry := range entries {
			name := entry.Name()
			if rel != "." {
				name = filepath.Join(rel, name)
			}
			names = append(names, name)
			info, err := root.Lstat(name)
			if err != nil {
				return err
			}
			if info.IsDir() {
				if err := collect(name); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := collect("."); err != nil {
		return zero, err
	}
	if len(names) > structurizrTreeMaxFiles {
		return zero, fmt.Errorf("structurizr snapshot has %d entries; limit is %d", len(names), structurizrTreeMaxFiles)
	}
	hash := sha256.New()
	var total int64
	for _, name := range names {
		if filepath.Clean(name) == ".machinery-structurizr-receipt" {
			continue
		}
		info, err := root.Lstat(name)
		if err != nil {
			return zero, err
		}
		logical := filepath.ToSlash(name)
		if info.IsDir() {
			_, _ = fmt.Fprintf(hash, "d\x00%s\x00", logical)
			continue
		}
		if !info.Mode().IsRegular() {
			return zero, fmt.Errorf("structurizr snapshot contains special entry %s", logical)
		}
		total += info.Size()
		if total > structurizrTreeMaxBytes {
			return zero, fmt.Errorf("structurizr snapshot exceeds %d bytes", structurizrTreeMaxBytes)
		}
		file, err := root.Open(name)
		if err != nil {
			return zero, err
		}
		opened, statErr := file.Stat()
		if statErr != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
			return zero, errors.Join(statErr, fmt.Errorf("structurizr snapshot entry %s changed identity", logical), file.Close())
		}
		_, _ = fmt.Fprintf(hash, "f\x00%s\x00%d\x00%t\x00", logical, info.Size(), info.Mode()&0o111 != 0)
		_, copyErr := io.Copy(hash, file)
		if err := errors.Join(copyErr, file.Close()); err != nil {
			return zero, err
		}
	}
	copy(result[:], hash.Sum(nil))
	return result, nil
}

func verifyStructurizrVersion(ctx context.Context, bin, want string, env []string) error {
	cmd := exec.CommandContext(ctx, bin, "version")
	cmd.Env = env
	out, err := runC4Process(ctx, cmd, verifyC4Timeout)
	if err != nil {
		return fmt.Errorf("run structurizr-cli version: %w: %s", err, strings.TrimSpace(out))
	}
	return validateStructurizrVersionOutput(out, want)
}

func validateStructurizrVersionOutput(output, want string) error {
	if strings.Contains(strings.ReplaceAll(output, "\r\n", ""), "\r") {
		return fmt.Errorf("structurizr-cli version output contains a noncanonical carriage return")
	}
	clean := strings.ReplaceAll(output, "\r\n", "\n")
	clean = strings.TrimSuffix(clean, "\n")
	lines := strings.Split(clean, "\n")
	if len(lines) != 4 {
		return fmt.Errorf("structurizr-cli version output must contain exactly the four canonical identity lines and no warnings or other output")
	}
	wantCLI := "structurizr-cli: " + want
	if lines[0] != wantCLI {
		got := strings.TrimSpace(strings.TrimPrefix(lines[0], "structurizr-cli:"))
		if strings.HasPrefix(lines[0], "structurizr-cli:") && got != want {
			return fmt.Errorf("structurizr CLI %s does not match supported embedded version %s", got, want)
		}
		return fmt.Errorf("structurizr-cli version identity must be exactly %q", wantCLI)
	}
	if lines[1] != "structurizr-java: 5.0.2" {
		return fmt.Errorf("structurizr-java version identity is not the supported canonical value")
	}
	javaPrefix := "Java: " + strings.TrimSuffix(runtimeclosure.PinnedJavaProbeVersion, "+1-LTS") + "/" + runtimeclosure.CIJavaVendor + " ("
	if !strings.HasPrefix(lines[2], javaPrefix) || !strings.HasSuffix(lines[2], ")") {
		return fmt.Errorf("structurizr Java identity is not the pinned canonical runtime")
	}
	javaHome := strings.TrimSuffix(strings.TrimPrefix(lines[2], javaPrefix), ")")
	if javaHome == "" || !filepath.IsAbs(javaHome) || filepath.Clean(javaHome) != javaHome {
		return fmt.Errorf("structurizr Java identity contains a noncanonical runtime home")
	}
	if !strings.HasPrefix(lines[3], "OS: ") || strings.TrimSpace(strings.TrimPrefix(lines[3], "OS: ")) != strings.TrimPrefix(lines[3], "OS: ") || len(lines[3]) == len("OS: ") {
		return fmt.Errorf("structurizr OS identity is not canonical")
	}
	return nil
}

func canonicalExistingPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	real, err := filepath.EvalSymlinks(abs)
	if err == nil {
		return real
	}
	return filepath.Clean(abs)
}

func validateStructurizrExportOutput(output, dsl, outDir string, exported []string) error {
	if strings.Contains(strings.ReplaceAll(output, "\r\n", ""), "\r") {
		return fmt.Errorf("successful export output contains a noncanonical carriage return")
	}
	clean := strings.TrimSuffix(strings.ReplaceAll(output, "\r\n", "\n"), "\n")
	lines := strings.Split(clean, "\n")
	const start = "Exporting workspace from "
	if !strings.HasPrefix(lines[0], start) || canonicalExistingPath(strings.TrimPrefix(lines[0], start)) != canonicalExistingPath(dsl) {
		return fmt.Errorf("successful export did not identify the exact workspace snapshot")
	}
	progress := 1
	if len(lines) > progress && lines[progress] == " - no views defined; creating default views" {
		progress++
	}
	wantLines := len(exported) + progress + 2
	if len(lines) != wantLines {
		return fmt.Errorf("successful export emitted %d progress lines, want exactly %d", len(lines), wantLines)
	}
	if lines[progress] != " - exporting with MermaidDiagramExporter" || lines[len(lines)-1] != " - finished" {
		return fmt.Errorf("successful export progress is not the canonical Mermaid exporter transcript")
	}
	want := make(map[string]bool, len(exported))
	for _, name := range exported {
		want[name] = true
	}
	seen := make(map[string]bool, len(exported))
	for i, line := range lines[progress+1 : len(lines)-1] {
		const writing = " - writing "
		if !strings.HasPrefix(line, writing) {
			return fmt.Errorf("successful export progress line %d is not a canonical write record", i+progress+2)
		}
		path := strings.TrimPrefix(line, writing)
		name := filepath.Base(path)
		if canonicalExistingPath(filepath.Dir(path)) != canonicalExistingPath(outDir) || !want[name] || seen[name] {
			return fmt.Errorf("successful export progress does not match the exact .mmd inventory")
		}
		seen[name] = true
	}
	if len(seen) != len(want) {
		return fmt.Errorf("successful export progress omitted an exported .mmd file")
	}
	return nil
}

func newVerifyC4Cmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "verify-c4 <design-dir>",
		Short: "Compile workspace.dsl under structurizr-cli export (the C4 engine phase)",
		Args:  cobra.ExactArgs(1),
	}
	c.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		stdoutW, stderrW := output.stdout, output.stderr
		design := args[0]
		displayDSL := filepath.Join(design, "workspace.dsl")
		return withDesignWorkspaceSnapshot(design, func(snapshot string) (retErr error) {
			dsl := filepath.Join(snapshot, "workspace.dsl")
			if fi, err := os.Lstat(dsl); err != nil || !fi.Mode().IsRegular() {
				fmt.Fprintf(stderrW, "verify-c4: %s does not exist as a regular file; nothing to compile\n", displayDSL)
				return commandExit(1)
			}
			if _, err := validateStructurizrClosure(snapshot, dsl); err != nil {
				fmt.Fprintf(stderrW, "verify-c4: deterministic DSL preflight: %v\n", remapSnapshotError(err, snapshot, design))
				return commandExitBecause(1, err)
			}
			bin := os.Getenv(structurizrEnv)
			explicitStructurizr := bin != ""
			if bin == "" {
				path, err := provisionStructurizr()
				if err != nil {
					fmt.Fprintf(stderrW, "verify-c4: provision checksum-pinned Structurizr CLI: %v\n", err)
					return commandExitBecause(1, err)
				}
				bin = path
			}
			bin, cleanupBin, err := snapshotStructurizrExecutable(bin)
			if err != nil {
				fmt.Fprintf(stderrW, "verify-c4: snapshot Structurizr executable: %v\n", err)
				return commandExitBecause(1, err)
			}
			defer func() { retErr = errors.Join(retErr, cleanupBin()) }()
			toolRoot := filepath.Dir(bin)
			toolHash, err := fingerprintStructurizrTree(toolRoot)
			if err != nil {
				fmt.Fprintf(stderrW, "verify-c4: fingerprint Structurizr snapshot: %v\n", err)
				return commandExitBecause(1, err)
			}
			if explicitStructurizr {
				want := os.Getenv(structurizrClosureSHAEnv)
				got := fmt.Sprintf("%x", toolHash)
				if len(want) != sha256.Size*2 || want != strings.ToLower(want) || want != got {
					fmt.Fprintf(stderrW, "verify-c4: explicit %s requires matching exact lowercase closure sha256 in %s (got sha256:%s)\n", structurizrEnv, structurizrClosureSHAEnv, got)
					return commandExit(1)
				}
			}
			runtimeDir, err := os.MkdirTemp("", "machinery-c4-runtime-")
			if err != nil {
				return err
			}
			defer func() { retErr = errors.Join(retErr, os.RemoveAll(runtimeDir)) }()
			java, err := openC4Java(runtimeDir)
			if err != nil {
				diagnostic := strings.ReplaceAll(err.Error(), runtimeDir, "<java-home>")
				fmt.Fprintf(stderrW, "verify-c4: %s\n", diagnostic)
				return commandExitBecause(1, err)
			}
			defer func() {
				if validateErr := java.Validate(); validateErr != nil {
					retErr = errors.Join(retErr, fmt.Errorf("revalidate %s after Structurizr: %w", java.Identity(), validateErr))
				}
				retErr = errors.Join(retErr, java.Close())
			}()
			env := runtimeclosure.Environment(runtimeDir, runtimeDir, java.Path())
			wantVersion := machversion.StructurizrVersion
			versionCtx, cancelVersion := context.WithTimeout(cmd.Context(), verifyC4Timeout)
			versionErr := verifyStructurizrVersion(versionCtx, bin, wantVersion, env)
			cancelVersion()
			if versionErr != nil {
				diagnostic := strings.ReplaceAll(versionErr.Error(), filepath.Dir(bin), "<structurizr-tool>")
				fmt.Fprintf(stderrW, "verify-c4: %s\n", diagnostic)
				return commandExitBecause(1, versionErr)
			}
			verifyC4AfterVersion(bin)
			if afterVersion, hashErr := fingerprintStructurizrTree(toolRoot); hashErr != nil || afterVersion != toolHash {
				fmt.Fprintf(stderrW, "verify-c4: Structurizr snapshot changed between version verification and export (expected sha256:%x)\n", toolHash)
				return commandExitBecause(1, hashErr)
			}
			if err := java.Validate(); err != nil {
				fmt.Fprintf(stderrW, "verify-c4: Java runtime changed between version probe and export: %v\n", err)
				return commandExitBecause(1, err)
			}
			out, err := os.MkdirTemp("", "machinery-verify-c4")
			if err != nil {
				fmt.Fprintf(stderrW, "verify-c4: %v\n", err)
				return commandExitBecause(1, err)
			}
			defer func() { retErr = errors.Join(retErr, os.RemoveAll(out)) }()
			exportCtx, cancelExport := context.WithTimeout(cmd.Context(), verifyC4Timeout)
			run := exec.CommandContext(exportCtx, bin, "export", "-workspace", dsl, "-format", "mermaid", "-output", out)
			run.Env = env
			combined, err := runC4Process(exportCtx, run, verifyC4Timeout)
			cancelExport()
			if afterExport, hashErr := fingerprintStructurizrTree(toolRoot); hashErr != nil || afterExport != toolHash {
				fmt.Fprintf(stderrW, "verify-c4: Structurizr snapshot changed during export (expected sha256:%x)\n", toolHash)
				return commandExitBecause(1, hashErr)
			}
			if err != nil {
				combined = remapSnapshotText(combined, snapshot, design)
				combined = strings.ReplaceAll(combined, out, "<structurizr-output>")
				combined = strings.ReplaceAll(combined, filepath.Dir(bin), "<structurizr-tool>")
				combined = strings.ReplaceAll(combined, runtimeDir, "<java-home>")
				// the exporter's own message is the diagnosis; pass it through
				fmt.Fprintf(stdoutW, "%s", combined)
				fmt.Fprintf(stderrW, "verify-c4: FAIL %s does not compile under structurizr-cli export\n", displayDSL)
				return commandExitBecause(1, err)
			}
			exported, err := validateC4ExportInventory(out)
			if err != nil {
				return fmt.Errorf("verify-c4: inventory export output: %w", err)
			}
			if err := validateStructurizrExportOutput(combined, dsl, out, exported); err != nil {
				diagnostic := remapSnapshotText(combined, snapshot, design)
				diagnostic = strings.ReplaceAll(diagnostic, out, "<structurizr-output>")
				fmt.Fprintf(stderrW, "verify-c4: Structurizr successful output is not canonical; warnings are errors: %v\n%s", err, diagnostic)
				if !strings.HasSuffix(diagnostic, "\n") {
					fmt.Fprintln(stderrW)
				}
				return commandExitBecause(1, err)
			}
			fmt.Fprintf(stdoutW, "verify-c4: ok, %s compiles (structurizr-cli export, %d view file(s))\n", displayDSL, len(exported))
			return nil
		})
	}
	return c
}
