package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	executableReceiptVersion = 2
	maximumExecutableSize    = 128 << 20
	maximumSymlinkDepth      = 16
)

type executableSnapshotReceipt struct {
	Version  int               `json:"version"`
	Source   string            `json:"source"`
	Target   string            `json:"target"`
	Chain    []symlinkWitness  `json:"chain"`
	Snapshot string            `json:"snapshot"`
	Original executableWitness `json:"original"`
	Copy     executableWitness `json:"copy"`
}

type symlinkWitness struct {
	Path     string `json:"path"`
	LinkText string `json:"link_text"`
	Digest   string `json:"digest"`
	Identity string `json:"identity"`
	Change   string `json:"change"`
	Mode     uint32 `json:"mode"`
	Size     int64  `json:"size"`
	ModTime  int64  `json:"mod_time_unix_nano"`
}

type executableWitness struct {
	Digest   string `json:"digest"`
	Identity string `json:"identity"`
	Change   string `json:"change"`
	Mode     uint32 `json:"mode"`
	Size     int64  `json:"size"`
	ModTime  int64  `json:"mod_time_unix_nano"`
}

func snapshotExecutableCommand(args []string, stderr io.Writer) int {
	flags := flag.NewFlagSet("run-safe snapshot-executable", flag.ContinueOnError)
	flags.SetOutput(stderr)
	source := flags.String("source", "", "original executable path")
	destination := flags.String("destination", "", "private executable snapshot path")
	receiptPath := flags.String("receipt", "", "snapshot receipt path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 || *source == "" || *destination == "" || *receiptPath == "" {
		fmt.Fprintln(stderr, "run-safe snapshot-executable: require -source, -destination, and -receipt")
		return 2
	}
	if err := createExecutableSnapshot(*source, *destination, *receiptPath); err != nil {
		fmt.Fprintf(stderr, "run-safe snapshot-executable: %v\n", err)
		return 1
	}
	return 0
}

func verifyExecutableCommand(args []string, stderr io.Writer) int {
	flags := flag.NewFlagSet("run-safe verify-executable", flag.ContinueOnError)
	flags.SetOutput(stderr)
	receiptPath := flags.String("receipt", "", "snapshot receipt path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 || *receiptPath == "" {
		fmt.Fprintln(stderr, "run-safe verify-executable: require -receipt")
		return 2
	}
	receipt, err := loadExecutableReceipt(*receiptPath)
	if err == nil {
		err = receipt.validateAll()
	}
	if err != nil {
		fmt.Fprintf(stderr, "run-safe verify-executable: %v\n", err)
		return 1
	}
	return 0
}

func createExecutableSnapshot(source, destination, receiptPath string) error {
	sourceAbs, err := filepath.Abs(source)
	if err != nil {
		return fmt.Errorf("resolve source executable: %w", err)
	}
	destinationAbs, err := filepath.Abs(destination)
	if err != nil {
		return fmt.Errorf("resolve executable snapshot: %w", err)
	}
	receiptAbs, err := filepath.Abs(receiptPath)
	if err != nil {
		return fmt.Errorf("resolve executable receipt: %w", err)
	}
	if sourceAbs == destinationAbs || sourceAbs == receiptAbs || destinationAbs == receiptAbs {
		return fmt.Errorf("source, snapshot, and receipt paths must be distinct")
	}
	chain, target, err := captureExecutableChain(sourceAbs)
	if err != nil {
		return fmt.Errorf("resolve source executable chain: %w", err)
	}
	body, original, err := captureExecutable(target)
	if err != nil {
		return fmt.Errorf("capture source executable: %w", err)
	}
	if err := writeExclusiveSynced(destinationAbs, body, 0o700); err != nil {
		return fmt.Errorf("write executable snapshot: %w", err)
	}
	_, copyWitness, err := captureExecutable(destinationAbs)
	if err != nil {
		return fmt.Errorf("verify executable snapshot: %w", err)
	}
	currentChain, currentTarget, err := captureExecutableChain(sourceAbs)
	if err != nil || currentTarget != target || !equalSymlinkChains(currentChain, chain) {
		return errors.Join(err, fmt.Errorf("source executable symlink chain changed while snapshotting"))
	}
	_, currentOriginal, err := captureExecutable(target)
	if err != nil || currentOriginal != original {
		return errors.Join(err, fmt.Errorf("source executable changed while snapshotting"))
	}
	receipt := executableSnapshotReceipt{
		Version: executableReceiptVersion, Source: sourceAbs, Target: target, Chain: chain, Snapshot: destinationAbs,
		Original: original, Copy: copyWitness,
	}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if err := writeExclusiveSynced(receiptAbs, encoded, 0o600); err != nil {
		return fmt.Errorf("write executable snapshot receipt: %w", err)
	}
	return nil
}

