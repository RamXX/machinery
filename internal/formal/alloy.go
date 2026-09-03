// Alloy orchestration for the opted-in static relational proof layers: fetch
// the pinned Alloy dist jar, run `exec` headless, and read the verdicts from
// its receipt. TLC checks machine behavior; Alloy checks admissible static
// configurations.

package formal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/RamXX/machinery/internal/alloy"
	"github.com/RamXX/machinery/internal/runtimeclosure"
)

const (
	alloyVersion       = "v6.2.0"
	alloySHA256        = "6b8c1cb5bc93bedfc7c61435c4e1ab6e688a242dc702a394628d9a9801edb78d"
	alloySolutionLimit = 8 << 20
)

var formalAfterAlloySolutionInspect = func(string) {}
var formalAfterAlloySolutionRead = func(string) {}

var alloySolutionStateLine = regexp.MustCompile(`^------State [0-9]+(?: \(loop\))?-------$`)

// alloyJarPath resolves the pinned Alloy dist jar location (env override honored).
func alloyJarPath() (string, error) {
	if j := os.Getenv("ALLOY_TOOLS_JAR"); j != "" {
		return j, nil
	}
	cache, err := formalUserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve user cache directory for pinned Alloy tool: %w", err)
	}
	if cache == "" {
		return "", fmt.Errorf("resolve user cache directory for pinned Alloy tool: empty path")
	}
	return filepath.Join(cache, "machinery", "alloy-dist-"+alloyVersion+".jar"), nil
}

func ensureAlloyJar() (string, error) {
	want, err := overrideSHA("ALLOY_TOOLS_JAR", "ALLOY_TOOLS_JAR_SHA256", alloySHA256)
	if err != nil {
		return "", err
	}
	path, err := alloyJarPath()
	if err != nil {
		return "", err
	}
	return fetchJar(path,
		"https://github.com/AlloyTools/org.alloytools.alloy/releases/download/"+alloyVersion+"/org.alloytools.alloy.dist.jar",
		"org.alloytools.alloy.dist.jar "+alloyVersion, want)
}

// --- receipt.json (what `alloy exec` writes next to the solutions) ---

type alloyRelation [][]string

type alloyInstance struct {
	Skolems map[string]struct {
		Arity int           `json:"arity"`
		Data  alloyRelation `json:"data"`
	} `json:"skolems"`
	Values map[string]map[string]alloyRelation `json:"values"`
}

type alloySolution struct {
	Duration    int             `json:"duration"`
	Incremental bool            `json:"incremental"`
	Instances   []alloyInstance `json:"instances"`
	LocalTime   string          `json:"localtime"`
	Timezone    string          `json:"timezone"`
	UTCTime     int64           `json:"utctime"`
}

type alloyCommandResult struct {
	Bitwidth  int             `json:"bitwidth"`
	Expects   int             `json:"expects"`
	MaxPrefix int             `json:"maxprefix"`
	MaxSeq    int             `json:"maxseq"`
	MinPrefix int             `json:"minprefix"`
	Name      string          `json:"name"`
	Overall   int             `json:"overall"`
	Solution  []alloySolution `json:"solution"`
	Source    string          `json:"source"`
	Type      string          `json:"type"` // check | run
}

type alloyReceipt struct {
	Commands             map[string]alloyCommandResult `json:"commands"`
	CoreMinimization     json.RawMessage               `json:"coreMinimization"`
	InferPartialInstance json.RawMessage               `json:"inferPartialInstance"`
	Repeat               json.RawMessage               `json:"repeat"`
	Sigs                 json.RawMessage               `json:"sigs"`
	Solver               json.RawMessage               `json:"solver"`
	Symmetry             json.RawMessage               `json:"symmetry"`
	Timestamp            json.RawMessage               `json:"timestamp"`
	Unrolls              json.RawMessage               `json:"unrolls"`
}

func (r alloyCommandResult) sat() bool {
	for _, s := range r.Solution {
		if len(s.Instances) > 0 {
			return true
		}
	}
	return false
}

