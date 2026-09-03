package fsatomic

import (
	"encoding/base64"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenameNoReplacePreservesDestinationCollision(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "source"), []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "destination"), []byte("destination"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := RenameNoReplace(root, "source", "destination"); err == nil {
		t.Fatal("expected destination collision")
	}
	for _, file := range []struct{ name, want string }{{"source", "source"}, {"destination", "destination"}} {
		body, err := os.ReadFile(filepath.Join(dir, file.name))
		if err != nil || string(body) != file.want {
			t.Fatalf("%s = %q, %v; want %q", file.name, body, err, file.want)
		}
	}
}

func TestRenameNoReplaceOpensRealParentsAndUsesBasenames(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "source-parent"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "destination-parent"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "source-parent", "source"), []byte("owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := RenameNoReplace(root, filepath.Join("source-parent", "source"), filepath.Join("destination-parent", "destination")); err != nil {
		t.Fatal(err)
	}
	if body, err := os.ReadFile(filepath.Join(dir, "destination-parent", "destination")); err != nil || string(body) != "owned" {
		t.Fatalf("nested destination = %q, %v", body, err)
	}
}

func TestRenameNoReplaceRejectsSymlinkedParent(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "real"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "real", "source"), []byte("owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real", filepath.Join(dir, "alias")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := RenameNoReplace(root, filepath.Join("alias", "source"), "destination"); err == nil || !strings.Contains(err.Error(), "must be a real directory") {
		t.Fatalf("symlinked parent error = %v", err)
	}
	if body, err := os.ReadFile(filepath.Join(dir, "real", "source")); err != nil || string(body) != "owned" {
		t.Fatalf("source behind rejected symlink = %q, %v", body, err)
	}
}

func TestRenameNoReplaceRejectsTraversalAndAbsolutePaths(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	for _, names := range [][2]string{{"../source", "destination"}, {"source", "../destination"}, {"/source", "destination"}, {"source", "/destination"}, {"source", `C:\target`}, {".", "destination"}} {
		if err := RenameNoReplace(root, names[0], names[1]); err == nil || !strings.Contains(err.Error(), "clean relative path") {
			t.Fatalf("RenameNoReplace(%q, %q) error = %v", names[0], names[1], err)
		}
	}
}

