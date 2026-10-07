package auth_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/auth"
	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/auth/authtest"
	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/store"
)

var secret = []byte("0123456789abcdef0123456789abcdef")

const ua = "Mozilla/5.0 test"

type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }
func newClock() *clock                   { return &clock{t: time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)} }
func ctx() context.Context               { return context.Background() }
func mustHash(t *testing.T, pw string) string {
	t.Helper()
	h, err := auth.HashPassword(pw)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func setup(t *testing.T) (*auth.Service, *authtest.MemStore, *clock, store.User) {
	t.Helper()
	ms := authtest.New()
	u := ms.AddUser("admin", mustHash(t, "secret"))
	c := newClock()
	return auth.NewService(ms, secret, c.now), ms, c, u
}

func TestPassword(t *testing.T) {
	h := mustHash(t, "hunter2")
	if !strings.HasPrefix(h, "$argon2id$v=19$") {
		t.Fatalf("unexpected format %q", h)
	}
	if ok, err := auth.VerifyPassword(h, "hunter2"); err != nil || !ok {
		t.Fatalf("verify good: %v %v", ok, err)
	}
	if ok, _ := auth.VerifyPassword(h, "hunter3"); ok {
		t.Fatal("verify bad password succeeded")
	}
	if _, err := auth.VerifyPassword("garbage", "x"); err == nil {
		t.Fatal("expected error for bad hash")
	}
}

func TestJWT(t *testing.T) {
	now := time.Now()
	uid, sid := uuid.New(), uuid.New()
	tok, err := auth.SignAccessToken(secret, uid, sid, now)
	if err != nil {
		t.Fatal(err)
	}
	id, err := auth.ParseAccessToken(secret, tok, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if id.UserID != uid || id.SessionID != sid {
		t.Fatalf("got %+v", id)
	}

	if _, err := auth.ParseAccessToken(secret, tok, now.Add(auth.AccessTTL+time.Second)); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("expired token: %v", err)
	}
	if _, err := auth.ParseAccessToken([]byte("another secret another secret!!!"), tok, now); err == nil {
		t.Fatal("wrong secret accepted")
	}

	// alg=none and other algorithms must be rejected.
	none := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub": uid.String(), "sid": sid.String(), "exp": now.Add(time.Hour).Unix(),
	})
	s, _ := none.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, err := auth.ParseAccessToken(secret, s, now); err == nil {
		t.Fatal("alg none accepted")
	}
	hs512 := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"sub": uid.String(), "sid": sid.String(), "exp": now.Add(time.Hour).Unix(),
	})
	s, _ = hs512.SignedString(secret)
	if _, err := auth.ParseAccessToken(secret, s, now); err == nil {
		t.Fatal("HS512 accepted")
	}
}

func TestLogin(t *testing.T) {
	svc, ms, _, u := setup(t)
	if _, err := svc.Login(ctx(), "admin", "wrong", ua, "1.2.3.4"); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("bad password: %v", err)
	}
	if _, err := svc.Login(ctx(), "nobody", "secret", ua, "1.2.3.4"); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("unknown user: %v", err)
	}
	tok, err := svc.Login(ctx(), "admin", "secret", ua, "1.2.3.4")
	if err != nil {
		t.Fatal(err)
	}
	rt, ok := ms.Token(auth.HashRefreshToken(tok.Refresh))
	if !ok {
		t.Fatal("refresh token not stored by hash")
	}
	if rt.FamilyID != tok.SessionID || rt.UserID != u.ID || rt.UserAgent != ua || rt.IP != "1.2.3.4" {
		t.Fatalf("stored token %+v", rt)
	}
	id, err := svc.Authenticate(ctx(), tok.Access)
	if err != nil || id.UserID != u.ID || id.SessionID != tok.SessionID {
		t.Fatalf("authenticate: %+v %v", id, err)
	}
}

func TestRefreshRotation(t *testing.T) {
	svc, ms, c, _ := setup(t)
	first, err := svc.Login(ctx(), "admin", "secret", ua, "1.2.3.4")
	if err != nil {
		t.Fatal(err)
	}
	c.advance(time.Minute)
	second, err := svc.Refresh(ctx(), first.Refresh, ua, "5.6.7.8")
	if err != nil {
		t.Fatal(err)
	}
	if second.Refresh == first.Refresh || second.SessionID != first.SessionID {
		t.Fatal("rotation must issue a new token in the same family")
	}
	old, _ := ms.Token(auth.HashRefreshToken(first.Refresh))
	if old.RevokedAt == nil || *old.RevokeReason != store.RevokeRotated {
		t.Fatalf("old token not rotated: %+v", old)
	}
	next, _ := ms.Token(auth.HashRefreshToken(second.Refresh))
	if next.IP != "5.6.7.8" || next.UserAgent != ua || !next.ExpiresAt.Equal(c.t.Add(auth.RefreshTTL)) {
		t.Fatalf("new token %+v", next)
	}
	// The access token of the family stays valid.
	if _, err := svc.Authenticate(ctx(), first.Access); err != nil {
		t.Fatalf("access token of active family rejected: %v", err)
	}
}

