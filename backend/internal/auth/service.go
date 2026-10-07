// Package auth implements password login, access tokens (JWT) and
// rotating refresh tokens grouped into sessions (families).
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/store"
)

const (
	// RefreshTTL is the lifetime of a refresh token.
	RefreshTTL = 30 * 24 * time.Hour
	// RotationGrace is how long a token replaced by rotation is rejected
	// without revoking its family (concurrent refresh from two tabs).
	RotationGrace = 10 * time.Second
)

// ErrRefreshRace means the refresh token was rotated moments ago by a
// concurrent request. The caller must reject the request but should not
// clear the client cookies, which were just replaced by the winner.
var ErrRefreshRace = fmt.Errorf("%w: refresh token rotated concurrently", ErrUnauthorized)

// Store is the persistence the auth service needs.
type Store interface {
	UserByLogin(ctx context.Context, login string) (store.User, error)
	CreateRefreshToken(ctx context.Context, t store.RefreshToken) error
	RefreshTokenByHash(ctx context.Context, hash string) (store.RefreshToken, error)
	// RotateRefreshToken atomically marks oldID as revoked with reason
	// "rotated" (only if it is not revoked yet) and inserts next.
	// It returns store.ErrConflict if oldID was already revoked.
	RotateRefreshToken(ctx context.Context, oldID uuid.UUID, at time.Time, next store.RefreshToken) error
	// RevokeFamily revokes all not yet revoked tokens of the family owned
	// by userID and reports whether any token was revoked.
	RevokeFamily(ctx context.Context, userID, familyID uuid.UUID, reason string, at time.Time) (bool, error)
	FamilyActive(ctx context.Context, userID, familyID uuid.UUID, now time.Time) (bool, error)
	ListSessions(ctx context.Context, userID uuid.UUID, now time.Time) ([]store.Session, error)
}

// Tokens is a freshly issued pair of tokens.
type Tokens struct {
	Access    string
	Refresh   string
	SessionID uuid.UUID
}

// Service implements the authentication flows.
type Service struct {
	store  Store
	secret []byte
	now    func() time.Time

	dummyOnce sync.Once
	dummyHash string
}

// NewService creates the auth service. now may be nil (time.Now is used).
func NewService(s Store, jwtSecret []byte, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: s, secret: jwtSecret, now: now}
}

// Login checks credentials and starts a new session.
func (s *Service) Login(ctx context.Context, login, password, userAgent, ip string) (Tokens, error) {
	user, err := s.store.UserByLogin(ctx, login)
	if errors.Is(err, store.ErrNotFound) {
		// Spend the same time as for an existing user.
		_, _ = VerifyPassword(s.dummy(), password)
		return Tokens{}, ErrUnauthorized
	}
	if err != nil {
		return Tokens{}, err
	}
	ok, err := VerifyPassword(user.PasswordHash, password)
	if err != nil {
		return Tokens{}, err
	}
	if !ok {
		return Tokens{}, ErrUnauthorized
	}

	now := s.now().UTC()
	raw, rt, err := newRefreshToken(user.ID, uuid.New(), userAgent, ip, now)
	if err != nil {
		return Tokens{}, err
	}
	if err := s.store.CreateRefreshToken(ctx, rt); err != nil {
		return Tokens{}, err
	}
	return s.issue(user.ID, rt.FamilyID, raw, now)
}

