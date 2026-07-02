// Package session performs login (argon2id verify), reads/writes the on-disk
// session token, and resolves the current User (BUILD.md 4.2, 5.4, 9). It
// enforces disabled-cannot-auth and session-active-user. It imports crm.repo and
// the model kernel only (BUILD.md 4.5 allow: session->repo).
package session

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"

	"crm/internal/model"
	"crm/internal/repo"
)

// Sessions is the boundary API (BUILD.md 4.6). Login/Current/Logout.
type Sessions interface {
	Login(name, password string) (model.Session, error) // ErrBadCredentials, ErrDisabled, ErrLocked
	Current() (model.User, error)                       // ErrNoSession, ErrExpired
	Logout() error
}

// sessions is the concrete implementation. TokenPath is ~/.crm/session by
// default; Now and HMACKey are injected for testability.
type sessions struct {
	repo      repo.Repo
	tokenPath string
	now       func() time.Time
	hmacKey   []byte
	ttl       time.Duration
}

// Option configures a Sessions (clock and TTL injection for testability).
type Option func(*sessions)

// WithClock injects the time source (default time.Now). Tests use it to forge
// token expiry (C-SESS-07).
func WithClock(now func() time.Time) Option { return func(s *sessions) { s.now = now } }

// WithTTL overrides the session TTL (default 8h, BUILD.md 9 sessionTTL).
func WithTTL(d time.Duration) Option { return func(s *sessions) { s.ttl = d } }

// New constructs a Sessions bound to a repo and a token-file path. The token TTL
// is 8h (BUILD.md 9 sessionTTL) and Now defaults to time.Now.
func New(r repo.Repo, tokenPath string, hmacKey []byte, opts ...Option) Sessions {
	s := &sessions{
		repo:      r,
		tokenPath: tokenPath,
		now:       time.Now,
		hmacKey:   hmacKey,
		ttl:       8 * time.Hour,
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// sessRetryBackoff is the store-locked verify/load backoff (BUILD.md 9
// verifyRetryBackoff ~500ms). Kept modest so the hermetic suite stays fast.
const sessRetryBackoff = 100 * time.Millisecond

// sessMaxAttempts is the bounded retry count (<=3, BUILD.md 4.6, 5.4).
const sessMaxAttempts = 3

// Login verifies credentials (argon2id) with bounded retry on a locked store,
// then writes the HMAC-signed token (BUILD.md 5.4, C-SESS-01..04). Bad
// credentials and a disabled user are never retried.
func (s *sessions) Login(name, password string) (model.Session, error) {
	var (
		u   model.User
		err error
	)
	for attempt := 0; attempt < sessMaxAttempts; attempt++ {
		u, err = s.verifyCredentials(name, password)
		if err == nil {
			break
		}
		if errors.Is(err, model.ErrLocked) {
			if attempt < sessMaxAttempts-1 {
				time.Sleep(sessRetryBackoff)
			}
			continue
		}
		return model.Session{}, err // ErrBadCredentials, ErrDisabled, ...: not retried
	}
	if err != nil {
		return model.Session{}, err // ErrLocked after the bounded retry
	}
	expiresAt := s.now().Add(s.ttl)
	if werr := s.writeSessionFile(u.ID, expiresAt); werr != nil {
		return model.Session{}, werr
	}
	return model.Session{UserID: u.ID, ExpiresAt: expiresAt}, nil
}

// Current resolves the acting User from the token, re-validating that the user
// is still Active (session-active-user) (BUILD.md 5.4, C-SESS-05..08).
func (s *sessions) Current() (model.User, error) {
	tok, err := s.readSessionFile()
	if err != nil {
		return model.User{}, err // ErrNoSession, ErrExpired, ErrUnreadable
	}
	u, err := s.loadUser(tok.UserID)
	if err != nil {
		return model.User{}, err // ErrNotFound, ErrLocked, ...
	}
	if u.Status != model.StatusActive {
		return model.User{}, model.ErrDisabled // session-active-user
	}
	return u, nil
}

// Logout best-effort clears the token (BUILD.md 5.4, C-SESS-09).
func (s *sessions) Logout() error {
	return s.clearSessionFile()
}

// Register is the create path owned by crm.session (BUILD.md 5.3 SCOPE note, 9):
// it hashes the password with argon2id and persists only the encoded hash
// (password-hashed), rejecting a duplicate username (username-unique).
func (s *sessions) Register(tx repo.Tx, username, password string, role model.UserRole) (model.User, error) {
	if _, err := s.repo.GetUserByName(tx, username); err == nil {
		return model.User{}, model.ErrConstraint // username-unique
	} else if !errors.Is(err, model.ErrNotFound) {
		return model.User{}, err
	}
	id, err := newID()
	if err != nil {
		return model.User{}, err
	}
	u := model.User{
		ID:           id,
		Username:     username,
		PasswordHash: hashPassword(password),
		Role:         role,
		Status:       model.StatusActive,
		CreatedAt:    s.now(),
	}
	if err := s.repo.SaveUser(tx, u); err != nil {
		return model.User{}, err
	}
	return u, nil
}

// ChangePassword is the update path owned by crm.session; it re-hashes with
// argon2id and never writes plaintext (password-hashed).
func (s *sessions) ChangePassword(tx repo.Tx, userID, newPassword string) error {
	u, err := s.repo.GetUser(tx, userID)
	if err != nil {
		return err
	}
	u.PasswordHash = hashPassword(newPassword)
	return s.repo.SaveUser(tx, u)
}

// Impl exposes the concrete type for the create/update paths that are not part
// of the Sessions boundary interface, so contract/property tests can reach
// Register/ChangePassword. The implementer keeps this seam or inlines it.
func Impl(s Sessions) *sessions { //nolint:revive // test seam for scaffolding
	c, _ := s.(*sessions)
	return c
}

// --- Actors (BUILD.md 5.4 named-unit contract table). ---

// verifyCredentials loads the user and verifies the argon2id hash; it never
// returns a User on bad credentials, and it enforces disabled-cannot-auth before
// the password check (BUILD.md 5.4).
func (s *sessions) verifyCredentials(username, password string) (model.User, error) {
	tx, err := s.repo.Open("")
	if err != nil {
		return model.User{}, err
	}
	u, err := s.repo.GetUserByName(tx, username)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return model.User{}, model.ErrBadCredentials // do not leak existence
		}
		return model.User{}, err // ErrLocked, ErrUnavailable, ...
	}
	if u.Status == model.StatusDisabled {
		return model.User{}, model.ErrDisabled // disabled-cannot-auth
	}
	if !verifyPassword(password, u.PasswordHash) {
		return model.User{}, model.ErrBadCredentials
	}
	return u, nil
}