func TestRefreshGraceDoesNotRevokeFamily(t *testing.T) {
	svc, _, c, _ := setup(t)
	first, _ := svc.Login(ctx(), "admin", "secret", ua, "1.2.3.4")
	second, err := svc.Refresh(ctx(), first.Refresh, ua, "1.2.3.4")
	if err != nil {
		t.Fatal(err)
	}
	c.advance(auth.RotationGrace - time.Second)
	if _, err := svc.Refresh(ctx(), first.Refresh, ua, "1.2.3.4"); !errors.Is(err, auth.ErrRefreshRace) {
		t.Fatalf("want ErrRefreshRace, got %v", err)
	}
	if _, err := svc.Refresh(ctx(), second.Refresh, ua, "1.2.3.4"); err != nil {
		t.Fatalf("family must stay active after a race: %v", err)
	}
}

func TestRefreshReuseRevokesFamily(t *testing.T) {
	svc, _, c, _ := setup(t)
	first, _ := svc.Login(ctx(), "admin", "secret", ua, "1.2.3.4")
	second, err := svc.Refresh(ctx(), first.Refresh, ua, "1.2.3.4")
	if err != nil {
		t.Fatal(err)
	}
	c.advance(auth.RotationGrace)
	_, err = svc.Refresh(ctx(), first.Refresh, ua, "1.2.3.4")
	if err == nil || errors.Is(err, auth.ErrRefreshRace) || !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("reuse: %v", err)
	}
	if _, err := svc.Refresh(ctx(), second.Refresh, ua, "1.2.3.4"); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("family should be revoked: %v", err)
	}
	if _, err := svc.Authenticate(ctx(), second.Access); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("access token of revoked family accepted: %v", err)
	}
}

func TestRefreshLogoutTokenReuseRevokes(t *testing.T) {
	svc, ms, _, _ := setup(t)
	tok, _ := svc.Login(ctx(), "admin", "secret", ua, "1.2.3.4")
	id, _ := svc.Authenticate(ctx(), tok.Access)
	if err := svc.Logout(ctx(), id); err != nil {
		t.Fatal(err)
	}
	// Revoked by logout (not rotation) even within the grace window.
	if _, err := svc.Refresh(ctx(), tok.Refresh, ua, "1.2.3.4"); err == nil || errors.Is(err, auth.ErrRefreshRace) {
		t.Fatalf("refresh after logout: %v", err)
	}
	rt, _ := ms.Token(auth.HashRefreshToken(tok.Refresh))
	if *rt.RevokeReason != store.RevokeLogout {
		t.Fatalf("reason overwritten: %s", *rt.RevokeReason)
	}
	if _, err := svc.Authenticate(ctx(), tok.Access); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatal("access token valid after logout")
	}
}

func TestRefreshRejects(t *testing.T) {
	svc, _, c, _ := setup(t)
	tok, _ := svc.Login(ctx(), "admin", "secret", ua, "1.2.3.4")

	if _, err := svc.Refresh(ctx(), "", ua, "1.2.3.4"); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("empty: %v", err)
	}
	if _, err := svc.Refresh(ctx(), "unknown", ua, "1.2.3.4"); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("unknown: %v", err)
	}
	if _, err := svc.Refresh(ctx(), tok.Refresh, "curl/8", "1.2.3.4"); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("ua mismatch: %v", err)
	}
	// UA mismatch does not consume the token.
	c.advance(auth.RefreshTTL)
	if _, err := svc.Refresh(ctx(), tok.Refresh, ua, "1.2.3.4"); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("expired: %v", err)
	}
}

func TestSessions(t *testing.T) {
	svc, _, c, u := setup(t)
	a, _ := svc.Login(ctx(), "admin", "secret", ua, "1.1.1.1")
	c.advance(time.Second)
	b, _ := svc.Login(ctx(), "admin", "secret", "other", "2.2.2.2")

	list, err := svc.Sessions(ctx(), u.ID)
	if err != nil || len(list) != 2 {
		t.Fatalf("sessions: %v %v", list, err)
	}
	if err := svc.RevokeSession(ctx(), u.ID, a.SessionID); err != nil {
		t.Fatal(err)
	}
	if err := svc.RevokeSession(ctx(), u.ID, a.SessionID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("second revoke: %v", err)
	}
	if _, err := svc.Authenticate(ctx(), a.Access); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatal("revoked session still authenticates")
	}
	if _, err := svc.Authenticate(ctx(), b.Access); err != nil {
		t.Fatalf("other session affected: %v", err)
	}
	list, _ = svc.Sessions(ctx(), u.ID)
	if len(list) != 1 || list[0].ID != b.SessionID {
		t.Fatalf("after revoke: %+v", list)
	}
}

func TestRefreshConcurrent(t *testing.T) {
	svc, _, _, _ := setup(t)
	tok, _ := svc.Login(ctx(), "admin", "secret", ua, "1.2.3.4")
	const n = 8
	errs := make(chan error, n)
	for range n {
		go func() {
			_, err := svc.Refresh(ctx(), tok.Refresh, ua, "1.2.3.4")
			errs <- err
		}()
	}
	var ok, race int
	for range n {
		switch err := <-errs; {
		case err == nil:
			ok++
		case errors.Is(err, auth.ErrRefreshRace):
			race++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	if ok != 1 || race != n-1 {
		t.Fatalf("ok=%d race=%d", ok, race)
	}
}
