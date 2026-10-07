// Package httpapi implements the HTTP API described in docs/API.md.
package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/netip"
	"time"

	"github.com/google/uuid"

	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/auth"
	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/objstore"
	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/store"
)

// Store is the content storage used by the handlers.
type Store interface {
	Ping(ctx context.Context) error
	UserByID(ctx context.Context, id uuid.UUID) (store.User, error)

	PublishedPosts(ctx context.Context, pg store.Page) ([]store.Post, int, error)
	VisiblePostBySlug(ctx context.Context, slug string) (store.Post, error)
	AdminPosts(ctx context.Context, status string, pg store.Page) ([]store.Post, int, error)
	PostByID(ctx context.Context, id uuid.UUID) (store.Post, error)
	CreatePost(ctx context.Context, in store.PostInput) (store.Post, error)
	UpdatePost(ctx context.Context, id uuid.UUID, in store.PostInput) (store.Post, error)
	DeletePost(ctx context.Context, id uuid.UUID) error

	PageBySlug(ctx context.Context, slug string) (store.PageDoc, error)
	PageByID(ctx context.Context, id uuid.UUID) (store.PageDoc, error)
	AdminPages(ctx context.Context, pg store.Page) ([]store.PageDoc, int, error)
	CreatePage(ctx context.Context, in store.PageInput) (store.PageDoc, error)
	UpdatePage(ctx context.Context, id uuid.UUID, in store.PageInput) (store.PageDoc, error)
	DeletePage(ctx context.Context, id uuid.UUID) error

	NavItems(ctx context.Context) ([]store.NavItem, error)
	ReplaceNav(ctx context.Context, items []store.NavItem) error

	FileByID(ctx context.Context, id uuid.UUID) (store.File, error)
	Files(ctx context.Context, pg store.Page) ([]store.File, int, error)
	CreateFile(ctx context.Context, f store.File) (store.File, error)
	DeleteFile(ctx context.Context, id uuid.UUID) error

	ApprovedComments(ctx context.Context, postID uuid.UUID) ([]store.Comment, error)
	CreateComment(ctx context.Context, c store.NewComment) error
	AdminComments(ctx context.Context, status string, pg store.Page) ([]store.Comment, int, error)
	SetCommentStatus(ctx context.Context, id int64, status string) error
	DeleteComment(ctx context.Context, id int64) error

	AddView(ctx context.Context, path string) error
	ViewStats(ctx context.Context, from, to time.Time) ([]store.PathCount, error)
}

// Objects is the file content storage.
type Objects interface {
	Ping(ctx context.Context) error
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) (int64, error)
	Open(ctx context.Context, key string) (objstore.Object, error)
	Remove(ctx context.Context, key string) error
}

// Options configure the server.
type Options struct {
	Store          Store
	Auth           *auth.Service
	Objects        Objects
	Log            *slog.Logger
	TrustedProxies []netip.Prefix
	IPHashSecret   []byte
	CookieSecure   bool
	MaxUploadBytes int64
	// Now is the clock; nil means time.Now.
	Now func() time.Time
}

// Server holds the handler dependencies.
type Server struct {
	store     Store
	auth      *auth.Service
	objects   Objects
	log       *slog.Logger
	trusted   []netip.Prefix
	ipSecret  []byte
	secure    bool
	maxUpload int64
	now       func() time.Time

	viewLimit  *rateLimiter
	loginLimit *rateLimiter
}

// New creates the server.
func New(o Options) *Server {
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.Log == nil {
		o.Log = slog.New(slog.DiscardHandler)
	}
	return &Server{
		store:      o.Store,
		auth:       o.Auth,
		objects:    o.Objects,
		log:        o.Log,
		trusted:    o.TrustedProxies,
		ipSecret:   o.IPHashSecret,
		secure:     o.CookieSecure,
		maxUpload:  o.MaxUploadBytes,
		now:        o.Now,
		viewLimit:  newRateLimiter(60, time.Minute, o.Now),
		loginLimit: newRateLimiter(10, time.Minute, o.Now),
	}
}