// writeSessionFile writes the HMAC-signed token with 0600 perms (BUILD.md 9).
func (s *sessions) writeSessionFile(userID string, expiresAt time.Time) error {
	body := userID + "|" + strconv.FormatInt(expiresAt.Unix(), 10)
	token := body + "|" + s.sign(body)
	return os.WriteFile(s.tokenPath, []byte(token), 0o600)
}

// readSessionFile parses and verifies the token, or returns a typed error
// (ErrNoSession, ErrExpired, ErrUnreadable) (BUILD.md 5.4).
func (s *sessions) readSessionFile() (model.Session, error) {
	raw, err := os.ReadFile(s.tokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return model.Session{}, model.ErrNoSession
		}
		return model.Session{}, model.ErrUnreadable
	}
	parts := strings.SplitN(strings.TrimSpace(string(raw)), "|", 3)
	if len(parts) != 3 {
		return model.Session{}, model.ErrUnreadable
	}
	userID, expStr, sig := parts[0], parts[1], parts[2]
	if !hmac.Equal([]byte(sig), []byte(s.sign(userID+"|"+expStr))) {
		return model.Session{}, model.ErrUnreadable // bad signature
	}
	secs, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		return model.Session{}, model.ErrUnreadable
	}
	expiresAt := time.Unix(secs, 0)
	if !expiresAt.After(s.now()) {
		return model.Session{}, model.ErrExpired
	}
	return model.Session{UserID: userID, ExpiresAt: expiresAt}, nil
}

// loadUser reads the user with its current status (BUILD.md 5.4).
func (s *sessions) loadUser(userID string) (model.User, error) {
	tx, err := s.repo.Open("")
	if err != nil {
		return model.User{}, err
	}
	return s.repo.GetUser(tx, userID)
}

// clearSessionFile removes the token, best-effort (BUILD.md 5.4): a missing file
// is not an error.
func (s *sessions) clearSessionFile() error {
	if err := os.Remove(s.tokenPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// sign returns the hex HMAC-SHA256 of body under the machine-local key.
func (s *sessions) sign(body string) string {
	mac := hmac.New(sha256.New, s.hmacKey)
	mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

// --- argon2id password hashing (BUILD.md 9, password-hashed). ---

const (
	argonTime    = 1
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
	argonSaltLen = 16
)

// hashPassword returns the argon2id PHC-encoded hash string ("$argon2id$..."),
// with a fresh random salt; only this encoded string is ever persisted.
func hashPassword(password string) string {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		// crypto/rand failure is unrecoverable; fail closed with an empty salt so
		// verification can never succeed against this hash.
		salt = make([]byte, argonSaltLen)
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		b64.EncodeToString(salt), b64.EncodeToString(key))
}

// verifyPassword recomputes the argon2id key over the encoded salt/params and
// compares in constant time. A malformed encoding fails closed.
func verifyPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false
	}
	var memory uint32
	var t uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &t, &threads); err != nil {
		return false
	}
	b64 := base64.RawStdEncoding
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return false
	}
	want, err := b64.DecodeString(parts[5])
	if err != nil {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, t, memory, threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

// newID returns a random 128-bit hex identifier for a freshly registered user.
func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
