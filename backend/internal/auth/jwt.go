package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AccessTTL is the lifetime of an access token.
const AccessTTL = 15 * time.Minute

// Claims are the access token claims.
type Claims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

// Identity is the authenticated principal extracted from an access token.
type Identity struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
}

// SignAccessToken creates an HS256 JWT with sub, sid, iat and exp.
func SignAccessToken(secret []byte, userID, sessionID uuid.UUID, now time.Time) (string, error) {
	claims := Claims{
		SessionID: sessionID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

// ParseAccessToken validates the token signature and expiry at time now.
func ParseAccessToken(secret []byte, token string, now time.Time) (Identity, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(token, &claims,
		func(*jwt.Token) (any, error) { return secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(func() time.Time { return now }),
	)
	if err != nil {
		return Identity{}, fmt.Errorf("%w: %w", ErrUnauthorized, err)
	}
	uid, err := uuid.Parse(claims.Subject)
	if err != nil {
		return Identity{}, fmt.Errorf("%w: bad sub", ErrUnauthorized)
	}
	sid, err := uuid.Parse(claims.SessionID)
	if err != nil {
		return Identity{}, fmt.Errorf("%w: bad sid", ErrUnauthorized)
	}
	return Identity{UserID: uid, SessionID: sid}, nil
}

// ErrUnauthorized is returned for any authentication failure.
var ErrUnauthorized = errors.New("unauthorized")
