package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/auth"
	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/auth/authtest"
	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/httpapi"
	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/objstore"
	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/store"
)

// fakeStore implements the methods the tests use; any other call panics
// through the nil embedded interface.
type fakeStore struct {
	httpapi.Store
	*authtest.MemStore

	mu       sync.Mutex
	posts    []store.Post
	comments []store.NewComment
	views    []string
	files    map[uuid.UUID]store.File
}

func (f *fakeStore) UserByID(ctx context.Context, id uuid.UUID) (store.User, error) {
	return f.MemStore.UserByID(ctx, id)
}

func (f *fakeStore) Ping(context.Context) error { return nil }

func (f *fakeStore) PublishedPosts(_ context.Context, pg store.Page) ([]store.Post, int, error) {
	var out []store.Post
	for _, p := range f.posts {
		if p.Status == store.PostPublished {
			out = append(out, p)
		}
	}
	total := len(out)
	if pg.Offset < len(out) {
		out = out[pg.Offset:min(len(out), pg.Offset+pg.Limit)]
	} else {
		out = nil
	}
	return out, total, nil
}

func (f *fakeStore) VisiblePostBySlug(_ context.Context, slug string) (store.Post, error) {
	for _, p := range f.posts {
		if p.Slug == slug && p.Status != store.PostDraft {
			return p, nil
		}
	}
	return store.Post{}, store.ErrNotFound
}

func (f *fakeStore) CreatePost(_ context.Context, in store.PostInput) (store.Post, error) {
	for _, p := range f.posts {
		if p.Slug == in.Slug {
			return store.Post{}, store.ErrConflict
		}
	}
	p := store.Post{ID: uuid.New(), Slug: in.Slug, Title: in.Title, Body: in.Body, Status: in.Status}
	f.posts = append(f.posts, p)
	return p, nil
}

func (f *fakeStore) CreateComment(_ context.Context, c store.NewComment) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, x := range f.comments {
		if x.IPHash == c.IPHash {
			return store.ErrConflict
		}
	}
	f.comments = append(f.comments, c)
	return nil
}

func (f *fakeStore) AddView(_ context.Context, path string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.views = append(f.views, path)
	return nil
}

func (f *fakeStore) FileByID(_ context.Context, id uuid.UUID) (store.File, error) {
	if file, ok := f.files[id]; ok {
		return file, nil
	}
	return store.File{}, store.ErrNotFound
}

func (f *fakeStore) CreateFile(_ context.Context, file store.File) (store.File, error) {
	file.CreatedAt = time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC)
	f.files[file.ID] = file
	return file, nil
}

type nopCloser struct{ *bytes.Reader }

func (nopCloser) Close() error { return nil }

type fakeObjects struct {
	mu   sync.Mutex
	data map[string][]byte
}

func (o *fakeObjects) Ping(context.Context) error { return nil }

func (o *fakeObjects) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) (int64, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.data[key] = b
	return int64(len(b)), nil
}

func (o *fakeObjects) Open(_ context.Context, key string) (objstore.Object, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	b, ok := o.data[key]
	if !ok {
		return nil, objstore.ErrNotFound
	}
	return nopCloser{bytes.NewReader(b)}, nil
}

func (o *fakeObjects) Remove(_ context.Context, key string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.data, key)
	return nil
}

type env struct {
	h       http.Handler
	store   *fakeStore
	objects *fakeObjects
	now     time.Time
}

const testUA = "test-agent"

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{now: time.Date(2026, 10, 6, 10, 0, 0, 0, time.UTC)}
	mem := authtest.New()
	hash, err := auth.HashPassword("pw")
	if err != nil {
		t.Fatal(err)
	}
	mem.AddUser("admin", hash)
	pub := e.now
	e.store = &fakeStore{
		MemStore: mem,
		files:    map[uuid.UUID]store.File{},
		posts: []store.Post{
			{ID: uuid.New(), Slug: "hello", Title: "Hello", Body: "hi", Status: store.PostPublished, PublishedAt: &pub},
			{ID: uuid.New(), Slug: "secret", Title: "Secret", Status: store.PostUnlisted},
			{ID: uuid.New(), Slug: "draft", Title: "Draft", Status: store.PostDraft},
		},
	}
	e.objects = &fakeObjects{data: map[string][]byte{}}
	clock := func() time.Time { return e.now }
	e.h = httpapi.New(httpapi.Options{
		Store:          e.store,
		Auth:           auth.NewService(mem, []byte("0123456789abcdef0123456789abcdef"), clock),
		Objects:        e.objects,
		TrustedProxies: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")},
		IPHashSecret:   []byte("ip-secret"),
		CookieSecure:   true,
		MaxUploadBytes: 1024,
		Now:            clock,
	}).Handler()
	return e
}

