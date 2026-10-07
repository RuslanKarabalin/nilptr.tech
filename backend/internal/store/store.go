// Package store defines the domain types and errors shared by the
// storage implementation, the auth service and the HTTP handlers.
package store

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrNotFound is returned when the requested row does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConflict is returned on a unique constraint violation or when a
	// conditional update lost a race.
	ErrConflict = errors.New("conflict")
)

// Post statuses.
const (
	PostDraft     = "draft"
	PostUnlisted  = "unlisted"
	PostPublished = "published"
)

// Comment statuses.
const (
	CommentPending  = "pending"
	CommentApproved = "approved"
	CommentRejected = "rejected"
)

// Refresh token revoke reasons.
const (
	RevokeRotated = "rotated"
	RevokeLogout  = "logout"
	RevokeReuse   = "reuse"
	RevokeManual  = "manual"
)

// Nav places.
const (
	NavHeader = "header"
	NavFooter = "footer"
)

// Page is a request window for list queries.
type Page struct {
	Limit  int
	Offset int
}

type User struct {
	ID           uuid.UUID
	Login        string
	PasswordHash string
}

type RefreshToken struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	TokenHash    string
	FamilyID     uuid.UUID
	UserAgent    string
	IP           string
	CreatedAt    time.Time
	LastUsedAt   time.Time
	ExpiresAt    time.Time
	RevokedAt    *time.Time
	RevokeReason *string
}

// Session is one active refresh token family.
type Session struct {
	ID         uuid.UUID `json:"id"`
	UserAgent  string    `json:"user_agent"`
	IP         string    `json:"ip"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
}

type Post struct {
	ID          uuid.UUID  `json:"id"`
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	PublishedAt *time.Time `json:"published_at"`
}

// PostInput is the editable part of a post.
type PostInput struct {
	Slug   string
	Title  string
	Body   string
	Status string
}

type PageDoc struct {
	ID        uuid.UUID `json:"id"`
	Slug      string    `json:"slug"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PageInput is the editable part of a page.
type PageInput struct {
	Slug  string
	Title string
	Body  string
}

type NavItem struct {
	ID    int64  `json:"id"`
	Place string `json:"-"`
	Label string `json:"label"`
	URL   string `json:"url"`
}

type File struct {
	ID          uuid.UUID
	Name        string
	ContentType string
	Size        int64
	S3Key       string
	CreatedAt   time.Time
}

type Comment struct {
	ID        int64     `json:"id"`
	PostSlug  string    `json:"post_slug"`
	PostTitle string    `json:"post_title"`
	Author    *string   `json:"author"`
	Body      string    `json:"body"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// NewComment is a comment to insert.
type NewComment struct {
	PostID uuid.UUID
	Author *string
	Body   string
	IPHash string
}

type PathCount struct {
	Path  string `json:"path"`
	Count int64  `json:"count"`
}
