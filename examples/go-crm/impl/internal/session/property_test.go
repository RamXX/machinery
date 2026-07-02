package session_test

// Property tests for the four invariants owned by crm.session (BUILD.md 7.3):
// P-password-hashed, P-username-unique, P-disabled-cannot-auth,
// P-session-active-user. Each seeds via Register/Login (RED against the stubs).

import (
	"errors"
	"strings"
	"testing"

	"crm/internal/model"
	"crm/internal/session"
)

// P-password-hashed (BUILD.md 7.3, corrected after the design round-trip): for
// any non-degenerate password the persisted passwordHash is a valid argon2id PHC
// encoding, never equals the plaintext, and the plaintext does not appear in the
// credential material (the salt and derived-key segments). The fixed algorithm
// and parameter prefix is metadata and is excluded from the leak check; a
// single-character fixture is excluded because a coincidental base64 substring is
// not a leak.
func TestPropPasswordHashed(t *testing.T) {
	for i, pw := range []string{"password", "correct horse battery staple", "p@$$w0rd!", "hunter2-guest"} {
		s, f, _ := newSess(t)
		name := "user" + string(rune('A'+i))
		reg(t, s, f, name, pw, model.RoleRep)
		tx, _ := f.Open("mem")
		u, err := f.GetUserByName(tx, name)
		if err != nil {
			t.Fatalf("P-password-hashed: user must be stored: %v", err)
		}
		if !strings.HasPrefix(u.PasswordHash, "$argon2id$") {
			t.Errorf("P-password-hashed: %q not an argon2id encoding", u.PasswordHash)
		}
		if u.PasswordHash == pw {
			t.Errorf("P-password-hashed: hash equals plaintext %q", pw)
		}
		// PHC form: $argon2id$v=..$m=..,t=..,p=..$<saltB64>$<hashB64>. The last
		// two "$" segments are the credential material; the prefix is metadata.
		seg := strings.Split(u.PasswordHash, "$")
		if len(seg) < 6 {
			t.Fatalf("P-password-hashed: malformed PHC encoding %q", u.PasswordHash)
		}
		material := seg[len(seg)-2] + seg[len(seg)-1]
		if strings.Contains(material, pw) {
			t.Errorf("P-password-hashed: plaintext %q leaked into credential material of %q", pw, u.PasswordHash)
		}
	}
}

// P-username-unique: two register attempts with the same username -> the second
// fails; the store never holds two Users with one username.
func TestPropUsernameUnique(t *testing.T) {
	s, f, _ := newSess(t)
	reg(t, s, f, "sam", "pw", model.RoleRep) // first must succeed (RED on stub)
	tx, _ := f.Open("mem")
	if _, err := session.Impl(s).Register(tx, "sam", "pw2", model.RoleRep); err == nil {
		t.Errorf("P-username-unique: duplicate username must be rejected")
	}
	count := 0
	for _, u := range f.Users {
		if u.Username == "sam" {
			count++
		}
	}
	if count > 1 {
		t.Errorf("P-username-unique: store holds %d users named sam", count)
	}
}

// P-disabled-cannot-auth: for any Disabled user and any password, Login never
// yields a Session (ErrDisabled).
func TestPropDisabledCannotAuth(t *testing.T) {
	for i, pw := range []string{"pw", "hunter2", ""} {
		s, f, _ := newSess(t)
		name := "d" + string(rune('A'+i))
		u := reg(t, s, f, name, pw, model.RoleRep)
		u.Status = model.StatusDisabled
		f.Users[u.ID] = u
		if _, err := s.Login(name, pw); !errors.Is(err, model.ErrDisabled) {
			t.Errorf("P-disabled-cannot-auth: Login for disabled %s must be ErrDisabled, got %v", name, err)
		}
	}
}

// P-session-active-user: a resume resolves to Active only while the user's status
// is Active; a flip to Disabled invalidates it.
func TestPropSessionActiveUser(t *testing.T) {
	s, f, _ := newSess(t)
	u := reg(t, s, f, "ivy", "pw", model.RoleRep)
	if _, err := s.Login("ivy", "pw"); err != nil {
		t.Fatalf("P-session-active-user: Login must succeed: %v", err)
	}
	// While Active, Current resolves.
	if _, err := s.Current(); err != nil {
		t.Fatalf("P-session-active-user: Current must resolve while Active: %v", err)
	}
	// Flip to Disabled: Current must no longer resolve.
	u.Status = model.StatusDisabled
	f.Users[u.ID] = u
	if _, err := s.Current(); err == nil {
		t.Errorf("P-session-active-user: Current must invalidate a now-Disabled user")
	}
}