func decodeAlloyReceipt(raw []byte) (alloyReceipt, error) {
	if err := rejectFormalDuplicateKeys(raw); err != nil {
		return alloyReceipt{}, fmt.Errorf("duplicate or malformed JSON: %w", err)
	}
	if err := validateAlloyReceiptExactKeys(raw); err != nil {
		return alloyReceipt{}, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var receipt alloyReceipt
	if err := dec.Decode(&receipt); err != nil {
		return alloyReceipt{}, err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return alloyReceipt{}, fmt.Errorf("multiple JSON values")
		}
		return alloyReceipt{}, err
	}
	if receipt.Commands == nil {
		return alloyReceipt{}, fmt.Errorf("missing commands object")
	}
	return receipt, nil
}

func unknownExactKey(object map[string]json.RawMessage, allowed ...string) error {
	want := make(map[string]bool, len(allowed))
	for _, key := range allowed {
		want[key] = true
	}
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !want[key] {
			return fmt.Errorf("unknown field %q", key)
		}
	}
	return nil
}

func validateAlloyReceiptExactKeys(raw []byte) error {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return err
	}
	if err := unknownExactKey(root, "commands", "coreMinimization", "inferPartialInstance", "repeat", "sigs", "solver", "symmetry", "timestamp", "unrolls"); err != nil {
		return err
	}
	commandsRaw, ok := root["commands"]
	if !ok {
		return fmt.Errorf("missing commands object")
	}
	var records map[string]json.RawMessage
	if err := json.Unmarshal(commandsRaw, &records); err != nil {
		return fmt.Errorf("commands must be an object: %w", err)
	}
	names := make([]string, 0, len(records))
	for name := range records {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		var record map[string]json.RawMessage
		if err := json.Unmarshal(records[name], &record); err != nil {
			return fmt.Errorf("command %s must be an object: %w", name, err)
		}
		if err := unknownExactKey(record, "bitwidth", "expects", "maxprefix", "maxseq", "minprefix", "name", "overall", "solution", "source", "type"); err != nil {
			return fmt.Errorf("command %s: %w", name, err)
		}
		for _, key := range []string{"bitwidth", "expects", "maxprefix", "maxseq", "minprefix", "name", "overall", "source", "type"} {
			if err := requireAlloyJSONField(record, key); err != nil {
				return fmt.Errorf("command %s: %w", name, err)
			}
		}
		if rawSolution, present := record["solution"]; present {
			if err := validateAlloySolutionReceipt(name, rawSolution); err != nil {
				return err
			}
		}
	}
	return nil
}

func requireAlloyJSONField(object map[string]json.RawMessage, key string) error {
	raw, ok := object[key]
	if !ok {
		return fmt.Errorf("missing required field %q", key)
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fmt.Errorf("required field %q must not be null", key)
	}
	return nil
}

func validateAlloySolutionReceipt(command string, raw json.RawMessage) error {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fmt.Errorf("command %s: solution must be a one-element array, not null", command)
	}
	var solutions []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &solutions); err != nil {
		return fmt.Errorf("command %s: solution must be an array: %w", command, err)
	}
	if len(solutions) != 1 {
		return fmt.Errorf("command %s: solution array has %d entries, want exactly 1", command, len(solutions))
	}
	solution := solutions[0]
	if err := unknownExactKey(solution, "duration", "incremental", "instances", "localtime", "timezone", "utctime"); err != nil {
		return fmt.Errorf("command %s solution: %w", command, err)
	}
	for _, key := range []string{"duration", "incremental", "instances", "localtime", "timezone", "utctime"} {
		if err := requireAlloyJSONField(solution, key); err != nil {
			return fmt.Errorf("command %s solution: %w", command, err)
		}
	}
	var instances []map[string]json.RawMessage
	if err := json.Unmarshal(solution["instances"], &instances); err != nil {
		return fmt.Errorf("command %s solution instances must be an array: %w", command, err)
	}
	if len(instances) != 1 {
		return fmt.Errorf("command %s solution has %d instances, want exactly 1", command, len(instances))
	}
	instance := instances[0]
	if err := unknownExactKey(instance, "skolems", "values"); err != nil {
		return fmt.Errorf("command %s solution instance: %w", command, err)
	}
	if err := requireAlloyJSONField(instance, "values"); err != nil {
		return fmt.Errorf("command %s solution instance: %w", command, err)
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(instance["values"], &values); err != nil || values == nil {
		return fmt.Errorf("command %s solution instance values must be an object", command)
	}
	if rawSkolems, ok := instance["skolems"]; ok {
		var skolems map[string]json.RawMessage
		if bytes.Equal(bytes.TrimSpace(rawSkolems), []byte("null")) || json.Unmarshal(rawSkolems, &skolems) != nil || skolems == nil {
			return fmt.Errorf("command %s solution instance skolems must be an object when present", command)
		}
	}
	return nil
}

