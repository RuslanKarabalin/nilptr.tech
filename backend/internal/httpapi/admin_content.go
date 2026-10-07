package httpapi

import (
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/store"
)

const (
	maxTitleLen    = 300
	maxNavLabelLen = 100
	maxNavURLLen   = 2048
	maxStatsDays   = 366
)

// ---- posts ----

type adminPostItem struct {
	ID          uuid.UUID  `json:"id"`
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	PublishedAt *time.Time `json:"published_at"`
}

func validPostStatus(s string) bool {
	return s == store.PostDraft || s == store.PostUnlisted || s == store.PostPublished
}

func (s *Server) adminListPosts(w http.ResponseWriter, r *http.Request) {
	pg, err := parsePage(r)
	if err != nil {
		badRequest(w, "%s", err)
		return
	}
	status := r.URL.Query().Get("status")
	if status != "" && !validPostStatus(status) {
		badRequest(w, "invalid status")
		return
	}
	posts, total, err := s.store.AdminPosts(r.Context(), status, pg)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	items := make([]adminPostItem, 0, len(posts))
	for _, p := range posts {
		items = append(items, adminPostItem{
			ID: p.ID, Slug: p.Slug, Title: p.Title, Status: p.Status,
			CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt, PublishedAt: p.PublishedAt,
		})
	}
	writeJSON(w, http.StatusOK, list(items, total))
}