func TestResumeQuarantineRejectsSymlinkAndMalformedProvenance(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	encoded := base64.RawURLEncoding.EncodeToString([]byte("source"))
	canonical := ".private-" + strings.Repeat("0a", quarantineNonceBytes) + quarantineSourceMarker + encoded
	if err := os.Mkdir(filepath.Join(dir, "target"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target", filepath.Join(dir, canonical)); err == nil {
		if _, err := ResumeQuarantine(root, canonical, "source"); err == nil || !strings.Contains(err.Error(), "real directory") {
			t.Fatalf("symlink quarantine error = %v", err)
		}
	} else {
		t.Logf("symlink case unavailable: %v", err)
	}
	for _, malformed := range []string{
		".private-" + strings.Repeat("0A", quarantineNonceBytes) + quarantineSourceMarker + encoded,
		".private-" + strings.Repeat("0a", quarantineNonceBytes) + quarantineSourceMarker + encoded + "=",
		".private-short" + quarantineSourceMarker + encoded,
	} {
		if _, err := ResumeQuarantine(root, malformed, ""); err == nil {
			t.Fatalf("ResumeQuarantine(%q) unexpectedly succeeded", malformed)
		}
	}
}

func TestQuarantineDeletionCannotDeletePublicReplacement(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "source"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "source", "owned"), []byte("owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	q, err := Quarantine(root, "source", ".private-")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "source"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "source", "replacement"), []byte("replacement"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := q.RemoveAll(); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "source", "replacement"))
	if err != nil || string(body) != "replacement" {
		t.Fatalf("public replacement = %q, %v", body, err)
	}
}

func TestQuarantinePreservesDirectoryReplacementBeforeSourceMove(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "source"), []byte("owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	originalHook := fsatomicBeforeQuarantineMove
	t.Cleanup(func() { fsatomicBeforeQuarantineMove = originalHook })
	var replacement string
	fsatomicBeforeQuarantineMove = func(directory string) error {
		kept := directory + ".kept"
		if err := root.Rename(directory, kept); err != nil {
			return err
		}
		if err := root.Mkdir(directory, 0o700); err != nil {
			return err
		}
		replacement = directory
		return root.WriteFile(filepath.Join(directory, "replacement"), []byte("replacement"), 0o600)
	}
	if _, err := Quarantine(root, "source", ".private-"); err == nil || !strings.Contains(err.Error(), "changed before source move") {
		t.Fatalf("pre-move replacement error = %v", err)
	}
	if body, err := os.ReadFile(filepath.Join(dir, "source")); err != nil || string(body) != "owned" {
		t.Fatalf("source after rejected pre-move replacement = %q, %v", body, err)
	}
	if body, err := os.ReadFile(filepath.Join(dir, replacement, "replacement")); err != nil || string(body) != "replacement" {
		t.Fatalf("quarantine replacement = %q, %v", body, err)
	}
}

func TestQuarantineRestoresSourceAfterDirectoryReplacementFollowingMove(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "source"), []byte("owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	originalHook := fsatomicAfterQuarantineMove
	t.Cleanup(func() { fsatomicAfterQuarantineMove = originalHook })
	var replacement string
	fsatomicAfterQuarantineMove = func(directory string) {
		kept := directory + ".kept"
		if err := root.Rename(directory, kept); err != nil {
			t.Fatal(err)
		}
		if err := root.Mkdir(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		replacement = directory
		if err := root.WriteFile(filepath.Join(directory, "replacement"), []byte("replacement"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Quarantine(root, "source", ".private-"); err == nil || !strings.Contains(err.Error(), "changed after source move") {
		t.Fatalf("post-move replacement error = %v", err)
	}
	if body, err := os.ReadFile(filepath.Join(dir, "source")); err != nil || string(body) != "owned" {
		t.Fatalf("source after rejected post-move replacement = %q, %v", body, err)
	}
	if body, err := os.ReadFile(filepath.Join(dir, replacement, "replacement")); err != nil || string(body) != "replacement" {
		t.Fatalf("quarantine replacement = %q, %v", body, err)
	}
}

func TestQuarantineRestoreNeverClobbersPublicReplacement(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "source"), []byte("owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	q, err := Quarantine(root, "source", ".private-")
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()
	if err := os.WriteFile(filepath.Join(dir, "source"), []byte("replacement"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := q.Restore(); err == nil {
		t.Fatal("expected no-clobber restore failure")
	}
	body, err := os.ReadFile(filepath.Join(dir, "source"))
	if err != nil || string(body) != "replacement" {
		t.Fatalf("public replacement = %q, %v", body, err)
	}
	body, err = q.Root().ReadFile(q.Name())
	if err != nil || string(body) != "owned" {
		t.Fatalf("quarantined authority = %q, %v", body, err)
	}
}

func TestQuarantineCloseEmptyPreservesReplacedPrivateDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "source"), []byte("owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	q, err := Quarantine(root, "source", ".private-")
	if err != nil {
		t.Fatal(err)
	}
	if err := q.private.Remove(q.entry); err != nil {
		t.Fatal(err)
	}
	kept := q.directory + ".kept"
	if err := root.Rename(q.directory, kept); err != nil {
		t.Fatal(err)
	}
	if err := root.Mkdir(q.directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := root.WriteFile(filepath.Join(q.directory, "replacement"), []byte("replacement"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := q.closeEmpty(); err == nil || !strings.Contains(err.Error(), "changed identity") {
		t.Fatalf("closeEmpty replacement error = %v", err)
	}
	body, err := root.ReadFile(filepath.Join(q.directory, "replacement"))
	if err != nil || string(body) != "replacement" {
		t.Fatalf("replacement private directory = %q, %v", body, err)
	}
}

func TestQuarantineCloseEmptyRevalidatesAfterProtocolCheck(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "source"), []byte("owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	q, err := Quarantine(root, "source", ".private-")
	if err != nil {
		t.Fatal(err)
	}
	if err := q.private.Remove(q.entry); err != nil {
		t.Fatal(err)
	}
	originalHook := fsatomicBeforePrivateRemove
	t.Cleanup(func() { fsatomicBeforePrivateRemove = originalHook })
	fsatomicBeforePrivateRemove = func() {
		fsatomicBeforePrivateRemove = nil
		kept := q.directory + ".kept"
		if err := root.Rename(q.directory, kept); err != nil {
			t.Fatal(err)
		}
		if err := root.Mkdir(q.directory, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := root.WriteFile(filepath.Join(q.directory, "replacement"), []byte("replacement"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := q.closeEmpty(); err == nil || !strings.Contains(err.Error(), "retirement isolation") {
		t.Fatalf("closeEmpty post-check replacement error = %v", err)
	}
	defer q.Close()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), ".private-") || !entry.IsDir() {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, entry.Name(), "replacement"))
		if err == nil && string(body) == "replacement" {
			found = true
		}
	}
	if !found {
		t.Fatal("post-check replacement private directory was not preserved")
	}
}

func TestQuarantineNameCollisionsAreDeterministicAndNonClobbering(t *testing.T) {
	dir := t.TempDir()
	source := "source"
	prefix := ".private-"
	encoded := base64.RawURLEncoding.EncodeToString([]byte(source))
	collision := prefix + strings.Repeat("00", 16) + quarantineSourceMarker + encoded
	if err := os.Mkdir(filepath.Join(dir, collision), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, collision, "sentinel"), []byte("collision"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, source), []byte("owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	originalRandom := fsatomicRandomRead
	t.Cleanup(func() { fsatomicRandomRead = originalRandom })
	calls := byte(0)
	fsatomicRandomRead = func(body []byte) (int, error) {
		for i := range body {
			body[i] = calls
		}
		calls++
		return len(body), nil
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	q, err := Quarantine(root, source, prefix)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(q.directory, strings.Repeat("00", 16)+quarantineSourceMarker) {
		t.Fatalf("quarantine reused colliding name %s", q.directory)
	}
	if err := q.Remove(); err != nil {
		t.Fatal(err)
	}
	if body, err := os.ReadFile(filepath.Join(dir, collision, "sentinel")); err != nil || string(body) != "collision" {
		t.Fatalf("collision sentinel = %q, %v", body, err)
	}
}

func TestQuarantineRetirementCollisionsAreDeterministicAndNonClobbering(t *testing.T) {
	dir := t.TempDir()
	source := "source"
	prefix := ".private-"
	encoded := base64.RawURLEncoding.EncodeToString([]byte(source))
	collision := prefix + strings.Repeat("01", quarantineNonceBytes) + quarantineSourceMarker + encoded
	if err := os.Mkdir(filepath.Join(dir, collision), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, collision, "sentinel"), []byte("collision"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, source), []byte("owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	originalRandom := fsatomicRandomRead
	t.Cleanup(func() { fsatomicRandomRead = originalRandom })
	calls := byte(0)
	fsatomicRandomRead = func(body []byte) (int, error) {
		for i := range body {
			body[i] = calls
		}
		calls++
		return len(body), nil
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	q, err := Quarantine(root, source, prefix)
	if err != nil {
		t.Fatal(err)
	}
	if err := q.Remove(); err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Fatalf("random source calls = %d, want 3 (create, collided retirement, successful retirement)", calls)
	}
	if body, err := os.ReadFile(filepath.Join(dir, collision, "sentinel")); err != nil || string(body) != "collision" {
		t.Fatalf("retirement collision sentinel = %q, %v", body, err)
	}
}

func TestQuarantineRejectsShortRandomReadWithoutCreatingAuthority(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "source"), []byte("owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	originalRandom := fsatomicRandomRead
	t.Cleanup(func() { fsatomicRandomRead = originalRandom })
	fsatomicRandomRead = func(body []byte) (int, error) {
		body[0] = 1
		return 1, nil
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if _, err := Quarantine(root, "source", ".private-"); err == nil || !strings.Contains(err.Error(), "returned 1 of 16 bytes") {
		t.Fatalf("short random read error = %v", err)
	}
	if body, err := os.ReadFile(filepath.Join(dir, "source")); err != nil || string(body) != "owned" {
		t.Fatalf("source after short random read = %q, %v", body, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "source" {
		t.Fatalf("directory after short random read = %v", entries)
	}
}

func TestQuarantineRetirementFailureKeepsRecoverableProvenance(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "source"), []byte("owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	q, err := Quarantine(root, "source", ".private-")
	if err != nil {
		t.Fatal(err)
	}
	originalRemove := fsatomicRemovePrivateDirectory
	removeErr := errors.New("injected retirement failure")
	fsatomicRemovePrivateDirectory = func(*os.Root, *os.Root, string) error { return removeErr }
	if err := q.Remove(); !errors.Is(err, removeErr) {
		t.Fatalf("retirement failure = %v", err)
	}
	fsatomicRemovePrivateDirectory = originalRemove
	t.Cleanup(func() { fsatomicRemovePrivateDirectory = originalRemove })
	retiredName := q.directory
	if err := q.Close(); err != nil {
		t.Fatal(err)
	}
	resumed, err := ResumeQuarantine(root, retiredName, "")
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Source() != "source" {
		t.Fatalf("recovered source = %q", resumed.Source())
	}
	if _, err := resumed.Root().Lstat(resumed.Name()); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("retired object exists: %v", err)
	}
	if err := resumed.FinishEmpty(); err != nil {
		t.Fatal(err)
	}
	if _, err := root.Lstat(retiredName); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("retired quarantine remains: %v", err)
	}
}

func TestQuarantineRequiresExactPrivateInventoryBeforeRetirement(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "source"), []byte("owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	q, err := Quarantine(root, "source", ".private-")
	if err != nil {
		t.Fatal(err)
	}
	if err := q.Root().WriteFile("foreign", []byte("foreign"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := q.Remove(); err == nil || !strings.Contains(err.Error(), "inventory is not empty") {
		t.Fatalf("foreign private inventory error = %v", err)
	}
	name := q.directory
	if err := q.Close(); err != nil {
		t.Fatal(err)
	}
	resumed, err := ResumeQuarantine(root, name, "source")
	if err != nil {
		t.Fatal(err)
	}
	if err := resumed.Root().Remove("foreign"); err != nil {
		t.Fatal(err)
	}
	if err := resumed.FinishEmpty(); err != nil {
		t.Fatal(err)
	}
}

func TestQuarantinePreservesPostInventoryAppenderForRecovery(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "source"), []byte("owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	q, err := Quarantine(root, "source", ".private-")
	if err != nil {
		t.Fatal(err)
	}
	originalHook := fsatomicBeforePrivateRemove
	t.Cleanup(func() { fsatomicBeforePrivateRemove = originalHook })
	fsatomicBeforePrivateRemove = func() {
		fsatomicBeforePrivateRemove = nil
		if err := q.Root().WriteFile("appended", []byte("concurrent"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := q.Remove(); err == nil {
		t.Fatal("post-inventory appender unexpectedly permitted retirement")
	}
	retiredName := q.directory
	if err := q.Close(); err != nil {
		t.Fatal(err)
	}
	resumed, err := ResumeQuarantine(root, retiredName, "")
	if err != nil {
		t.Fatal(err)
	}
	if body, err := resumed.Root().ReadFile("appended"); err != nil || string(body) != "concurrent" {
		t.Fatalf("preserved appended file = %q, %v", body, err)
	}
	if err := resumed.Root().Remove("appended"); err != nil {
		t.Fatal(err)
	}
	if err := resumed.FinishEmpty(); err != nil {
		t.Fatal(err)
	}
}