// Refresh rotates the refresh token and issues a new pair.
func (s *Service) Refresh(ctx context.Context, rawRefresh, userAgent, ip string) (Tokens, error) {
	if rawRefresh == "" {
		return Tokens{}, ErrUnauthorized
	}
	old, err := s.store.RefreshTokenByHash(ctx, HashRefreshToken(rawRefresh))
	if errors.Is(err, store.ErrNotFound) {
		return Tokens{}, ErrUnauthorized
	}
	if err != nil {
		return Tokens{}, err
	}

	now := s.now().UTC()
	if old.RevokedAt != nil {
		if old.RevokeReason != nil && *old.RevokeReason == store.RevokeRotated &&
			now.Sub(*old.RevokedAt) < RotationGrace {
			return Tokens{}, ErrRefreshRace
		}
		// A revoked token was presented again: assume it was stolen.
		if _, err := s.store.RevokeFamily(ctx, old.UserID, old.FamilyID, store.RevokeReuse, now); err != nil {
			return Tokens{}, err
		}
		return Tokens{}, fmt.Errorf("%w: refresh token reuse", ErrUnauthorized)
	}
	if !now.Before(old.ExpiresAt) {
		return Tokens{}, fmt.Errorf("%w: refresh token expired", ErrUnauthorized)
	}
	if old.UserAgent != userAgent {
		return Tokens{}, fmt.Errorf("%w: user agent mismatch", ErrUnauthorized)
	}

	raw, next, err := newRefreshToken(old.UserID, old.FamilyID, old.UserAgent, ip, now)
	if err != nil {
		return Tokens{}, err
	}
	err = s.store.RotateRefreshToken(ctx, old.ID, now, next)
	if errors.Is(err, store.ErrConflict) {
		// Lost the race against a concurrent rotation of the same token.
		return Tokens{}, ErrRefreshRace
	}
	if err != nil {
		return Tokens{}, err
	}
	return s.issue(old.UserID, old.FamilyID, raw, now)
}

// Authenticate validates an access token and checks that its session is
// still active.
func (s *Service) Authenticate(ctx context.Context, accessToken string) (Identity, error) {
	if accessToken == "" {
		return Identity{}, ErrUnauthorized
	}
	now := s.now()
	id, err := ParseAccessToken(s.secret, accessToken, now)
	if err != nil {
		return Identity{}, err
	}
	ok, err := s.store.FamilyActive(ctx, id.UserID, id.SessionID, now.UTC())
	if err != nil {
		return Identity{}, err
	}
	if !ok {
		return Identity{}, fmt.Errorf("%w: session revoked", ErrUnauthorized)
	}
	return id, nil
}

// Logout revokes the session of the given identity.
func (s *Service) Logout(ctx context.Context, id Identity) error {
	_, err := s.store.RevokeFamily(ctx, id.UserID, id.SessionID, store.RevokeLogout, s.now().UTC())
	return err
}

// Sessions lists active sessions of the user.
func (s *Service) Sessions(ctx context.Context, userID uuid.UUID) ([]store.Session, error) {
	return s.store.ListSessions(ctx, userID, s.now().UTC())
}

// RevokeSession ends a session of the user. It returns store.ErrNotFound
// if there is no such active session.
func (s *Service) RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	ok, err := s.store.RevokeFamily(ctx, userID, sessionID, store.RevokeManual, s.now().UTC())
	if err != nil {
		return err
	}
	if !ok {
		return store.ErrNotFound
	}
	return nil
}

func (s *Service) issue(userID, familyID uuid.UUID, rawRefresh string, now time.Time) (Tokens, error) {
	access, err := SignAccessToken(s.secret, userID, familyID, now)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{Access: access, Refresh: rawRefresh, SessionID: familyID}, nil
}

func (s *Service) dummy() string {
	s.dummyOnce.Do(func() {
		h, err := HashPassword("dummy password for timing")
		if err == nil {
			s.dummyHash = h
		}
	})
	return s.dummyHash
}

// HashRefreshToken returns the value stored in refresh_tokens.token_hash.
func HashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func newRefreshToken(userID, familyID uuid.UUID, userAgent, ip string, now time.Time) (string, store.RefreshToken, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", store.RefreshToken{}, err
	}
	raw := base64.RawURLEncoding.EncodeToString(b)
	return raw, store.RefreshToken{
		ID:         uuid.New(),
		UserID:     userID,
		TokenHash:  HashRefreshToken(raw),
		FamilyID:   familyID,
		UserAgent:  userAgent,
		IP:         ip,
		CreatedAt:  now,
		LastUsedAt: now,
		ExpiresAt:  now.Add(RefreshTTL),
	}, nil
}