func loadExecutableReceipt(path string) (*executableSnapshotReceipt, error) {
	body, err := readStrictRegular(path, 8192)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var receipt executableSnapshotReceipt
	if err := decoder.Decode(&receipt); err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("receipt contains trailing content")
	}
	canonical, err := json.Marshal(receipt)
	if err != nil {
		return nil, err
	}
	canonical = append(canonical, '\n')
	if !bytes.Equal(canonical, body) || receipt.Version != executableReceiptVersion ||
		!filepath.IsAbs(receipt.Source) || !filepath.IsAbs(receipt.Target) || !filepath.IsAbs(receipt.Snapshot) ||
		!validSymlinkChain(receipt.Source, receipt.Target, receipt.Chain) ||
		!validExecutableWitness(receipt.Original) || !validExecutableWitness(receipt.Copy) {
		return nil, fmt.Errorf("receipt is not canonical or supported")
	}
	return &receipt, nil
}

func (receipt *executableSnapshotReceipt) validateSnapshot() error {
	_, current, err := captureExecutable(receipt.Snapshot)
	if err != nil {
		return err
	}
	if current != receipt.Copy {
		return fmt.Errorf("executable snapshot changed identity, metadata, or content")
	}
	return nil
}

func (receipt *executableSnapshotReceipt) validateAll() error {
	if err := receipt.validateSnapshot(); err != nil {
		return err
	}
	chain, target, err := captureExecutableChain(receipt.Source)
	if err != nil {
		return fmt.Errorf("revalidate source executable chain: %w", err)
	}
	if target != receipt.Target || !equalSymlinkChains(chain, receipt.Chain) {
		return fmt.Errorf("source executable symlink chain changed identity, metadata, or link target")
	}
	_, current, err := captureExecutable(receipt.Target)
	if err != nil {
		return fmt.Errorf("revalidate source executable: %w", err)
	}
	if current != receipt.Original {
		return fmt.Errorf("source executable changed identity, metadata, or content")
	}
	return nil
}

func captureExecutableChain(source string) ([]symlinkWitness, string, error) {
	current := filepath.Clean(source)
	seen := make(map[string]bool)
	chain := make([]symlinkWitness, 0, 2)
	for depth := 0; depth <= maximumSymlinkDepth; depth++ {
		if seen[current] {
			return nil, "", fmt.Errorf("source executable symlink chain contains a cycle at %s", current)
		}
		seen[current] = true
		before, err := os.Lstat(current)
		if err != nil {
			return nil, "", err
		}
		if before.Mode()&os.ModeSymlink == 0 {
			if !before.Mode().IsRegular() {
				return nil, "", fmt.Errorf("source executable target %s is special (%s)", current, before.Mode())
			}
			return chain, current, nil
		}
		if depth == maximumSymlinkDepth {
			return nil, "", fmt.Errorf("source executable symlink chain exceeds %d links", maximumSymlinkDepth)
		}
		linkText, err := os.Readlink(current)
		if err != nil || linkText == "" || strings.IndexByte(linkText, 0) >= 0 {
			return nil, "", errors.Join(err, fmt.Errorf("source executable symlink %s has an invalid target", current))
		}
		identity, change, err := nativeSymlinkWitness(current, before)
		if err != nil {
			return nil, "", err
		}
		after, err := os.Lstat(current)
		if err != nil || !os.SameFile(before, after) || after.Mode() != before.Mode() || after.Size() != before.Size() ||
			!after.ModTime().Equal(before.ModTime()) {
			return nil, "", errors.Join(err, fmt.Errorf("source executable symlink %s changed while resolving", current))
		}
		secondText, err := os.Readlink(current)
		if err != nil || secondText != linkText {
			return nil, "", errors.Join(err, fmt.Errorf("source executable symlink %s changed link target while resolving", current))
		}
		afterIdentity, afterChange, err := nativeSymlinkWitness(current, after)
		if err != nil || afterIdentity != identity || afterChange != change {
			return nil, "", errors.Join(err, fmt.Errorf("source executable symlink %s changed native identity while resolving", current))
		}
		linkDigest := sha256.Sum256([]byte(linkText))
		chain = append(chain, symlinkWitness{
			Path: current, LinkText: linkText, Digest: "sha256:" + hex.EncodeToString(linkDigest[:]), Identity: identity, Change: change,
			Mode: uint32(after.Mode()), Size: after.Size(), ModTime: after.ModTime().UnixNano(),
		})
		if filepath.IsAbs(linkText) {
			current = filepath.Clean(linkText)
		} else {
			current = filepath.Clean(filepath.Join(filepath.Dir(current), linkText))
		}
		if !filepath.IsAbs(current) {
			return nil, "", fmt.Errorf("source executable symlink escaped absolute path resolution")
		}
	}
	return nil, "", fmt.Errorf("source executable symlink resolution failed closed")
}