func (e *env) do(method, path, body string, mod ...func(*http.Request)) *httptest.ResponseRecorder {
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	req.Header.Set("User-Agent", testUA)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, m := range mod {
		m(req)
	}
	w := httptest.NewRecorder()
	e.h.ServeHTTP(w, req)
	return w
}

func withCookies(cs ...*http.Cookie) func(*http.Request) {
	return func(r *http.Request) {
		for _, c := range cs {
			r.AddCookie(&http.Cookie{Name: c.Name, Value: c.Value})
		}
	}
}

func fromIP(peer, xff string) func(*http.Request) {
	return func(r *http.Request) {
		r.RemoteAddr = peer + ":1234"
		if xff != "" {
			r.Header.Set("X-Forwarded-For", xff)
		}
	}
}

func cookieMap(w *httptest.ResponseRecorder) map[string]*http.Cookie {
	out := map[string]*http.Cookie{}
	for _, c := range w.Result().Cookies() {
		out[c.Name] = c
	}
	return out
}

func decode[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %q: %v", w.Body.String(), err)
	}
	return v
}

func wantStatus(t *testing.T, w *httptest.ResponseRecorder, code int) {
	t.Helper()
	if w.Code != code {
		t.Fatalf("status %d, want %d, body %s", w.Code, code, w.Body.String())
	}
}

func (e *env) login(t *testing.T) map[string]*http.Cookie {
	t.Helper()
	w := e.do("POST", "/api/admin/login", `{"login":"admin","password":"pw"}`)
	wantStatus(t, w, http.StatusNoContent)
	return cookieMap(w)
}

func TestHealth(t *testing.T) {
	e := newEnv(t)
	wantStatus(t, e.do("GET", "/healthz", ""), 200)
	wantStatus(t, e.do("GET", "/readyz", ""), 200)
	w := e.do("GET", "/nope", "")
	wantStatus(t, w, 404)
	if decode[map[string]string](t, w)["error"] == "" {
		t.Fatal("error body missing")
	}
}

func TestPublicPosts(t *testing.T) {
	e := newEnv(t)
	w := e.do("GET", "/api/posts", "")
	wantStatus(t, w, 200)
	list := decode[struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}](t, w)
	if list.Total != 1 || len(list.Items) != 1 || list.Items[0]["slug"] != "hello" {
		t.Fatalf("list %+v", list)
	}
	if _, ok := list.Items[0]["body"]; ok {
		t.Fatal("list items must not include body")
	}
	if list.Items[0]["published_at"] != "2026-10-06T10:00:00Z" {
		t.Fatalf("published_at %v", list.Items[0]["published_at"])
	}

	wantStatus(t, e.do("GET", "/api/posts?limit=0", ""), 400)
	wantStatus(t, e.do("GET", "/api/posts?offset=-1", ""), 400)
	wantStatus(t, e.do("GET", "/api/posts/hello", ""), 200)
	wantStatus(t, e.do("GET", "/api/posts/secret", ""), 200)
	wantStatus(t, e.do("GET", "/api/posts/draft", ""), 404)
}