func loadAlloyReceipt(raw []byte, commands []alloy.Command) (alloyReceipt, error) {
	receipt, err := decodeAlloyReceipt(raw)
	if err != nil {
		return alloyReceipt{}, err
	}
	if err := validateAlloyReceiptInventory(receipt, commands); err != nil {
		return alloyReceipt{}, err
	}
	return receipt, nil
}

func validateAlloyReceiptInventory(receipt alloyReceipt, commands []alloy.Command) error {
	expected := make(map[string]alloy.Command, len(commands))
	for _, command := range commands {
		if _, exists := expected[command.Name]; exists {
			return fmt.Errorf("generated command inventory repeats %s", command.Name)
		}
		expected[command.Name] = command
	}
	actualNames := make([]string, 0, len(receipt.Commands))
	for name := range receipt.Commands {
		actualNames = append(actualNames, name)
	}
	sort.Strings(actualNames)
	for _, name := range actualNames {
		result := receipt.Commands[name]
		command, ok := expected[name]
		if !ok {
			return fmt.Errorf("receipt.json has unexpected command %s; the model and the run disagree", name)
		}
		if result.Name != name {
			return fmt.Errorf("receipt.json command key %s contains identity %s", name, result.Name)
		}
		if result.Type != command.Kind {
			return fmt.Errorf("receipt.json command %s has type %s, expected %s", name, result.Type, command.Kind)
		}
	}
	expectedNames := make([]string, 0, len(expected))
	for name := range expected {
		expectedNames = append(expectedNames, name)
	}
	sort.Strings(expectedNames)
	for _, name := range expectedNames {
		if _, ok := receipt.Commands[name]; !ok {
			return fmt.Errorf("receipt.json has no result for command %s; the model and the run disagree", name)
		}
	}
	return nil
}

// AlloyVerdict is one command's outcome, in the model's command order.
type AlloyVerdict struct {
	Command alloy.Command
	Pass    bool
	Detail  string // counterexample or vacuity note on failure
}

