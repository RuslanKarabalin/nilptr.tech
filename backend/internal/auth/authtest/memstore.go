// Package authtest provides an in-memory auth.Store for tests.
package authtest

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/store"
)

// MemStore is a goroutine safe in-memory implementation of auth.Store.
type MemStore struct {
	mu     sync.Mutex
	Users  map[string]store.User
	Tokens map[uuid.UUID]*store.RefreshToken
}

// New returns an empty store.
func New() *MemStore {
	return &MemStore{Users: map[string]store.User{}, Tokens: map[uuid.UUID]*store.RefreshToken{}}
}

// AddUser adds a user with the given password hash and returns it.
func (m *MemStore) AddUser(login, hash string) store.User {
	m.mu.Lock()
	defer m.mu.Unlock()
	u := store.User{ID: uuid.New(), Login: login, PasswordHash: hash}
	m.Users[login] = u
	return u
}

// Token returns a copy of the token with the given hash.
func (m *MemStore) Token(hash string) (store.RefreshToken, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.Tokens {
		if t.TokenHash == hash {
			return *t, true
		}
	}
	return store.RefreshToken{}, false
}

func (m *MemStore) UserByLogin(_ context.Context, login string) (store.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.Users[login]
	if !ok {
		return u, store.ErrNotFound
	}
	return u, nil
}

func (m *MemStore) UserByID(_ context.Context, id uuid.UUID) (store.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.Users {
		if u.ID == id {
			return u, nil
		}
	}
	return store.User{}, store.ErrNotFound
}

func (m *MemStore) CreateRefreshToken(_ context.Context, t store.RefreshToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Tokens[t.ID] = &t
	return nil
}

func (m *MemStore) RefreshTokenByHash(_ context.Context, hash string) (store.RefreshToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.Tokens {
		if t.TokenHash == hash {
			return *t, nil
		}
	}
	return store.RefreshToken{}, store.ErrNotFound
}

func (m *MemStore) RotateRefreshToken(_ context.Context, oldID uuid.UUID, at time.Time, next store.RefreshToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	old, ok := m.Tokens[oldID]
	if !ok || old.RevokedAt != nil {
		return store.ErrConflict
	}
	reason := store.RevokeRotated
	old.RevokedAt, old.RevokeReason, old.LastUsedAt = &at, &reason, at
	m.Tokens[next.ID] = &next
	return nil
}

func (m *MemStore) RevokeFamily(_ context.Context, userID, familyID uuid.UUID, reason string, at time.Time) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	revoked := false
	for _, t := range m.Tokens {
		if t.FamilyID == familyID && t.UserID == userID && t.RevokedAt == nil {
			r := reason
			t.RevokedAt, t.RevokeReason = &at, &r
			revoked = true
		}
	}
	return revoked, nil
}

func (m *MemStore) FamilyActive(_ context.Context, userID, familyID uuid.UUID, now time.Time) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.Tokens {
		if t.FamilyID == familyID && t.UserID == userID && t.RevokedAt == nil && t.ExpiresAt.After(now) {
			return true, nil
		}
	}
	return false, nil
}

func (m *MemStore) ListSessions(_ context.Context, userID uuid.UUID, now time.Time) ([]store.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	first := map[uuid.UUID]time.Time{}
	for _, t := range m.Tokens {
		if t.UserID != userID {
			continue
		}
		if c, ok := first[t.FamilyID]; !ok || t.CreatedAt.Before(c) {
			first[t.FamilyID] = t.CreatedAt
		}
	}
	var out []store.Session
	for _, t := range m.Tokens {
		if t.UserID == userID && t.RevokedAt == nil && t.ExpiresAt.After(now) {
			out = append(out, store.Session{
				ID: t.FamilyID, UserAgent: t.UserAgent, IP: t.IP,
				CreatedAt: first[t.FamilyID], LastUsedAt: t.LastUsedAt,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastUsedAt.After(out[j].LastUsedAt) })
	return out, nil
}