func TestCreateComment(t *testing.T) {
	e := newEnv(t)
	client := fromIP("10.1.2.3", "198.51.100.7")

	wantStatus(t, e.do("POST", "/api/posts/draft/comments", `{"body":"x"}`, client), 404)
	wantStatus(t, e.do("POST", "/api/posts/hello/comments", `{"body":"  "}`, client), 400)
	wantStatus(t, e.do("POST", "/api/posts/hello/comments", `{"body":"x","author":"`+strings.Repeat("a", 65)+`"}`, client), 400)
	wantStatus(t, e.do("POST", "/api/posts/hello/comments", `{"body":"`+strings.Repeat("я", 4001)+`"}`, client), 400)

	// Honeypot: 201, nothing stored, limit not consumed.
	wantStatus(t, e.do("POST", "/api/posts/hello/comments", `{"body":"spam","website":"x"}`, client), 201)
	if len(e.store.comments) != 0 {
		t.Fatal("honeypot comment stored")
	}

	w := e.do("POST", "/api/posts/hello/comments", `{"body":"nice","author":""}`, client)
	wantStatus(t, w, 201)
	if decode[map[string]string](t, w)["status"] != "pending" {
		t.Fatal("want status pending")
	}
	if c := e.store.comments[0]; c.Author != nil || c.Body != "nice" || len(c.IPHash) != 64 {
		t.Fatalf("stored %+v", c)
	}

	// Same client, same day, other post: limit is site wide.
	wantStatus(t, e.do("POST", "/api/posts/secret/comments", `{"body":"again"}`, client), 429)
	// Another client is fine.
	wantStatus(t, e.do("POST", "/api/posts/hello/comments", `{"body":"hi"}`, fromIP("10.1.2.3", "198.51.100.8")), 201)
	// Next UTC day the first client may comment again.
	e.now = e.now.Add(14 * time.Hour)
	wantStatus(t, e.do("POST", "/api/posts/hello/comments", `{"body":"tomorrow"}`, client), 201)
}

func TestViews(t *testing.T) {
	e := newEnv(t)
	wantStatus(t, e.do("POST", "/api/views", `{"path":"/admin/posts"}`), 400)
	wantStatus(t, e.do("POST", "/api/views", `{"path":"posts"}`), 400)
	wantStatus(t, e.do("POST", "/api/views", `not json`), 400)
	for range 70 {
		wantStatus(t, e.do("POST", "/api/views", `{"path":"/posts/hello"}`), 204)
	}
	if len(e.store.views) != 60 {
		t.Fatalf("stored %d views, want 60 (rate limited)", len(e.store.views))
	}
}

func TestAdminRequiresAuth(t *testing.T) {
	e := newEnv(t)
	for _, p := range []string{"/api/admin/me", "/api/admin/posts", "/api/admin/sessions", "/api/admin/stats"} {
		wantStatus(t, e.do("GET", p, ""), 401)
	}
	wantStatus(t, e.do("GET", "/api/admin/me", "", withCookies(&http.Cookie{Name: "access_token", Value: "garbage"})), 401)
}