// runAlloy executes every command in alsPath and maps the receipt back onto
// the generated command list (kind decides pass semantics: check passes on
// UNSAT, run passes on SAT). Solutions are written as text (-t text): the
// receipt's per-atom values are unreliable for inherited and total relations,
// while the text form carries every relation in full.
func runAlloy(alsPath string, commands []alloy.Command) (result []AlloyVerdict, retErr error) {
	jar, err := ensureAlloyJar()
	if err != nil {
		return nil, err
	}
	formalAfterJarResolved(jar)
	outDir, err := os.MkdirTemp("", "machinery-alloy-output-")
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = errors.Join(retErr, os.RemoveAll(outDir))
		retErr = redactPrivateError(retErr, outDir, "<alloy-workdir>")
	}()
	toolDir, err := os.MkdirTemp("", "machinery-alloy-tool-")
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = errors.Join(retErr, os.RemoveAll(toolDir))
		retErr = redactPrivateError(retErr, toolDir, "<alloy-tool>")
	}()
	want, err := overrideSHA("ALLOY_TOOLS_JAR", "ALLOY_TOOLS_JAR_SHA256", alloySHA256)
	if err != nil {
		return nil, err
	}
	jar, err = snapshotVerifiedJar(jar, want, "Alloy dist jar", toolDir)
	if err != nil {
		return nil, err
	}
	java, err := openFormalJava(outDir)
	if err != nil {
		return nil, err
	}
	defer func() {
		if validateErr := java.Validate(); validateErr != nil {
			retErr = errors.Join(retErr, fmt.Errorf("revalidate %s after Alloy: %w", java.Identity(), validateErr))
		}
		retErr = errors.Join(retErr, java.Close())
	}()
	ctx, cancel := context.WithTimeout(context.Background(), formalProcessTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, java.Path(), "-jar", jar, "exec", "-f", "-t", "text", "-c", "*", "-o", outDir, filepath.Base(alsPath))
	cmd.Dir = filepath.Dir(alsPath)
	cmd.Env = runtimeclosure.Environment(outDir, outDir, java.Path())
	processOut, runErr := runBoundedProcess(ctx, cmd, formalProcessTimeout)
	gotJarSHA, jarErr := fileSHA256(jar)
	if jarErr != nil || gotJarSHA != want {
		return nil, errors.Join(jarErr, fmt.Errorf("alloy tool snapshot changed during execution"))
	}
	if runErr != nil {
		return nil, fmt.Errorf("alloy exec failed on %s: %w\n%s", filepath.Base(alsPath), runErr, tail(processOut, 20))
	}
	raw, err := readStableAlloyReceipt(outDir)
	if err != nil {
		return nil, fmt.Errorf("alloy exec wrote no receipt.json for %s: %w", filepath.Base(alsPath), err)
	}
	outcomes, err := parseAlloySuccessOutput(processOut, commands)
	if err != nil {
		return nil, fmt.Errorf("alloy exec emitted unexpected success diagnostics for %s: %w", filepath.Base(alsPath), err)
	}
	receipt, err := loadAlloyReceipt(raw, commands)
	if err != nil {
		return nil, fmt.Errorf("receipt.json for %s does not parse: %w", filepath.Base(alsPath), err)
	}
	if err := validateAlloyReceiptOutcomes(receipt, commands, outcomes); err != nil {
		return nil, fmt.Errorf("receipt.json for %s contradicts engine results: %w", filepath.Base(alsPath), err)
	}
	details, err := loadAlloySolutionDetails(outDir, receipt, commands)
	if err != nil {
		return nil, fmt.Errorf("alloy exec emitted invalid solution artifacts for %s: %w", filepath.Base(alsPath), err)
	}
	return verdicts(receipt, commands, func(name string) string { return details[name] })
}

func readStableAlloyReceipt(outDir string) (_ []byte, retErr error) {
	root, err := os.OpenRoot(outDir)
	if err != nil {
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	const name = "receipt.json"
	before, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, fmt.Errorf("receipt.json must be a regular non-symlink file")
	}
	read := func() ([]byte, os.FileInfo, error) {
		f, err := root.Open(name)
		if err != nil {
			return nil, nil, err
		}
		info, statErr := f.Stat()
		if statErr == nil && (info.Size() < 0 || info.Size() > alloySolutionLimit) {
			statErr = fmt.Errorf("receipt.json exceeds %d bytes", alloySolutionLimit)
		}
		var body bytes.Buffer
		var readErr error
		if statErr == nil {
			body.Grow(int(info.Size()))
			_, readErr = copyFormalExact(&body, f, info.Size(), "Alloy receipt")
		}
		closeErr := f.Close()
		if err := errors.Join(statErr, readErr, closeErr); err != nil {
			return nil, nil, err
		}
		return body.Bytes(), info, nil
	}
	first, firstInfo, err := read()
	if err != nil || !firstInfo.Mode().IsRegular() || !os.SameFile(before, firstInfo) || before.Mode() != firstInfo.Mode() {
		return nil, errors.Join(err, fmt.Errorf("receipt.json changed identity or mode while opening"))
	}
	second, secondInfo, err := read()
	final, finalErr := root.Lstat(name)
	if err := errors.Join(err, finalErr); err != nil {
		return nil, err
	}
	if !secondInfo.Mode().IsRegular() || !os.SameFile(before, secondInfo) || !os.SameFile(before, final) ||
		secondInfo.Mode() != before.Mode() || final.Mode() != before.Mode() || secondInfo.Size() != before.Size() || final.Size() != before.Size() ||
		!secondInfo.ModTime().Equal(before.ModTime()) || !final.ModTime().Equal(before.ModTime()) || !bytes.Equal(first, second) {
		return nil, fmt.Errorf("receipt.json changed content, identity, or mode while validating")
	}
	return first, nil
}

