package session_test

// Session boundary contract tests (BUILD.md 7.2 C-SESS-01..10). Per the BUILD.md
// mock policy (unit tests may mock the single Repo collaborator) these run
// against the in-memory fake Repo plus a REAL temporary token file, so the
// default suite stays hermetic. (7.2 suggests a "real (temp) repo"; the mock
// policy in section 10 permits mocking the Repo for unit tests. Recorded as a
// spec tension.) Each test seeds via Register/Login (which Fatalf on the stub),
// so every C-SESS test is RED against the scaffolding.

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"crm/internal/model"
	"crm/internal/session"
	"crm/internal/testsupport"
)

var testKey = []byte("crm-test-hmac-key-0123456789")

func newSess(t *testing.T) (session.Sessions, *testsupport.FakeRepo, string) {
	t.Helper()
	f := testsupport.NewFakeRepo()
	path := filepath.Join(t.TempDir(), "session")
	return session.New(f, path, testKey), f, path
}

func reg(t *testing.T, s session.Sessions, f *testsupport.FakeRepo, name, pw string, role model.UserRole) model.User {
	t.Helper()
	tx, _ := f.Open("mem")
	u, err := session.Impl(s).Register(tx, name, pw, role)
	if err != nil {
		t.Fatalf("Register(%s) must succeed: %v", name, err)
	}
	return u
}

// C-SESS-01: Login with valid credentials -> Session with future expiry; token written.
func TestCSess01LoginValid(t *testing.T) {
	s, f, path := newSess(t)
	reg(t, s, f, "alice", "pw", model.RoleRep)
	sess, err := s.Login("alice", "pw")
	if err != nil {
		t.Fatalf("C-SESS-01: Login must succeed: %v", err)
	}
	if !sess.ExpiresAt.After(time.Now()) {
		t.Errorf("C-SESS-01: session expiry must be in the future, got %v", sess.ExpiresAt)
	}
	if fi, err := os.Stat(path); err != nil || fi.Size() == 0 {
		t.Errorf("C-SESS-01: token file must be written")
	}
}

// C-SESS-02: Login with a wrong password -> ErrBadCredentials (not retried).
func TestCSess02WrongPassword(t *testing.T) {
	s, f, _ := newSess(t)
	reg(t, s, f, "alice", "pw", model.RoleRep)
	_, err := s.Login("alice", "wrong")
	if !errors.Is(err, model.ErrBadCredentials) {
		t.Errorf("C-SESS-02: want ErrBadCredentials, got %v", err)
	}
}

// C-SESS-03: Login as a Disabled user -> ErrDisabled (disabled-cannot-auth).
func TestCSess03Disabled(t *testing.T) {
	s, f, _ := newSess(t)
	u := reg(t, s, f, "bob", "pw", model.RoleRep)
	u.Status = model.StatusDisabled
	f.Users[u.ID] = u
	_, err := s.Login("bob", "pw")
	if !errors.Is(err, model.ErrDisabled) {
		t.Errorf("C-SESS-03: want ErrDisabled, got %v", err)
	}
}

// C-SESS-04: Login while the store is locked -> ErrLocked (bounded retry <=3).
func TestCSess04StoreLocked(t *testing.T) {
	s, f, _ := newSess(t)
	reg(t, s, f, "carol", "pw", model.RoleRep)
	f.GetUserByNameErr = model.ErrLocked // verify hits a locked store
	_, err := s.Login("carol", "pw")
	if !errors.Is(err, model.ErrLocked) {
		t.Errorf("C-SESS-04: want ErrLocked after bounded retry, got %v", err)
	}
}

// C-SESS-05: Current with a valid token -> returns the User.
func TestCSess05CurrentValid(t *testing.T) {
	s, f, _ := newSess(t)
	reg(t, s, f, "dan", "pw", model.RoleRep)
	if _, err := s.Login("dan", "pw"); err != nil {
		t.Fatalf("C-SESS-05: Login must succeed: %v", err)
	}
	u, err := s.Current()
	if err != nil {
		t.Fatalf("C-SESS-05: Current must resolve: %v", err)
	}
	if u.Username != "dan" {
		t.Errorf("C-SESS-05: Current returned %q, want dan", u.Username)
	}
}

// C-SESS-06: Current with no token file -> ErrNoSession.
func TestCSess06NoToken(t *testing.T) {
	s, _, _ := newSess(t)
	_, err := s.Current()
	if !errors.Is(err, model.ErrNoSession) {
		t.Errorf("C-SESS-06: want ErrNoSession, got %v", err)
	}
}

// C-SESS-07: Current with an expired token -> ErrExpired.
func TestCSess07Expired(t *testing.T) {
	f := testsupport.NewFakeRepo()
	path := filepath.Join(t.TempDir(), "session")
	clock := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s := session.New(f, path, testKey, session.WithClock(func() time.Time { return clock }))
	reg(t, s, f, "erin", "pw", model.RoleRep)
	if _, err := s.Login("erin", "pw"); err != nil {
		t.Fatalf("C-SESS-07: Login must succeed: %v", err)
	}
	clock = clock.Add(9 * time.Hour) // advance past the 8h TTL
	_, err := s.Current()
	if !errors.Is(err, model.ErrExpired) {
		t.Errorf("C-SESS-07: want ErrExpired, got %v", err)
	}
}

// C-SESS-08: Current when the user is now Disabled -> invalidated / not Active
// (session-active-user).
func TestCSess08UserDisabledOnResume(t *testing.T) {
	s, f, _ := newSess(t)
	u := reg(t, s, f, "fay", "pw", model.RoleRep)
	if _, err := s.Login("fay", "pw"); err != nil {
		t.Fatalf("C-SESS-08: Login must succeed: %v", err)
	}
	u.Status = model.StatusDisabled
	f.Users[u.ID] = u
	if _, err := s.Current(); err == nil {
		t.Errorf("C-SESS-08: Current must not resolve a now-Disabled user")
	}
}

// C-SESS-09: Logout then Current -> token cleared; Current -> ErrNoSession.
func TestCSess09LogoutThenCurrent(t *testing.T) {
	s, f, _ := newSess(t)
	reg(t, s, f, "gil", "pw", model.RoleRep)
	if _, err := s.Login("gil", "pw"); err != nil {
		t.Fatalf("C-SESS-09: Login must succeed: %v", err)
	}
	if err := s.Logout(); err != nil {
		t.Fatalf("C-SESS-09: Logout must succeed: %v", err)
	}
	if _, err := s.Current(); !errors.Is(err, model.ErrNoSession) {
		t.Errorf("C-SESS-09: after logout Current must be ErrNoSession, got %v", err)
	}
}

// C-SESS-10: after register, only an argon2id encoded hash is stored; no plaintext
// (password-hashed).
func TestCSess10OnlyHashStored(t *testing.T) {
	s, f, _ := newSess(t)
	reg(t, s, f, "hal", "s3cr3t", model.RoleRep)
	tx, _ := f.Open("mem")
	u, err := f.GetUserByName(tx, "hal")
	if err != nil {
		t.Fatalf("C-SESS-10: registered user must be stored: %v", err)
	}
	if !strings.HasPrefix(u.PasswordHash, "$argon2id$") {
		t.Errorf("C-SESS-10: stored credential must be an argon2id hash, got %q", u.PasswordHash)
	}
	if strings.Contains(u.PasswordHash, "s3cr3t") {
		t.Errorf("C-SESS-10: plaintext password must never be stored")
	}
}
