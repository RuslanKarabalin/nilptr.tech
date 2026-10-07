package postgres_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/auth"
	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/httpapi"
	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/migrate"
	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/objstore"
	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/store/postgres"
)

// This test runs only when TEST_DATABASE_URL points to an empty
// PostgreSQL database, for example:
//
//	TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable go test ./internal/store/postgres/
//
// It applies the migrations and drives the whole API through the real
// store, with an in-memory object storage.

type memObjects struct {
	mu sync.Mutex
	m  map[string][]byte
}

type readSeekCloser struct{ *bytes.Reader }

func (readSeekCloser) Close() error { return nil }

func (o *memObjects) Ping(context.Context) error { return nil }

func (o *memObjects) Put(_ context.Context, k string, r io.Reader, _ int64, _ string) (int64, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.m[k] = b
	return int64(len(b)), nil
}

func (o *memObjects) Open(_ context.Context, k string) (objstore.Object, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	b, ok := o.m[k]
	if !ok {
		return nil, objstore.ErrNotFound
	}
	return readSeekCloser{bytes.NewReader(b)}, nil
}

func (o *memObjects) Remove(_ context.Context, k string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.m, k)
	return nil
}

func check(t *testing.T, cond bool, msg string, args ...any) {
	t.Helper()
	if !cond {
		t.Errorf(msg, args...)
	}
}