func loadAlloySolutionDetails(outDir string, receipt alloyReceipt, commands []alloy.Command) (_ map[string]string, retErr error) {
	beforeRoot, err := os.Lstat(outDir)
	if err != nil {
		return nil, err
	}
	if beforeRoot.Mode()&os.ModeSymlink != 0 || !beforeRoot.IsDir() {
		return nil, fmt.Errorf("solution root must be a real directory")
	}
	root, err := os.OpenRoot(outDir)
	if err != nil {
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	openedRoot, err := root.Stat(".")
	if err != nil || !os.SameFile(beforeRoot, openedRoot) {
		return nil, errors.Join(err, fmt.Errorf("solution root changed identity while opening"))
	}

	expected := make([]string, 0, len(commands))
	failedChecks := map[string]string{}
	failedBySolution := map[string]string{}
	for _, command := range commands {
		result := receipt.Commands[command.Name]
		if !result.sat() {
			continue
		}
		name, err := alloySolutionName(command.Name)
		if err != nil {
			return nil, err
		}
		expected = append(expected, name)
		if command.Kind == "check" {
			failedChecks[command.Name] = name
			failedBySolution[name] = command.Name
		}
	}
	sort.Strings(expected)
	if err := validateAlloySolutionInventory(root, expected); err != nil {
		return nil, err
	}

	details := make(map[string]string, len(failedChecks))
	for _, solutionName := range expected {
		body, err := readStableAlloySolution(root, solutionName)
		if err != nil {
			return nil, err
		}
		if commandName := failedBySolution[solutionName]; commandName != "" {
			details[commandName] = renderSolutionText(string(body))
		}
	}
	if err := validateAlloySolutionInventory(root, expected); err != nil {
		return nil, fmt.Errorf("solution inventory changed while reading: %w", err)
	}
	afterRoot, err := os.Lstat(outDir)
	if err != nil || !os.SameFile(beforeRoot, afterRoot) {
		return nil, errors.Join(err, fmt.Errorf("solution root changed identity while reading"))
	}
	return details, nil
}

func alloySolutionName(commandName string) (string, error) {
	if commandName == "" || filepath.Base(commandName) != commandName || strings.ContainsAny(commandName, "/\\\x00\r\n") {
		return "", fmt.Errorf("unsafe Alloy command identity %q for solution artifact", commandName)
	}
	return commandName + "-solution-0.txt", nil
}

func validateAlloySolutionInventory(root *os.Root, expected []string) error {
	entries, err := readFormalRootDirectory(root, "Alloy output")
	if err != nil {
		return err
	}
	actual := make([]string, 0, len(expected))
	for _, entry := range entries {
		name := entry.Name()
		if name == "receipt.json" {
			info, err := root.Lstat(name)
			if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
				return errors.Join(err, fmt.Errorf("receipt.json must be a regular non-symlink file"))
			}
			continue
		}
		if !strings.Contains(name, "-solution-") || !strings.HasSuffix(name, ".txt") {
			return fmt.Errorf("unexpected Alloy output artifact %q", name)
		}
		actual = append(actual, name)
	}
	sort.Strings(actual)
	if strings.Join(actual, "\x00") != strings.Join(expected, "\x00") {
		return fmt.Errorf("solution inventory is %v, want exactly %v", actual, expected)
	}
	return nil
}