func validSymlinkChain(source, target string, chain []symlinkWitness) bool {
	if chain == nil || filepath.Clean(source) != source || filepath.Clean(target) != target || len(chain) > maximumSymlinkDepth {
		return false
	}
	current := source
	for _, entry := range chain {
		if entry.Path != current || !filepath.IsAbs(entry.Path) || entry.LinkText == "" || !validSHA256(entry.Digest) || entry.Identity == "" || entry.Change == "" || entry.Size < 0 ||
			os.FileMode(entry.Mode)&os.ModeSymlink == 0 {
			return false
		}
		linkDigest := sha256.Sum256([]byte(entry.LinkText))
		if entry.Digest != "sha256:"+hex.EncodeToString(linkDigest[:]) {
			return false
		}
		if filepath.IsAbs(entry.LinkText) {
			current = filepath.Clean(entry.LinkText)
		} else {
			current = filepath.Clean(filepath.Join(filepath.Dir(current), entry.LinkText))
		}
	}
	return current == target
}

func equalSymlinkChains(left, right []symlinkWitness) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func captureExecutable(path string) (body []byte, witness executableWitness, retErr error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, executableWitness{}, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() || before.Size() < 0 || before.Size() > maximumExecutableSize {
		return nil, executableWitness{}, fmt.Errorf("%s must be a regular non-symlink file no larger than %d bytes", path, maximumExecutableSize)
	}
	if runtime.GOOS != "windows" && before.Mode().Perm()&0o111 == 0 {
		return nil, executableWitness{}, fmt.Errorf("%s has no execute bit", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, executableWitness{}, err
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(before, opened) || opened.Mode() != before.Mode() {
		return nil, executableWitness{}, errors.Join(err, fmt.Errorf("executable changed identity while opening"))
	}
	body, err = io.ReadAll(io.LimitReader(file, maximumExecutableSize+1))
	if err != nil || len(body) > maximumExecutableSize {
		return nil, executableWitness{}, errors.Join(err, fmt.Errorf("executable exceeds bounded snapshot size"))
	}
	identity, change, err := nativeExecutableWitness(file, opened)
	if err != nil {
		return nil, executableWitness{}, err
	}
	after, err := os.Lstat(path)
	if err != nil || !os.SameFile(before, after) || after.Mode() != before.Mode() || after.Size() != before.Size() ||
		!after.ModTime().Equal(before.ModTime()) {
		return nil, executableWitness{}, errors.Join(err, fmt.Errorf("executable changed while reading"))
	}
	digest := sha256.Sum256(body)
	witness = executableWitness{
		Digest: "sha256:" + hex.EncodeToString(digest[:]), Identity: identity, Change: change,
		Mode: uint32(after.Mode()), Size: after.Size(), ModTime: after.ModTime().UnixNano(),
	}
	return body, witness, nil
}

func validExecutableWitness(w executableWitness) bool {
	mode := os.FileMode(w.Mode)
	if !validSHA256(w.Digest) || w.Identity == "" || w.Change == "" || w.Size < 0 || w.Size > maximumExecutableSize ||
		!mode.IsRegular() || runtime.GOOS != "windows" && mode.Perm()&0o111 == 0 {
		return false
	}
	return true
}

func validSHA256(value string) bool {
	if len(value) != len("sha256:")+sha256.Size*2 || !strings.HasPrefix(value, "sha256:") || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func readStrictRegular(path string, limit int64) ([]byte, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() || before.Size() > limit {
		return nil, fmt.Errorf("%s must be a bounded regular non-symlink file", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	opened, statErr := file.Stat()
	body, readErr := io.ReadAll(io.LimitReader(file, limit+1))
	closeErr := file.Close()
	after, pathErr := os.Lstat(path)
	if err := errors.Join(statErr, readErr, closeErr, pathErr); err != nil {
		return nil, err
	}
	if int64(len(body)) > limit || !os.SameFile(before, opened) || !os.SameFile(before, after) ||
		opened.Mode() != before.Mode() || after.Mode() != before.Mode() || opened.Size() != before.Size() || after.Size() != before.Size() ||
		!opened.ModTime().Equal(before.ModTime()) || !after.ModTime().Equal(before.ModTime()) {
		return nil, fmt.Errorf("%s changed while reading", path)
	}
	return body, nil
}

func writeExclusiveSynced(path string, body []byte, mode os.FileMode) (retErr error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()
	if _, err := file.Write(body); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	return nil
}