func TestIntegration(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	log := slog.New(slog.DiscardHandler)

	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	check(t, migrate.Run(ctx, conn, log) == nil, "migrate first run")
	check(t, migrate.Run(ctx, conn, log) == nil, "migrate second run must be a no-op")
	var n int
	_ = conn.QueryRow(ctx, "select count(*) from schema_migrations").Scan(&n)
	check(t, n == 1, "schema_migrations rows: %d", n)
	_ = conn.Close(ctx)

	db, err := postgres.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	h, _ := auth.HashPassword("pw")
	check(t, db.UpsertUser(ctx, "admin", h) == nil, "upsert admin")
	h2, _ := auth.HashPassword("pw2")
	check(t, db.UpsertUser(ctx, "admin", h2) == nil, "upsert admin again")

	now := time.Now()
	var mu sync.Mutex
	clock := func() time.Time { mu.Lock(); defer mu.Unlock(); return now }
	srv := httpapi.New(httpapi.Options{
		Store: db, Auth: auth.NewService(db, []byte(strings.Repeat("k", 32)), clock),
		Objects: &memObjects{m: map[string][]byte{}}, Log: log, IPHashSecret: []byte("x"),
		MaxUploadBytes: 1 << 20, Now: clock,
	}).Handler()

	do := func(method, path, body string, cookies []*http.Cookie, hdr ...string) *httptest.ResponseRecorder {
		var r io.Reader
		if body != "" {
			r = strings.NewReader(body)
		}
		req := httptest.NewRequest(method, path, r)
		req.Header.Set("User-Agent", "ua")
		for i := 0; i+1 < len(hdr); i += 2 {
			req.Header.Set(hdr[i], hdr[i+1])
		}
		for _, c := range cookies {
			req.AddCookie(&http.Cookie{Name: c.Name, Value: c.Value})
		}
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		return w
	}
	cookie := func(w *httptest.ResponseRecorder, name string) *http.Cookie {
		for _, c := range w.Result().Cookies() {
			if c.Name == name {
				return c
			}
		}
		return nil
	}

	w := do("POST", "/api/admin/login", `{"login":"admin","password":"pw"}`, nil)
	check(t, w.Code == 401, "old password rejected after upsert")
	w = do("POST", "/api/admin/login", `{"login":"admin","password":"pw2"}`, nil)
	check(t, w.Code == 204, "login %d", w.Code)
	at, rt := cookie(w, "access_token"), cookie(w, "refresh_token")
	A := []*http.Cookie{at}

	w = do("GET", "/api/admin/me", "", A)
	check(t, w.Code == 200 && strings.Contains(w.Body.String(), `"login":"admin"`), "me %s", w.Body)
	w = do("GET", "/api/admin/sessions", "", A)
	check(t, w.Code == 200 && strings.Contains(w.Body.String(), `"current":true`) && strings.Contains(w.Body.String(), `"ip":"192.0.2.1"`), "sessions %s", w.Body)

	// concurrent refresh against real DB
	var wg sync.WaitGroup
	codes := make([]int, 6)
	var newRT *http.Cookie
	var nmu sync.Mutex
	for i := range codes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := do("POST", "/api/admin/refresh", "", []*http.Cookie{rt})
			codes[i] = r.Code
			if r.Code == 204 {
				nmu.Lock()
				newRT = cookie(r, "refresh_token")
				A = []*http.Cookie{cookie(r, "access_token")}
				nmu.Unlock()
			}
		}()
	}
	wg.Wait()
	ok := 0
	for _, c := range codes {
		if c == 204 {
			ok++
		}
	}
	check(t, ok == 1, "concurrent refresh: exactly one success %v", codes)
	w = do("GET", "/api/admin/me", "", A)
	check(t, w.Code == 200, "family still active after race")

	mu.Lock()
	now = now.Add(11 * time.Second)
	mu.Unlock()
	w = do("POST", "/api/admin/refresh", "", []*http.Cookie{rt})
	check(t, w.Code == 401, "reuse after grace 401")
	w = do("POST", "/api/admin/refresh", "", []*http.Cookie{newRT})
	check(t, w.Code == 401, "family revoked after reuse")
	w = do("GET", "/api/admin/me", "", A)
	check(t, w.Code == 401, "access token invalid after family revoked")

	w = do("POST", "/api/admin/login", `{"login":"admin","password":"pw2"}`, nil)
	A = []*http.Cookie{cookie(w, "access_token")}

	// posts
	w = do("POST", "/api/admin/posts", `{"slug":"hello","title":"Hello","body":"# hi","status":"draft"}`, A)
	check(t, w.Code == 201 && strings.Contains(w.Body.String(), `"published_at":null`), "create draft %s", w.Body)
	var p map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &p)
	id := p["id"].(string)
	w = do("POST", "/api/admin/posts", `{"slug":"hello","title":"Dup","body":"","status":"draft"}`, A)
	check(t, w.Code == 409, "duplicate slug 409 (%d)", w.Code)
	check(t, do("GET", "/api/posts/hello", "", nil).Code == 404, "draft hidden")
	w = do("PUT", "/api/admin/posts/"+id, `{"slug":"hello","title":"Hello","body":"# hi","status":"published"}`, A)
	_ = json.Unmarshal(w.Body.Bytes(), &p)
	pub := p["published_at"]
	check(t, w.Code == 200 && pub != nil, "publish sets published_at %v", pub)
	time.Sleep(10 * time.Millisecond)
	w = do("PUT", "/api/admin/posts/"+id, `{"slug":"hello","title":"Hello","body":"x","status":"unlisted"}`, A)
	_ = json.Unmarshal(w.Body.Bytes(), &p)
	w = do("PUT", "/api/admin/posts/"+id, `{"slug":"hello","title":"Hello","body":"x","status":"published"}`, A)
	_ = json.Unmarshal(w.Body.Bytes(), &p)
	check(t, p["published_at"] == pub, "published_at kept %v vs %v", p["published_at"], pub)
	w = do("GET", "/api/posts", "", nil)
	check(t, strings.Contains(w.Body.String(), `"total":1`), "public list %s", w.Body)
	w = do("GET", "/api/admin/posts?status=published", "", A)
	check(t, strings.Contains(w.Body.String(), `"total":1`) && !strings.Contains(w.Body.String(), `"body"`), "admin list %s", w.Body)
	check(t, do("GET", "/api/admin/posts/"+id, "", A).Code == 200, "admin get post")
	check(t, do("GET", "/api/admin/posts/not-uuid", "", A).Code == 404, "admin get bad uuid 404")

	// comments
	w = do("POST", "/api/posts/hello/comments", `{"body":"nice","author":"bob"}`, nil)
	check(t, w.Code == 201, "comment 201 %s", w.Body)
	w = do("POST", "/api/posts/hello/comments", `{"body":"again"}`, nil)
	check(t, w.Code == 429, "comment 429 (%d)", w.Code)
	w = do("GET", "/api/posts/hello/comments", "", nil)
	check(t, w.Body.String() == "[]\n", "no approved comments yet %q", w.Body)
	w = do("GET", "/api/admin/comments?status=pending", "", A)
	check(t, strings.Contains(w.Body.String(), `"post_slug":"hello"`), "admin comments %s", w.Body)
	var cl struct{ Items []struct{ ID int64 } }
	_ = json.Unmarshal(w.Body.Bytes(), &cl)
	check(t, do("POST", fmt.Sprintf("/api/admin/comments/%d/approve", cl.Items[0].ID), "", A).Code == 204, "approve")
	w = do("GET", "/api/posts/hello/comments", "", nil)
	check(t, strings.Contains(w.Body.String(), `"author":"bob"`), "approved visible %s", w.Body)
	check(t, do("POST", "/api/admin/comments/999/reject", "", A).Code == 404, "reject missing 404")

	// pages & nav
	w = do("POST", "/api/admin/pages", `{"slug":"cv","title":"CV","body":"me"}`, A)
	check(t, w.Code == 201, "create page %s", w.Body)
	check(t, do("GET", "/api/pages/cv", "", nil).Code == 200, "public page")
	check(t, strings.Contains(do("GET", "/api/admin/pages", "", A).Body.String(), `"total":1`), "admin pages list")
	w = do("PUT", "/api/admin/nav", `{"header":[{"label":"CV","url":"/cv"},{"label":"Blog","url":"/"}],"footer":[{"label":"GitHub","url":"https://github.com/RuslanKarabalin"}]}`, A)
	check(t, w.Code == 200, "put nav %s", w.Body)
	w = do("PUT", "/api/admin/nav", `{"header":[{"label":"Blog","url":"/"},{"label":"CV","url":"/cv"}],"footer":[]}`, A)
	check(t, w.Code == 200, "put nav again %s", w.Body)
	w = do("GET", "/api/nav", "", nil)
	check(t, w.Body.String() == `{"footer":[],"header":[{"label":"Blog","url":"/"},{"label":"CV","url":"/cv"}]}`+"\n", "nav %s", w.Body)

	// views & stats
	for _, pth := range []string{"/", "/", "/posts/hello"} {
		do("POST", "/api/views", `{"path":"`+pth+`"}`, nil)
	}
	w = do("GET", "/api/admin/stats", "", A)
	check(t, strings.Contains(w.Body.String(), `{"items":[{"path":"/","count":2},{"path":"/posts/hello","count":1}],"total":3}`), "stats %s", w.Body)

	// files
	w = do("POST", "/api/admin/files", "--b\r\nContent-Disposition: form-data; name=\"file\"; filename=\"a.txt\"\r\nContent-Type: text/plain\r\n\r\nhello\r\n--b--\r\n", A, "Content-Type", "multipart/form-data; boundary=b")
	check(t, w.Code == 201, "upload %s", w.Body)
	var f map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &f)
	fid, _ := f["id"].(string)
	w = do("GET", "/files/"+fid+"/a.txt", "", nil)
	check(t, w.Code == 200 && w.Body.String() == "hello" && strings.HasPrefix(w.Header().Get("Content-Disposition"), "attachment"), "download %d %q", w.Code, w.Body)
	check(t, strings.Contains(do("GET", "/api/admin/files", "", A).Body.String(), `"total":1`), "files list")
	check(t, do("DELETE", "/api/admin/files/"+fid, "", A).Code == 204, "delete file")
	check(t, do("GET", "/files/"+fid, "", nil).Code == 404, "deleted file 404")

	// delete post cascades comments
	check(t, do("DELETE", "/api/admin/posts/"+id, "", A).Code == 204, "delete post")
	check(t, do("DELETE", "/api/admin/posts/"+id, "", A).Code == 404, "delete post again 404")

	// logout
	check(t, do("POST", "/api/admin/logout", "", A).Code == 204, "logout")
	check(t, do("GET", "/api/admin/me", "", A).Code == 401, "me after logout 401")
	check(t, do("GET", "/readyz", "", nil).Code == 200, "readyz")

}
