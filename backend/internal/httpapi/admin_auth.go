package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/auth"
	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/store"
)

const (
	accessCookie  = "access_token"
	refreshCookie = "refresh_token"
	refreshPath   = "/admin"
)

func (s *Server) cookie(name, value, path string, maxAge time.Duration) *http.Cookie {
	c := &http.Cookie{ //nolint:gosec // Secure is configurable via COOKIE_SECURE for local development
		Name:     name,
		Value:    value,
		Path:     path,
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteStrictMode,
	}
	if maxAge > 0 {
		c.MaxAge = int(maxAge / time.Second)
	} else {
		c.MaxAge = -1 // Max-Age=0: delete
	}
	return c
}

func (s *Server) setAuthCookies(w http.ResponseWriter, t auth.Tokens) {
	http.SetCookie(w, s.cookie(accessCookie, t.Access, "/", auth.AccessTTL))
	http.SetCookie(w, s.cookie(refreshCookie, t.Refresh, refreshPath, auth.RefreshTTL))
	w.Header().Set("Cache-Control", "no-store")
}

func (s *Server) clearAuthCookies(w http.ResponseWriter) {
	http.SetCookie(w, s.cookie(accessCookie, "", "/", 0))
	http.SetCookie(w, s.cookie(refreshCookie, "", refreshPath, 0))
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &in, smallBodyLimit) {
		return
	}
	ip := ClientIP(r, s.trusted)
	if !s.loginLimit.Allow(ip.String()) {
		writeError(w, http.StatusTooManyRequests, "too many login attempts")
		return
	}
	tokens, err := s.auth.Login(r.Context(), in.Login, in.Password, r.UserAgent(), ip.String())
	if errors.Is(err, auth.ErrUnauthorized) {
		writeError(w, http.StatusUnauthorized, "invalid login or password")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.setAuthCookies(w, tokens)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	var raw string
	if c, err := r.Cookie(refreshCookie); err == nil {
		raw = c.Value
	}
	ip := ClientIP(r, s.trusted)
	tokens, err := s.auth.Refresh(r.Context(), raw, r.UserAgent(), ip.String())
	switch {
	case errors.Is(err, auth.ErrRefreshRace):
		// A concurrent refresh already rotated this token and set new
		// cookies; clearing them here would log the browser out.
		writeError(w, http.StatusUnauthorized, "refresh token already rotated")
		return
	case errors.Is(err, auth.ErrUnauthorized):
		s.clearAuthCookies(w)
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	case err != nil:
		s.fail(w, r, err)
		return
	}
	s.setAuthCookies(w, tokens)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if err := s.auth.Logout(r.Context(), identityFrom(r.Context())); err != nil {
		s.fail(w, r, err)
		return
	}
	s.clearAuthCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	u, err := s.store.UserByID(r.Context(), identityFrom(r.Context()).UserID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"id": u.ID.String(), "login": u.Login})
}

type sessionItem struct {
	store.Session
	Current bool `json:"current"`
}

func (s *Server) listSessions(w http.ResponseWriter, r *http.Request) {
	id := identityFrom(r.Context())
	sessions, err := s.auth.Sessions(r.Context(), id.UserID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	out := make([]sessionItem, 0, len(sessions))
	for _, sess := range sessions {
		out = append(out, sessionItem{Session: sess, Current: sess.ID == id.SessionID})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) revokeSession(w http.ResponseWriter, r *http.Request) {
	sid, ok := pathUUID(r, "id")
	if !ok {
		notFound(w, r)
		return
	}
	id := identityFrom(r.Context())
	if err := s.auth.RevokeSession(r.Context(), id.UserID, sid); err != nil {
		s.fail(w, r, err)
		return
	}
	if sid == id.SessionID {
		s.clearAuthCookies(w)
	}
	w.WriteHeader(http.StatusNoContent)
}