func TestLoginRefreshLogout(t *testing.T) {
	e := newEnv(t)
	wantStatus(t, e.do("POST", "/api/admin/login", `{"login":"admin","password":"bad"}`), 401)

	c := e.login(t)
	at, rt := c["access_token"], c["refresh_token"]
	if at == nil || rt == nil {
		t.Fatalf("cookies %v", c)
	}
	if at.Path != "/" || at.MaxAge != 900 || !at.HttpOnly || !at.Secure || at.SameSite != http.SameSiteStrictMode {
		t.Fatalf("access cookie %+v", at)
	}
	if rt.Path != "/admin" || rt.MaxAge != 30*24*3600 || !rt.HttpOnly || !rt.Secure || rt.SameSite != http.SameSiteStrictMode {
		t.Fatalf("refresh cookie %+v", rt)
	}

	w := e.do("GET", "/api/admin/me", "", withCookies(at))
	wantStatus(t, w, 200)
	if decode[map[string]string](t, w)["login"] != "admin" {
		t.Fatal("me")
	}

	// Bearer header works too.
	wantStatus(t, e.do("GET", "/api/admin/me", "", func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+at.Value)
	}), 200)

	// Refresh with a different user agent fails and clears cookies.
	w = e.do("POST", "/api/admin/refresh", "", withCookies(rt), func(r *http.Request) { r.Header.Set("User-Agent", "evil") })
	wantStatus(t, w, 401)
	if cm := cookieMap(w); cm["access_token"] == nil || cm["access_token"].MaxAge >= 0 {
		t.Fatal("cookies not cleared on failure")
	}

	// Proper refresh rotates.
	e.now = e.now.Add(time.Minute)
	w = e.do("POST", "/api/admin/refresh", "", withCookies(rt))
	wantStatus(t, w, 204)
	c2 := cookieMap(w)
	if c2["refresh_token"].Value == rt.Value {
		t.Fatal("refresh token not rotated")
	}

	// Concurrent duplicate within the grace window: 401 without clearing.
	w = e.do("POST", "/api/admin/refresh", "", withCookies(rt))
	wantStatus(t, w, 401)
	if len(w.Result().Cookies()) != 0 {
		t.Fatal("grace rejection must not touch cookies")
	}

	// Sessions list marks the current one.
	w = e.do("GET", "/api/admin/sessions", "", withCookies(c2["access_token"]))
	wantStatus(t, w, 200)
	sessions := decode[[]map[string]any](t, w)
	if len(sessions) != 1 || sessions[0]["current"] != true || sessions[0]["user_agent"] != testUA {
		t.Fatalf("sessions %+v", sessions)
	}

	// Logout revokes the session: the access token stops working at once.
	w = e.do("POST", "/api/admin/logout", "", withCookies(c2["access_token"]))
	wantStatus(t, w, 204)
	wantStatus(t, e.do("GET", "/api/admin/me", "", withCookies(c2["access_token"])), 401)
	wantStatus(t, e.do("POST", "/api/admin/refresh", "", withCookies(c2["refresh_token"])), 401)
}

func TestRefreshReuseAfterGrace(t *testing.T) {
	e := newEnv(t)
	c := e.login(t)
	w := e.do("POST", "/api/admin/refresh", "", withCookies(c["refresh_token"]))
	wantStatus(t, w, 204)
	c2 := cookieMap(w)

	e.now = e.now.Add(auth.RotationGrace)
	w = e.do("POST", "/api/admin/refresh", "", withCookies(c["refresh_token"]))
	wantStatus(t, w, 401)
	if len(w.Result().Cookies()) != 2 {
		t.Fatal("reuse must clear cookies")
	}
	// Whole family revoked.
	wantStatus(t, e.do("GET", "/api/admin/me", "", withCookies(c2["access_token"])), 401)
	wantStatus(t, e.do("POST", "/api/admin/refresh", "", withCookies(c2["refresh_token"])), 401)
}

func TestRevokeOtherSession(t *testing.T) {
	e := newEnv(t)
	a := e.login(t)
	b := e.login(t)
	w := e.do("GET", "/api/admin/sessions", "", withCookies(a["access_token"]))
	sessions := decode[[]map[string]any](t, w)
	if len(sessions) != 2 {
		t.Fatalf("sessions %v", sessions)
	}
	var other string
	for _, s := range sessions {
		if s["current"] != true {
			other = s["id"].(string)
		}
	}
	wantStatus(t, e.do("DELETE", "/api/admin/sessions/"+other, "", withCookies(a["access_token"])), 204)
	wantStatus(t, e.do("DELETE", "/api/admin/sessions/"+other, "", withCookies(a["access_token"])), 404)
	wantStatus(t, e.do("GET", "/api/admin/me", "", withCookies(b["access_token"])), 401)
	wantStatus(t, e.do("GET", "/api/admin/me", "", withCookies(a["access_token"])), 200)
}

func TestAdminCreatePostValidation(t *testing.T) {
	e := newEnv(t)
	at := withCookies(e.login(t)["access_token"])
	wantStatus(t, e.do("POST", "/api/admin/posts", `{"slug":"Bad Slug","title":"t","body":"","status":"draft"}`, at), 400)
	wantStatus(t, e.do("POST", "/api/admin/posts", `{"slug":"ok","title":"","body":"","status":"draft"}`, at), 400)
	wantStatus(t, e.do("POST", "/api/admin/posts", `{"slug":"ok","title":"t","body":"","status":"weird"}`, at), 400)
	wantStatus(t, e.do("POST", "/api/admin/posts", `{"slug":"hello","title":"t","body":"","status":"draft"}`, at), 409)
	w := e.do("POST", "/api/admin/posts", `{"slug":"new-post","title":"T","body":"b","status":"draft"}`, at)
	wantStatus(t, w, 201)
	p := decode[map[string]any](t, w)
	if p["id"] == "" || p["body"] != "b" || p["status"] != "draft" {
		t.Fatalf("created %+v", p)
	}
}