// Handler returns the root handler with all routes and middleware.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("GET /readyz", s.readyz)

	mux.HandleFunc("GET /api/posts", s.listPosts)
	mux.HandleFunc("GET /api/posts/{slug}", s.getPost)
	mux.HandleFunc("GET /api/posts/{slug}/comments", s.listComments)
	mux.HandleFunc("POST /api/posts/{slug}/comments", s.createComment)
	mux.HandleFunc("GET /api/pages/{slug}", s.getPage)
	mux.HandleFunc("GET /api/nav", s.getNav)
	mux.HandleFunc("GET /api/files/{id}", s.getFileMeta)
	mux.HandleFunc("POST /api/views", s.addView)
	mux.HandleFunc("GET /files/{id}", s.serveFile)
	mux.HandleFunc("GET /files/{id}/{name...}", s.serveFile)

	mux.HandleFunc("POST /api/admin/login", s.login)
	mux.HandleFunc("POST /api/admin/refresh", s.refresh)
	mux.Handle("POST /api/admin/logout", s.admin(s.logout))
	mux.Handle("GET /api/admin/me", s.admin(s.me))
	mux.Handle("GET /api/admin/sessions", s.admin(s.listSessions))
	mux.Handle("DELETE /api/admin/sessions/{id}", s.admin(s.revokeSession))

	mux.Handle("GET /api/admin/posts", s.admin(s.adminListPosts))
	mux.Handle("POST /api/admin/posts", s.admin(s.adminCreatePost))
	mux.Handle("GET /api/admin/posts/{id}", s.admin(s.adminGetPost))
	mux.Handle("PUT /api/admin/posts/{id}", s.admin(s.adminUpdatePost))
	mux.Handle("DELETE /api/admin/posts/{id}", s.admin(s.adminDeletePost))

	mux.Handle("GET /api/admin/pages", s.admin(s.adminListPages))
	mux.Handle("POST /api/admin/pages", s.admin(s.adminCreatePage))
	mux.Handle("GET /api/admin/pages/{id}", s.admin(s.adminGetPage))
	mux.Handle("PUT /api/admin/pages/{id}", s.admin(s.adminUpdatePage))
	mux.Handle("DELETE /api/admin/pages/{id}", s.admin(s.adminDeletePage))

	mux.Handle("GET /api/admin/nav", s.admin(s.adminGetNav))
	mux.Handle("PUT /api/admin/nav", s.admin(s.adminPutNav))

	mux.Handle("GET /api/admin/files", s.admin(s.adminListFiles))
	mux.Handle("POST /api/admin/files", s.admin(s.adminUploadFile))
	mux.Handle("DELETE /api/admin/files/{id}", s.admin(s.adminDeleteFile))

	mux.Handle("GET /api/admin/comments", s.admin(s.adminListComments))
	mux.Handle("POST /api/admin/comments/{id}/approve", s.admin(s.adminSetCommentStatus(store.CommentApproved)))
	mux.Handle("POST /api/admin/comments/{id}/reject", s.admin(s.adminSetCommentStatus(store.CommentRejected)))
	mux.Handle("DELETE /api/admin/comments/{id}", s.admin(s.adminDeleteComment))

	mux.Handle("GET /api/admin/stats", s.admin(s.adminStats))

	mux.HandleFunc("/", notFound)

	return s.logRequests(s.recoverPanics(mux))
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := s.store.Ping(ctx); err != nil {
		s.log.WarnContext(ctx, "readyz: postgres", "err", err)
		writeError(w, http.StatusServiceUnavailable, "postgres unavailable")
		return
	}
	if err := s.objects.Ping(ctx); err != nil {
		s.log.WarnContext(ctx, "readyz: s3", "err", err)
		writeError(w, http.StatusServiceUnavailable, "s3 unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
