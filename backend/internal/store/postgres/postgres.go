// Package postgres implements the storage on top of PostgreSQL (pgx).
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/store"
)

// DB is the PostgreSQL backed store.
type DB struct {
	pool *pgxpool.Pool
}

// Open connects to PostgreSQL and verifies the connection.
func Open(ctx context.Context, url string) (*DB, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &DB{pool: pool}, nil
}

// New wraps an existing pool.
func New(pool *pgxpool.Pool) *DB { return &DB{pool: pool} }

// Close closes the pool.
func (db *DB) Close() { db.pool.Close() }

// Ping checks the connection.
func (db *DB) Ping(ctx context.Context) error { return db.pool.Ping(ctx) }

func mapErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return store.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return fmt.Errorf("%w: %s", store.ErrConflict, pgErr.ConstraintName)
		case "22P02": // invalid_text_representation, e.g. a bad uuid
			return store.ErrNotFound
		}
	}
	return err
}

func utc(t time.Time) time.Time { return t.UTC() }

func utcPtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}

// ---- users ----

// UpsertUser creates the user or updates its password hash.
func (db *DB) UpsertUser(ctx context.Context, login, passwordHash string) error {
	_, err := db.pool.Exec(ctx, `
		insert into users (login, password_hash) values ($1, $2)
		on conflict (login) do update set password_hash = excluded.password_hash`,
		login, passwordHash)
	return mapErr(err)
}

func (db *DB) UserByLogin(ctx context.Context, login string) (store.User, error) {
	var u store.User
	err := db.pool.QueryRow(ctx,
		`select id, login, password_hash from users where login = $1`, login,
	).Scan(&u.ID, &u.Login, &u.PasswordHash)
	return u, mapErr(err)
}

func (db *DB) UserByID(ctx context.Context, id uuid.UUID) (store.User, error) {
	var u store.User
	err := db.pool.QueryRow(ctx,
		`select id, login, password_hash from users where id = $1`, id,
	).Scan(&u.ID, &u.Login, &u.PasswordHash)
	return u, mapErr(err)
}

// ---- refresh tokens ----

//nolint:gosec // G101: SQL text, not a credential
const insertRefreshToken = `
	insert into refresh_tokens
		(id, user_id, token_hash, family_id, user_agent, ip, created_at, last_used_at, expires_at)
	values ($1, $2, $3, $4, $5, $6::inet, $7, $8, $9)`

func refreshArgs(t store.RefreshToken) []any {
	return []any{t.ID, t.UserID, t.TokenHash, t.FamilyID, t.UserAgent, t.IP, t.CreatedAt, t.LastUsedAt, t.ExpiresAt}
}

func (db *DB) CreateRefreshToken(ctx context.Context, t store.RefreshToken) error {
	_, err := db.pool.Exec(ctx, insertRefreshToken, refreshArgs(t)...)
	return mapErr(err)
}

func (db *DB) RefreshTokenByHash(ctx context.Context, hash string) (store.RefreshToken, error) {
	var t store.RefreshToken
	err := db.pool.QueryRow(ctx, `
		select id, user_id, token_hash, family_id, user_agent, host(ip), created_at,
		       last_used_at, expires_at, revoked_at, revoke_reason::text
		from refresh_tokens where token_hash = $1`, hash,
	).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.FamilyID, &t.UserAgent, &t.IP, &t.CreatedAt,
		&t.LastUsedAt, &t.ExpiresAt, &t.RevokedAt, &t.RevokeReason)
	if err != nil {
		return t, mapErr(err)
	}
	t.CreatedAt, t.LastUsedAt, t.ExpiresAt = utc(t.CreatedAt), utc(t.LastUsedAt), utc(t.ExpiresAt)
	t.RevokedAt = utcPtr(t.RevokedAt)
	return t, nil
}

func (db *DB) RotateRefreshToken(ctx context.Context, oldID uuid.UUID, at time.Time, next store.RefreshToken) error {
	return pgx.BeginFunc(ctx, db.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			update refresh_tokens
			set revoked_at = $2, revoke_reason = 'rotated', last_used_at = $2
			where id = $1 and revoked_at is null`, oldID, at)
		if err != nil {
			return mapErr(err)
		}
		if tag.RowsAffected() == 0 {
			return store.ErrConflict
		}
		_, err = tx.Exec(ctx, insertRefreshToken, refreshArgs(next)...)
		return mapErr(err)
	})
}

func (db *DB) RevokeFamily(ctx context.Context, userID, familyID uuid.UUID, reason string, at time.Time) (bool, error) {
	tag, err := db.pool.Exec(ctx, `
		update refresh_tokens set revoked_at = $3, revoke_reason = $4::revoke_reason
		where family_id = $1 and user_id = $2 and revoked_at is null`,
		familyID, userID, at, reason)
	if err != nil {
		return false, mapErr(err)
	}
	return tag.RowsAffected() > 0, nil
}

func (db *DB) FamilyActive(ctx context.Context, userID, familyID uuid.UUID, now time.Time) (bool, error) {
	var ok bool
	err := db.pool.QueryRow(ctx, `
		select exists (
			select 1 from refresh_tokens
			where family_id = $1 and user_id = $2 and revoked_at is null and expires_at > $3
		)`, familyID, userID, now).Scan(&ok)
	return ok, mapErr(err)
}

func (db *DB) ListSessions(ctx context.Context, userID uuid.UUID, now time.Time) ([]store.Session, error) {
	rows, err := db.pool.Query(ctx, `
		select t.family_id, t.user_agent, host(t.ip), f.created_at, f.last_used_at
		from refresh_tokens t
		join (
			select family_id, min(created_at) as created_at, max(last_used_at) as last_used_at
			from refresh_tokens where user_id = $1
			group by family_id
		) f on f.family_id = t.family_id
		where t.user_id = $1 and t.revoked_at is null and t.expires_at > $2
		order by f.last_used_at desc`, userID, now)
	if err != nil {
		return nil, mapErr(err)
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (store.Session, error) {
		var s store.Session
		err := row.Scan(&s.ID, &s.UserAgent, &s.IP, &s.CreatedAt, &s.LastUsedAt)
		s.CreatedAt, s.LastUsedAt = utc(s.CreatedAt), utc(s.LastUsedAt)
		return s, err
	})
	return out, mapErr(err)
}
