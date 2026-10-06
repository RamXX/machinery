package hook

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/dirscan"
	"github.com/RamXX/machinery/internal/filelock"
)

// AdoptOptions are the operator's explicit acknowledgements for adoption.
type AdoptOptions struct {
	// RebindIdentity accepts a store whose native directory identity changed
	// (an inode change: the store was moved or restored onto another
	// filesystem). The store's generation must still match the independent
	// initialization marker, so this rebinds the recorded store and never
	// accepts a different one.
	RebindIdentity bool
}

// AdoptState reaffirms an operator-inspected store without discharging any
// project obligation. A parked store can be restored only with its sentinel's
// generation. Live hook writers cause a refusal rather than a partial handoff.
func AdoptState(w io.Writer, root, from string) error {
	return AdoptStateWith(w, root, from, AdoptOptions{})
}

// adoptionRefusal decides, from the store's own identity record, its current
// native identity, and the independent marker, whether adoption may proceed.
// doctor asks the same question so it never prescribes a command that is
// guaranteed to refuse.
func adoptionRefusal(stored stateDirectoryBinding, native string, marker stateDirectoryBinding, restoring bool, opts AdoptOptions) error {
	generationMatches := stored.generation == marker.generation
	if !sameHookNativeIdentity(stored.native, native) {
		if !opts.RebindIdentity {
			hint := ""
			if generationMatches {
				hint = "; its generation matches the independent initialization marker, so if you verified that this is the recorded store moved or restored to another filesystem, rerun with --rebind-identity"
			}
			return fmt.Errorf("store changed native identity (%s versus %s); refusing to accept a replacement store%s", stored.native, native, hint)
		}
		if !generationMatches {
			return fmt.Errorf("store changed native identity (%s versus %s) and its generation %s does not match the independent initialization marker %s; refusing to rebind a different store", stored.native, native, stored.generation, marker.generation)
		}
	}
	if restoring && (!generationMatches || (!sameHookNativeIdentity(stored.native, marker.native) && !opts.RebindIdentity)) {
		if generationMatches {
			return fmt.Errorf("quarantined store does not match the independent initialization marker's directory identity (%s versus %s); if you verified it is the recorded store, rerun with --rebind-identity", stored.native, marker.native)
		}
		return fmt.Errorf("quarantined store does not match the independent initialization marker; refusing to restore a different ledger")
	}
	return nil
}