func readStableAlloySolution(root *os.Root, name string) (_ []byte, retErr error) {
	before, err := root.Lstat(name)
	if err != nil {
		return nil, fmt.Errorf("required solution %s: %w", name, err)
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, fmt.Errorf("required solution %s must be a regular non-symlink file", name)
	}
	formalAfterAlloySolutionInspect(name)
	file, err := root.Open(name)
	if err != nil {
		return nil, fmt.Errorf("open required solution %s: %w", name, err)
	}
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) || before.Mode() != opened.Mode() {
		return nil, errors.Join(err, fmt.Errorf("required solution %s changed identity or mode while opening", name), file.Close())
	}
	if opened.Size() < 0 || opened.Size() > alloySolutionLimit {
		return nil, errors.Join(fmt.Errorf("required solution %s exceeds %d bytes", name, alloySolutionLimit), file.Close())
	}
	var body bytes.Buffer
	body.Grow(int(opened.Size()))
	_, err = copyFormalExact(&body, file, opened.Size(), "Alloy solution "+name)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("read required solution %s: %w", name, err), file.Close())
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	after, err := root.Lstat(name)
	if err != nil || !os.SameFile(opened, after) || opened.Mode() != after.Mode() || opened.Size() != after.Size() || !opened.ModTime().Equal(after.ModTime()) {
		return nil, errors.Join(err, fmt.Errorf("required solution %s changed while reading", name))
	}
	if err := validateAlloySolutionBody(body.Bytes()); err != nil {
		return nil, fmt.Errorf("required solution %s is not canonical: %w", name, err)
	}
	formalAfterAlloySolutionRead(name)
	again, err := root.Open(name)
	if err != nil {
		return nil, fmt.Errorf("reopen required solution %s: %w", name, err)
	}
	againInfo, statErr := again.Stat()
	var againBody bytes.Buffer
	var readErr error
	if statErr == nil && (againInfo.Size() < 0 || againInfo.Size() > alloySolutionLimit) {
		statErr = fmt.Errorf("required solution %s exceeds %d bytes", name, alloySolutionLimit)
	}
	if statErr == nil {
		againBody.Grow(int(againInfo.Size()))
		_, readErr = copyFormalExact(&againBody, again, againInfo.Size(), "Alloy solution "+name)
	}
	closeErr := again.Close()
	final, finalErr := root.Lstat(name)
	if err := errors.Join(statErr, readErr, closeErr, finalErr); err != nil {
		return nil, err
	}
	if !againInfo.Mode().IsRegular() || !os.SameFile(before, againInfo) || !os.SameFile(before, final) ||
		againInfo.Mode() != before.Mode() || final.Mode() != before.Mode() || againInfo.Size() != before.Size() || final.Size() != before.Size() ||
		!againInfo.ModTime().Equal(before.ModTime()) || !final.ModTime().Equal(before.ModTime()) || !bytes.Equal(body.Bytes(), againBody.Bytes()) {
		return nil, fmt.Errorf("required solution %s changed content, identity, or mode while validating", name)
	}
	return body.Bytes(), nil
}

func validateAlloySolutionBody(body []byte) error {
	if len(body) == 0 || len(bytes.TrimSpace(body)) == 0 {
		return fmt.Errorf("solution is empty")
	}
	if !utf8.Valid(body) || bytes.IndexByte(body, 0) >= 0 || bytes.Contains(body, []byte{'\r'}) {
		return fmt.Errorf("solution is not canonical UTF-8 text")
	}
	if !bytes.HasSuffix(body, []byte("\n\n")) || bytes.HasSuffix(body, []byte("\n\n\n")) {
		return fmt.Errorf("solution does not have the canonical final blank line")
	}
	lines := strings.Split(strings.TrimSuffix(string(body), "\n\n"), "\n")
	if len(lines) < 3 || lines[0] != "---Trace---" || !alloySolutionStateLine.MatchString(lines[1]) {
		return fmt.Errorf("solution does not begin with the canonical Alloy trace header")
	}
	haveAssignment := false
	for _, line := range lines[2:] {
		for _, r := range line {
			if r < 0x20 && r != '\t' {
				return fmt.Errorf("solution contains a control character")
			}
		}
		if alloySolutionStateLine.MatchString(line) {
			continue
		}
		if !strings.Contains(line, "={") || !strings.HasSuffix(line, "}") {
			return fmt.Errorf("solution contains a noncanonical trace line")
		}
		haveAssignment = true
	}
	if !haveAssignment {
		return fmt.Errorf("solution contains no canonical relation assignment")
	}
	return nil
}

