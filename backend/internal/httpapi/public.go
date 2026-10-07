package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/store"
)

const (
	maxAuthorLen      = 64
	maxCommentLen     = 4000
	maxViewPathLen    = 512
	smallBodyLimit    = 16 << 10
	contentBodyLimit  = 8 << 20
	commentStatusResp = "pending"
)

type postSummary struct {
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	PublishedAt *time.Time `json:"published_at"`
}

type publicPost struct {
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	Status      string     `json:"status"`
	PublishedAt *time.Time `json:"published_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type publicComment struct {
	ID        int64     `json:"id"`
	Author    *string   `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type publicPage struct {
	Slug      string    `json:"slug"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	UpdatedAt time.Time `json:"updated_at"`
}

type publicNavItem struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

func (s *Server) listPosts(w http.ResponseWriter, r *http.Request) {
	pg, err := parsePage(r)
	if err != nil {
		badRequest(w, "%s", err)
		return
	}
	posts, total, err := s.store.PublishedPosts(r.Context(), pg)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	items := make([]postSummary, 0, len(posts))
	for _, p := range posts {
		items = append(items, postSummary{Slug: p.Slug, Title: p.Title, PublishedAt: p.PublishedAt})
	}
	writeJSON(w, http.StatusOK, list(items, total))
}

func (s *Server) getPost(w http.ResponseWriter, r *http.Request) {
	p, err := s.store.VisiblePostBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, publicPost{
		Slug: p.Slug, Title: p.Title, Body: p.Body, Status: p.Status,
		PublishedAt: p.PublishedAt, UpdatedAt: p.UpdatedAt,
	})
}

func (s *Server) listComments(w http.ResponseWriter, r *http.Request) {
	p, err := s.store.VisiblePostBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	comments, err := s.store.ApprovedComments(r.Context(), p.ID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	out := make([]publicComment, 0, len(comments))
	for _, c := range comments {
		out = append(out, publicComment{ID: c.ID, Author: c.Author, Body: c.Body, CreatedAt: c.CreatedAt})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) createComment(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Author  string `json:"author"`
		Body    string `json:"body"`
		Website string `json:"website"`
	}
	if !decodeJSON(w, r, &in, smallBodyLimit) {
		return
	}
	p, err := s.store.VisiblePostBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	pending := map[string]string{"status": commentStatusResp}
	if in.Website != "" {
		// Honeypot filled in: pretend success, store nothing.
		writeJSON(w, http.StatusCreated, pending)
		return
	}

	body := strings.TrimSpace(in.Body)
	author := strings.TrimSpace(in.Author)
	switch {
	case body == "":
		badRequest(w, "body is required")
		return
	case utf8.RuneCountInString(body) > maxCommentLen:
		badRequest(w, "body must be at most %d characters", maxCommentLen)
		return
	case utf8.RuneCountInString(author) > maxAuthorLen:
		badRequest(w, "author must be at most %d characters", maxAuthorLen)
		return
	}
	var authorPtr *string
	if author != "" {
		authorPtr = &author
	}

	ip := ClientIP(r, s.trusted)
	err = s.store.CreateComment(r.Context(), store.NewComment{
		PostID: p.ID,
		Author: authorPtr,
		Body:   body,
		IPHash: IPHash(s.ipSecret, ip, s.now()),
	})
	if errors.Is(err, store.ErrConflict) {
		writeError(w, http.StatusTooManyRequests, "only one comment per day is allowed")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, pending)
}

func (s *Server) getPage(w http.ResponseWriter, r *http.Request) {
	p, err := s.store.PageBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, publicPage{Slug: p.Slug, Title: p.Title, Body: p.Body, UpdatedAt: p.UpdatedAt})
}

func (s *Server) getNav(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.NavItems(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	out := map[string][]publicNavItem{store.NavHeader: {}, store.NavFooter: {}}
	for _, it := range items {
		out[it.Place] = append(out[it.Place], publicNavItem{Label: it.Label, URL: it.URL})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) addView(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Path string `json:"path"`
	}
	if !decodeJSON(w, r, &in, smallBodyLimit) {
		return
	}
	if !validViewPath(in.Path) {
		badRequest(w, "invalid path")
		return
	}
	if !s.viewLimit.Allow(ClientIP(r, s.trusted).String()) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err := s.store.AddView(r.Context(), in.Path); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validViewPath(p string) bool {
	if p == "" || p[0] != '/' || len(p) > maxViewPathLen || !utf8.ValidString(p) {
		return false
	}
	return !strings.HasPrefix(p, "/admin")
}