func TestFiles(t *testing.T) {
	e := newEnv(t)
	at := withCookies(e.login(t)["access_token"])

	upload := func(name, ct string, content []byte) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		_ = mw.WriteField("other", "x")
		h := make(map[string][]string)
		h["Content-Disposition"] = []string{`form-data; name="file"; filename="` + name + `"`}
		if ct != "" {
			h["Content-Type"] = []string{ct}
		}
		part, _ := mw.CreatePart(h)
		_, _ = part.Write(content)
		_ = mw.Close()
		return e.do("POST", "/api/admin/files", "", at, func(r *http.Request) {
			r.Body = io.NopCloser(&buf)
			r.ContentLength = int64(buf.Len())
			r.Header.Set("Content-Type", mw.FormDataContentType())
		})
	}

	wantStatus(t, upload("big.bin", "", bytes.Repeat([]byte("x"), 2048)), 413)
	if len(e.objects.data) != 0 {
		t.Fatal("oversized upload left an object")
	}

	content := []byte("0123456789")
	w := upload("my video.mp4", "video/mp4", content)
	wantStatus(t, w, 201)
	item := decode[map[string]any](t, w)
	id := item["id"].(string)
	if item["url"] != "/files/"+id+"/my%20video.mp4" || item["size"] != float64(10) || item["content_type"] != "video/mp4" {
		t.Fatalf("item %+v", item)
	}

	w = e.do("GET", "/api/files/"+id, "")
	wantStatus(t, w, 200)

	w = e.do("GET", "/files/"+id+"/whatever-name", "")
	wantStatus(t, w, 200)
	if w.Body.String() != string(content) ||
		w.Header().Get("Content-Type") != "video/mp4" ||
		w.Header().Get("Cache-Control") != "public, max-age=31536000, immutable" ||
		!strings.HasPrefix(w.Header().Get("Content-Disposition"), "inline") ||
		w.Header().Get("Accept-Ranges") != "bytes" {
		t.Fatalf("headers %v body %q", w.Header(), w.Body.String())
	}

	w = e.do("GET", "/files/"+id, "", func(r *http.Request) { r.Header.Set("Range", "bytes=2-4") })
	wantStatus(t, w, http.StatusPartialContent)
	if w.Body.String() != "234" || w.Header().Get("Content-Range") != "bytes 2-4/10" {
		t.Fatalf("range: %q %v", w.Body.String(), w.Header())
	}

	w = e.do("HEAD", "/files/"+id, "")
	wantStatus(t, w, 200)
	if w.Body.Len() != 0 || w.Header().Get("Content-Length") != "10" {
		t.Fatalf("head: %v", w.Header())
	}

	w = upload("doc.pdf", "application/pdf", []byte("%PDF"))
	wantStatus(t, w, 201)
	pdf := decode[map[string]any](t, w)["id"].(string)
	w = e.do("GET", "/files/"+pdf, "")
	if !strings.HasPrefix(w.Header().Get("Content-Disposition"), "attachment") {
		t.Fatalf("pdf disposition %q", w.Header().Get("Content-Disposition"))
	}

	wantStatus(t, e.do("GET", "/files/not-a-uuid", ""), 404)
	wantStatus(t, e.do("GET", "/files/"+uuid.NewString(), ""), 404)
}

func TestPanicRecovery(t *testing.T) {
	e := newEnv(t)
	// The fake store does not implement NavItems, so the embedded nil
	// interface panics; the middleware must turn it into a 500.
	w := e.do("GET", "/api/nav", "")
	wantStatus(t, w, 500)
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body["error"] == "" {
		t.Fatalf("body %q", w.Body.String())
	}
}