func validateAlloySuccessOutput(output string, commands []alloy.Command) error {
	_, err := parseAlloySuccessOutput(output, commands)
	return err
}

func parseAlloySuccessOutput(output string, commands []alloy.Command) (map[string]bool, error) {
	clean := strings.ReplaceAll(strings.ReplaceAll(output, "\r", ""), "\b", "")
	lines := strings.Split(strings.TrimSpace(clean), "\n")
	if len(commands) == 0 && len(lines) == 1 && lines[0] == "" {
		return map[string]bool{}, nil
	}
	if len(lines) != len(commands) {
		return nil, fmt.Errorf("got %d nonempty engine line(s), want exactly %d command result line(s)", len(lines), len(commands))
	}
	outcomes := make(map[string]bool, len(commands))
	for i, command := range commands {
		fields := strings.Fields(lines[i])
		wantIndex := fmt.Sprintf("%02d.", i)
		if containsEngineDiagnostic(fields) || !canonicalAlloyProgress(fields, wantIndex, command) {
			return nil, fmt.Errorf("line %d is not the canonical result for %s %s: %q", i+1, command.Kind, command.Name, lines[i])
		}
		outcomes[command.Name] = fields[len(fields)-1] == "SAT"
	}
	return outcomes, nil
}

func validateAlloyReceiptOutcomes(receipt alloyReceipt, commands []alloy.Command, outcomes map[string]bool) error {
	for _, command := range commands {
		stdoutSAT, ok := outcomes[command.Name]
		if !ok {
			return fmt.Errorf("stdout has no outcome for command %s", command.Name)
		}
		receiptSAT := receipt.Commands[command.Name].sat()
		if stdoutSAT != receiptSAT {
			return fmt.Errorf("command %s stdout is %s but receipt solution is %s", command.Name, alloyOutcome(stdoutSAT), alloyOutcome(receiptSAT))
		}
	}
	return nil
}

func alloyOutcome(sat bool) string {
	if sat {
		return "SAT"
	}
	return "UNSAT"
}

func containsEngineDiagnostic(fields []string) bool {
	for _, field := range fields {
		word := strings.ToLower(strings.Trim(field, "[]():,.;"))
		switch word {
		case "warning", "warn", "deprecated", "exception", "error", "fatal":
			return true
		}
	}
	return false
}

func canonicalAlloyProgress(fields []string, wantIndex string, command alloy.Command) bool {
	if len(fields) < 5 || fields[0] != wantIndex || fields[1] != command.Kind || fields[2] != command.Name || !asciiDecimal(fields[3]) {
		return false
	}
	switch fields[len(fields)-1] {
	case "UNSAT":
		return len(fields) == 5
	case "SAT":
		if len(fields) != 6 {
			return false
		}
		left, right, ok := strings.Cut(fields[4], "/")
		return ok && asciiDecimal(left) && asciiDecimal(right)
	default:
		return false
	}
}