// readPostInput decodes and validates a post body; it writes a 4xx and
// returns false on failure.
func readPostInput(w http.ResponseWriter, r *http.Request) (store.PostInput, bool) {
	var in struct {
		Slug   string `json:"slug"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		Status string `json:"status"`
	}
	if !decodeJSON(w, r, &in, contentBodyLimit) {
		return store.PostInput{}, false
	}
	out := store.PostInput{
		Slug:   strings.TrimSpace(in.Slug),
		Title:  strings.TrimSpace(in.Title),
		Body:   in.Body,
		Status: in.Status,
	}
	if out.Status == "" {
		out.Status = store.PostDraft
	}
	if msg := validateDoc(out.Slug, out.Title); msg != "" {
		badRequest(w, "%s", msg)
		return out, false
	}
	if !validPostStatus(out.Status) {
		badRequest(w, "status must be draft, unlisted or published")
		return out, false
	}
	return out, true
}

func validateDoc(slug, title string) string {
	if !ValidSlug(slug) {
		return "slug must match ^[a-z0-9]+(-[a-z0-9]+)*$"
	}
	if title == "" || utf8.RuneCountInString(title) > maxTitleLen {
		return "title is required and must be at most 300 characters"
	}
	return ""
}

func (s *Server) adminCreatePost(w http.ResponseWriter, r *http.Request) {
	in, ok := readPostInput(w, r)
	if !ok {
		return
	}
	p, err := s.store.CreatePost(r.Context(), in)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) adminGetPost(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(r, "id")
	if !ok {
		notFound(w, r)
		return
	}
	p, err := s.store.PostByID(r.Context(), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) adminUpdatePost(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(r, "id")
	if !ok {
		notFound(w, r)
		return
	}
	in, ok := readPostInput(w, r)
	if !ok {
		return
	}
	p, err := s.store.UpdatePost(r.Context(), id, in)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) adminDeletePost(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(r, "id")
	if !ok {
		notFound(w, r)
		return
	}
	if err := s.store.DeletePost(r.Context(), id); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- pages ----

type adminPageItem struct {
	ID        uuid.UUID `json:"id"`
	Slug      string    `json:"slug"`
	Title     string    `json:"title"`
	UpdatedAt time.Time `json:"updated_at"`
}

func readPageInput(w http.ResponseWriter, r *http.Request) (store.PageInput, bool) {
	var in struct {
		Slug  string `json:"slug"`
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if !decodeJSON(w, r, &in, contentBodyLimit) {
		return store.PageInput{}, false
	}
	out := store.PageInput{Slug: strings.TrimSpace(in.Slug), Title: strings.TrimSpace(in.Title), Body: in.Body}
	if msg := validateDoc(out.Slug, out.Title); msg != "" {
		badRequest(w, "%s", msg)
		return out, false
	}
	return out, true
}

func (s *Server) adminListPages(w http.ResponseWriter, r *http.Request) {
	pg, err := parsePage(r)
	if err != nil {
		badRequest(w, "%s", err)
		return
	}
	pages, total, err := s.store.AdminPages(r.Context(), pg)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	items := make([]adminPageItem, 0, len(pages))
	for _, p := range pages {
		items = append(items, adminPageItem{ID: p.ID, Slug: p.Slug, Title: p.Title, UpdatedAt: p.UpdatedAt})
	}
	writeJSON(w, http.StatusOK, list(items, total))
}

func (s *Server) adminCreatePage(w http.ResponseWriter, r *http.Request) {
	in, ok := readPageInput(w, r)
	if !ok {
		return
	}
	p, err := s.store.CreatePage(r.Context(), in)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) adminGetPage(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(r, "id")
	if !ok {
		notFound(w, r)
		return
	}
	p, err := s.store.PageByID(r.Context(), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) adminUpdatePage(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(r, "id")
	if !ok {
		notFound(w, r)
		return
	}
	in, ok := readPageInput(w, r)
	if !ok {
		return
	}
	p, err := s.store.UpdatePage(r.Context(), id, in)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) adminDeletePage(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(r, "id")
	if !ok {
		notFound(w, r)
		return
	}
	if err := s.store.DeletePage(r.Context(), id); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- navigation ----

func (s *Server) adminGetNav(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.NavItems(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	out := map[string][]store.NavItem{store.NavHeader: {}, store.NavFooter: {}}
	for _, it := range items {
		out[it.Place] = append(out[it.Place], it)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) adminPutNav(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Header []publicNavItem `json:"header"`
		Footer []publicNavItem `json:"footer"`
	}
	if !decodeJSON(w, r, &in, smallBodyLimit*4) {
		return
	}
	var items []store.NavItem
	for place, entries := range map[string][]publicNavItem{store.NavHeader: in.Header, store.NavFooter: in.Footer} {
		for i, it := range entries {
			label, url := strings.TrimSpace(it.Label), strings.TrimSpace(it.URL)
			if label == "" || utf8.RuneCountInString(label) > maxNavLabelLen {
				badRequest(w, "%s[%d]: label is required and must be at most %d characters", place, i, maxNavLabelLen)
				return
			}
			if !ValidNavURL(url) {
				badRequest(w, "%s[%d]: url must start with / or be an http(s) or mailto URL", place, i)
				return
			}
			items = append(items, store.NavItem{Place: place, Label: label, URL: url})
		}
	}
	// Keep a deterministic order: header first, then footer, each in input order.
	ordered := make([]store.NavItem, 0, len(items))
	for _, place := range []string{store.NavHeader, store.NavFooter} {
		for _, it := range items {
			if it.Place == place {
				ordered = append(ordered, it)
			}
		}
	}
	if err := s.store.ReplaceNav(r.Context(), ordered); err != nil {
		s.fail(w, r, err)
		return
	}
	s.adminGetNav(w, r)
}

// ValidNavURL accepts internal paths ("/cv") and absolute http, https and
// mailto URLs.
func ValidNavURL(u string) bool {
	if u == "" || len(u) > maxNavURLLen || strings.ContainsAny(u, " \t\r\n") {
		return false
	}
	if strings.HasPrefix(u, "/") {
		return !strings.HasPrefix(u, "//") && !strings.HasPrefix(u, "/\\")
	}
	lower := strings.ToLower(u)
	for _, p := range []string{"https://", "http://", "mailto:"} {
		if strings.HasPrefix(lower, p) && len(u) > len(p) {
			return true
		}
	}
	return false
}

// ---- comments ----

func (s *Server) adminListComments(w http.ResponseWriter, r *http.Request) {
	pg, err := parsePage(r)
	if err != nil {
		badRequest(w, "%s", err)
		return
	}
	status := r.URL.Query().Get("status")
	switch status {
	case "", store.CommentPending, store.CommentApproved, store.CommentRejected:
	default:
		badRequest(w, "invalid status")
		return
	}
	items, total, err := s.store.AdminComments(r.Context(), status, pg)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list(items, total))
}

func (s *Server) adminSetCommentStatus(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathInt(r, "id")
		if !ok {
			notFound(w, r)
			return
		}
		if err := s.store.SetCommentStatus(r.Context(), id, status); err != nil {
			s.fail(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) adminDeleteComment(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		notFound(w, r)
		return
	}
	if err := s.store.DeleteComment(r.Context(), id); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- stats ----

func (s *Server) adminStats(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseStatsRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"), s.now())
	if err != "" {
		badRequest(w, "%s", err)
		return
	}
	items, e := s.store.ViewStats(r.Context(), from, to.AddDate(0, 0, 1))
	if e != nil {
		s.fail(w, r, e)
		return
	}
	var total int
	for _, it := range items {
		total += int(it.Count)
	}
	writeJSON(w, http.StatusOK, list(items, total))
}

// parseStatsRange returns the inclusive UTC day range [from, to]. By
// default it is the last 30 days ending today.
func parseStatsRange(fromS, toS string, now time.Time) (from, to time.Time, errMsg string) {
	today := now.UTC().Truncate(24 * time.Hour)
	to, from = today, today.AddDate(0, 0, -29)
	var err error
	if toS != "" {
		if to, err = time.Parse(time.DateOnly, toS); err != nil {
			return from, to, "to must be YYYY-MM-DD"
		}
		if fromS == "" {
			from = to.AddDate(0, 0, -29)
		}
	}
	if fromS != "" {
		if from, err = time.Parse(time.DateOnly, fromS); err != nil {
			return from, to, "from must be YYYY-MM-DD"
		}
	}
	if to.Before(from) {
		return from, to, "from must not be after to"
	}
	if to.Sub(from) > maxStatsDays*24*time.Hour {
		return from, to, "range must be at most 366 days"
	}
	return from, to, ""
}