// AdoptStateWith is AdoptState with explicit operator acknowledgements.
func AdoptStateWith(w io.Writer, root, from string, opts AdoptOptions) (retErr error) {
	if err := operatorGate(); err != nil {
		return err
	}
	root, err := canonicalHookRoot(root)
	if err != nil {
		return err
	}
	dir, err := stateDirPathExact()
	if err != nil {
		return err
	}
	marker, err := stateInitializationMarkerPath()
	if err != nil {
		return err
	}
	markerLock, err := acquireHookFileLock(marker, "hook state adoption")
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, markerLock.Release()) }()
	present, legacy, expected, err := readStateInitializationMarker(marker)
	if err != nil {
		return err
	}
	if !present || legacy {
		return fmt.Errorf("adoption requires an existing bound initialization marker; refusing to discard loss evidence")
	}
	storeLock, err := filelock.Acquire(hookStateStoreScope(dir))
	if err != nil {
		return fmt.Errorf("store is active; stop hook callers before adoption: %w", err)
	}
	defer func() { retErr = errors.Join(retErr, storeLock.Release()) }()
	// Nested atomic publication inventories must not reacquire our store lock.
	hookStateRepairDepth.Add(1)
	defer hookStateRepairDepth.Add(-1)
	source := dir
	if from != "" {
		source, err = filepath.Abs(from)
		if err != nil {
			return err
		}
		if source == dir {
			return fmt.Errorf("restore source must differ from the live store")
		}
		if _, err := os.Lstat(dir); !os.IsNotExist(err) {
			return fmt.Errorf("restore requires the live store to be absent; refusing to overwrite it")
		}
	}
	native, err := captureStateDirectoryIdentity(source)
	if err != nil {
		return err
	}
	identity, err := readBoundedHookFile(filepath.Join(source, stateDirectoryIdentityName), "hook state directory identity", hookStateIdentityMaxBytes)
	if err != nil {
		return err
	}
	if identity == nil {
		return fmt.Errorf("adoption requires the recorded store identity")
	}
	binding, err := parseStateDirectoryBinding(identity.body, "machinery-hook-state-directory-v1")
	if err != nil {
		return err
	}
	if err := adoptionRefusal(binding, native, expected, from != "", opts); err != nil {
		return err
	}
	entries, err := dirscan.Read(source, hookStateDirMaxEntries)
	if err != nil {
		return err
	}
	var projectLocks []*filelock.Lock
	defer func() {
		for i := len(projectLocks) - 1; i >= 0; i-- {
			retErr = errors.Join(retErr, projectLocks[i].Release())
		}
	}()
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".state") {
			continue
		}
		lock, err := filelock.Acquire(filepath.Join(dir, entry.Name()) + ".session")
		if err != nil {
			return fmt.Errorf("project ledger is active; stop hook callers before adoption: %w", err)
		}
		projectLocks = append(projectLocks, lock)
	}
	requestedScope, err := filelock.ScopeIdentity(root)
	if err != nil {
		return err
	}
	requestedName := stateFileName(requestedScope)
	var requested bool
	var summary bytes.Buffer
	for _, entry := range entries {
		name := entry.Name()
		witness, err := readBoundedHookFile(filepath.Join(source, name), "adoption store entry", hookStateMaxBytes)
		if err != nil {
			return err
		}
		if witness == nil {
			return fmt.Errorf("store entry disappeared during adoption: %s", name)
		}
		switch {
		case name == stateDirectoryIdentityName:
		case strings.HasSuffix(name, ".state") && validHookHexDigest(strings.TrimSuffix(name, ".state")):
			record, err := parseHookStateRecord(witness.body)
			if err != nil {
				return err
			}
			if record.root != "" {
				scope, scopeErr := filelock.ScopeIdentity(record.root)
				if scopeErr != nil || stateFileName(scope) != name {
					return fmt.Errorf("ledger %s lacks its canonical root binding", name)
				}
			}
			label := record.root
			if name == requestedName {
				requested = true
				if label == "" {
					label = root + " (legacy)"
				}
			}
			if label == "" {
				label = "ledger " + name + " (root unrecorded)"
			}
			fmt.Fprintf(&summary, "  retained %s: design=%t impl=%t pending=%d routes=%d\n", label, record.design, record.impl, len(record.pending), len(record.routes))
		case strings.Contains(name, ".state.route-") && strings.HasSuffix(name, ".json"):
			base, suffix, ok := strings.Cut(name, ".state.route-")
			if !ok || !validHookHexDigest(base) || !validHookHexDigest(strings.TrimSuffix(suffix, ".json")) {
				return fmt.Errorf("noncanonical route snapshot %s", name)
			}
			if _, err := decodeConfig(witness.body); err != nil {
				return err
			}
		default:
			return fmt.Errorf("store contains unrecognized or interrupted entry %s; recover it before adoption", name)
		}
	}
	current, err := captureStateDirectoryIdentity(source)
	if err != nil {
		return err
	}
	if current != native {
		return fmt.Errorf("store identity changed during adoption")
	}
	after, err := dirscan.Read(source, hookStateDirMaxEntries)
	if err != nil {
		return err
	}
	if len(after) != len(entries) {
		return fmt.Errorf("store inventory changed during adoption; stop hook callers and retry")
	}
	for i := range entries {
		if entries[i].Name() != after[i].Name() {
			return fmt.Errorf("store inventory changed during adoption; stop hook callers and retry")
		}
	}
	// The journal records the operator's requested binding before any mutation.
	// A failed publication leaves the store blocked and can be retried safely.
	journal := marker + ".handoffs"
	prior, err := readBoundedHookFile(journal, "hook state handoff journal", 1<<20)
	if err != nil {
		return err
	}
	var history []byte
	if prior != nil {
		history = prior.body
	}
	event, err := json.Marshal(map[string]string{"time": time.Now().UTC().Format(time.RFC3339Nano), "root": root, "source": source, "directory": native, "generation": binding.generation, "prior_directory": expected.native, "prior_generation": expected.generation})
	if err != nil {
		return err
	}
	history = append(history, append(event, '\n')...)
	if len(history) > 1<<20 {
		return fmt.Errorf("handoff journal exceeds its bounded capacity")
	}
	if err := writeAdoptionFile(journal, history); err != nil {
		return err
	}
	if from != "" {
		if err := os.Rename(source, dir); err != nil {
			return err
		}
		if err := syncStateDirectory(filepath.Dir(dir)); err != nil {
			return err
		}
	}
	if from != "" {
		// a rename within one filesystem keeps the directory; bind what is live
		if native, err = captureStateDirectoryIdentity(dir); err != nil {
			return err
		}
	}
	priorNative := expected.native
	binding.native = native
	if err := writeAdoptionFile(filepath.Join(dir, stateDirectoryIdentityName), binding.identityBody()); err != nil {
		return err
	}
	if err := writeStateInitializationMarker(marker, binding, true); err != nil {
		return err
	}
	fmt.Fprintf(w, "Adopted governance hook state store %s; initialization marker retained.\n", dir)
	if priorNative != native {
		fmt.Fprintf(w, "  rebound store identity: %s -> %s\n", priorNative, native)
	}
	if !requested {
		fmt.Fprintf(w, "  no recorded obligations for %s\n", root)
	}
	_, err = w.Write(summary.Bytes())
	return err
}

func writeAdoptionFile(target string, body []byte) (retErr error) {
	file, err := os.CreateTemp(filepath.Dir(target), ".hook-adopt-")
	if err != nil {
		return err
	}
	temp := file.Name()
	defer func() {
		if err := os.Remove(temp); err != nil && !os.IsNotExist(err) {
			retErr = errors.Join(retErr, err)
		}
	}()
	if err := file.Chmod(0600); err != nil {
		return errors.Join(err, file.Close())
	}
	if _, err := file.Write(body); err != nil {
		return errors.Join(err, file.Close())
	}
	if err := file.Sync(); err != nil {
		return errors.Join(err, file.Close())
	}
	if err := file.Close(); err != nil {
		return err
	}
	return replaceStateFile(temp, target)
}