func asciiDecimal(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// verdicts maps the receipt onto the generated command list. A command the
// receipt does not mention is an error, never a silent pass. detail renders
// a counterexample summary for a failed check ("" is acceptable).
func verdicts(receipt alloyReceipt, commands []alloy.Command, detail func(name string) string) ([]AlloyVerdict, error) {
	if err := validateAlloyReceiptInventory(receipt, commands); err != nil {
		return nil, err
	}
	var out []AlloyVerdict
	for _, c := range commands {
		res, ok := receipt.Commands[c.Name]
		if !ok {
			return nil, fmt.Errorf("receipt.json has no result for command %s; the model and the run disagree", c.Name)
		}
		v := AlloyVerdict{Command: c}
		switch c.Kind {
		case "check":
			v.Pass = !res.sat()
			if !v.Pass && detail != nil {
				v.Detail = detail(c.Name)
			}
		default: // run
			v.Pass = res.sat()
			if !v.Pass {
				v.Detail = "no instance within scope: the asserted possibility does not exist in any admissible world (vacuous or contradictory policy)"
			}
		}
		out = append(out, v)
	}
	return out, nil
}

// renderSolutionText compresses a `-t text` solution into one line: skolem
// witnesses first, then each atom with its fields ("(none)" for an atom a
// total-looking field skips, e.g. a teamless subject).
func renderSolutionText(raw string) string {
	type fieldInfo map[string]map[string]struct{} // atom -> values
	sigAtoms := map[string]map[string]struct{}{}
	sigFields := map[string]map[string]fieldInfo{}
	skolemSet := map[string]struct{}{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "$") { // skolem witness: $X_u={User$5}
			skolemSet[strings.TrimPrefix(line, "$")] = struct{}{}
			continue
		}
		if !strings.HasPrefix(line, "this/") {
			continue
		}
		lhs, rest, found := strings.Cut(strings.TrimPrefix(line, "this/"), "={")
		if !found {
			continue
		}
		set := strings.TrimSuffix(rest, "}")
		if sig, field, ok := strings.Cut(lhs, "<:"); ok {
			fields := sigFields[sig]
			if fields == nil {
				fields = map[string]fieldInfo{}
				sigFields[sig] = fields
			}
			values := fields[field]
			if values == nil {
				values = fieldInfo{}
				fields[field] = values
			}
			for _, tuple := range splitSet(set) {
				atom, val, ok := strings.Cut(tuple, "->")
				if !ok {
					continue
				}
				atomValues := values[atom]
				if atomValues == nil {
					atomValues = map[string]struct{}{}
					values[atom] = atomValues
				}
				atomValues[val] = struct{}{}
			}
			continue
		}
		atoms := sigAtoms[lhs]
		if atoms == nil {
			atoms = map[string]struct{}{}
			sigAtoms[lhs] = atoms
		}
		for _, atom := range splitSet(set) {
			atoms[atom] = struct{}{}
		}
	}
	parts := sortedSet(skolemSet)
	sigNames := make([]string, 0, len(sigFields))
	for sig := range sigFields {
		sigNames = append(sigNames, sig)
	}
	sort.Strings(sigNames)
	const maxAtoms = 10
	total := 0
	for _, sig := range sigNames {
		atoms := sortedSet(sigAtoms[sig])
		if len(atoms) == 0 {
			continue
		}
		fieldNames := make([]string, 0, len(sigFields[sig]))
		for field := range sigFields[sig] {
			fieldNames = append(fieldNames, field)
		}
		sort.Strings(fieldNames)
		for _, atom := range atoms {
			if total >= maxAtoms {
				parts = append(parts, "(...)")
				return "counterexample: " + strings.Join(parts, " ")
			}
			var renderedFields []string
			for _, field := range fieldNames {
				values := sortedSet(sigFields[sig][field][atom])
				if len(values) == 0 {
					renderedFields = append(renderedFields, field+"=(none)")
					continue
				}
				for _, value := range values {
					renderedFields = append(renderedFields, field+"="+value)
				}
			}
			parts = append(parts, fmt.Sprintf("%s{%s}", atom, strings.Join(renderedFields, ", ")))
			total++
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "counterexample: " + strings.Join(parts, " ")
}

func sortedSet(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func splitSet(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	for _, x := range strings.Split(s, ",") {
		out = append(out, strings.TrimSpace(x))
	}
	return out
}

func tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
