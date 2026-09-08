//go:build unix

package tdd

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

// TestInodeKeyIdentifiesHardlinkAliasesAcrossStatFieldWidths pins the
// portable identity contract of inodeKey. syscall.Stat_t disagrees on the
// width and signedness of Dev across the unix targets this package builds
// for, so the widening must be exact on every one of them: two names for one
// inode share a key, two distinct files do not.
func TestInodeKeyIdentifiesHardlinkAliasesAcrossStatFieldWidths(t *testing.T) {
	dir := t.TempDir()
	original := filepath.Join(dir, "original")
	if err := os.WriteFile(original, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(dir, "alias")
	if err := os.Link(original, alias); err != nil {
		t.Skipf("hardlinks unavailable: %v", err)
	}
	other := filepath.Join(dir, "other")
	if err := os.WriteFile(other, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	key := func(path string) inodeIdentity {
		t.Helper()
		info, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		identity, ok := inodeKey(info)
		if !ok {
			t.Fatalf("no inode identity for %s", path)
		}
		return identity
	}
	if key(original) != key(alias) {
		t.Fatalf("hardlink alias did not share an inode identity: %+v vs %+v", key(original), key(alias))
	}
	if key(original) == key(other) {
		t.Fatal("distinct files reported the same inode identity")
	}
	if key(original).ino == 0 {
		t.Fatal("inode identity lost the inode field")
	}
}

// TestWidenIdentityFieldIsExactForEverySupportedStatWidth freezes the
// widening itself: signed darwin-shaped fields sign-extend deterministically
// and unsigned linux-shaped fields pass through unchanged.
func TestWidenIdentityFieldIsExactForEverySupportedStatWidth(t *testing.T) {
	if got := widenIdentityField(int32(-1)); got != math.MaxUint64 {
		t.Errorf("signed 32-bit widening = %#x, want %#x", got, uint64(math.MaxUint64))
	}
	if got := widenIdentityField(int32(16777232)); got != 16777232 {
		t.Errorf("signed 32-bit widening = %d, want 16777232", got)
	}
	if got := widenIdentityField(uint64(math.MaxUint64)); got != math.MaxUint64 {
		t.Errorf("unsigned 64-bit widening = %#x, want %#x", got, uint64(math.MaxUint64))
	}
	if widenIdentityField(int32(-1)) == widenIdentityField(int32(-2)) {
		t.Error("distinct signed device numbers collided after widening")
	}
}
