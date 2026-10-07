package httpapi

import (
	"bufio"
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/objstore"
	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/store"
)

const maxFileNameLen = 255

type fileMeta struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	URL         string    `json:"url"`
}

type adminFileItem struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
	URL         string    `json:"url"`
}

// FileURL is the permanent public link of a file.
func FileURL(id uuid.UUID, name string) string {
	return "/files/" + id.String() + "/" + url.PathEscape(name)
}

func toAdminFile(f store.File) adminFileItem {
	return adminFileItem{
		ID: f.ID, Name: f.Name, ContentType: f.ContentType, Size: f.Size,
		CreatedAt: f.CreatedAt, URL: FileURL(f.ID, f.Name),
	}
}

func (s *Server) getFileMeta(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(r, "id")
	if !ok {
		notFound(w, r)
		return
	}
	f, err := s.store.FileByID(r.Context(), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, fileMeta{
		ID: f.ID, Name: f.Name, ContentType: f.ContentType, Size: f.Size, URL: FileURL(f.ID, f.Name),
	})
}

// serveFile streams the object with Range and HEAD support.
func (s *Server) serveFile(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(r, "id")
	if !ok {
		notFound(w, r)
		return
	}
	f, err := s.store.FileByID(r.Context(), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	obj, err := s.objects.Open(r.Context(), f.S3Key)
	if errors.Is(err, objstore.ErrNotFound) {
		notFound(w, r)
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	defer func() { _ = obj.Close() }()

	h := w.Header()
	h.Set("Content-Type", f.ContentType)
	h.Set("Cache-Control", "public, max-age=31536000, immutable")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Content-Security-Policy", "sandbox")
	h.Set("Content-Disposition", ContentDisposition(f.ContentType, f.Name))
	http.ServeContent(w, r, "", f.CreatedAt, obj)
}

// ContentDisposition returns "inline" for images, video and audio and
// "attachment" otherwise, with the file name.
func ContentDisposition(contentType, name string) string {
	kind := "attachment"
	for _, p := range []string{"image/", "video/", "audio/"} {
		if strings.HasPrefix(contentType, p) {
			kind = "inline"
			break
		}
	}
	if v := mime.FormatMediaType(kind, map[string]string{"filename": name}); v != "" {
		return v
	}
	return kind
}

func (s *Server) adminListFiles(w http.ResponseWriter, r *http.Request) {
	pg, err := parsePage(r)
	if err != nil {
		badRequest(w, "%s", err)
		return
	}
	files, total, err := s.store.Files(r.Context(), pg)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	items := make([]adminFileItem, 0, len(files))
	for _, f := range files {
		items = append(items, toAdminFile(f))
	}
	writeJSON(w, http.StatusOK, list(items, total))
}

var errFileTooLarge = errors.New("file too large")

// capReader fails with errFileTooLarge once more than n bytes are read.
type capReader struct {
	r io.Reader
	n int64
}

func (c *capReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n -= int64(n)
	if c.n < 0 {
		return n, errFileTooLarge
	}
	return n, err
}

func (s *Server) adminUploadFile(w http.ResponseWriter, r *http.Request) {
	// Allow some room for the multipart envelope around the file.
	r.Body = http.MaxBytesReader(w, r.Body, s.maxUpload+1<<20)
	mr, err := r.MultipartReader()
	if err != nil {
		badRequest(w, "expected multipart/form-data")
		return
	}
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			badRequest(w, "field file is required")
			return
		}
		if err != nil {
			s.uploadError(w, r, err)
			return
		}
		if part.FormName() != "file" {
			_ = part.Close()
			continue
		}
		s.storeUpload(w, r, part.FileName(), part.Header.Get("Content-Type"), part)
		return
	}
}

func (s *Server) storeUpload(w http.ResponseWriter, r *http.Request, rawName, declaredType string, body io.Reader) {
	name := SanitizeFileName(rawName)
	br := bufio.NewReaderSize(body, 512)
	head, err := br.Peek(512)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, bufio.ErrBufferFull) {
		s.uploadError(w, r, err)
		return
	}
	contentType := DetectContentType(declaredType, name, head)

	id := uuid.New()
	key := id.String()
	size, err := s.objects.Put(r.Context(), key, &capReader{r: br, n: s.maxUpload}, -1, contentType)
	if err != nil {
		s.removeObject(r.Context(), key)
		s.uploadError(w, r, err)
		return
	}
	f, err := s.store.CreateFile(r.Context(), store.File{
		ID: id, Name: name, ContentType: contentType, Size: size, S3Key: key,
	})
	if err != nil {
		s.removeObject(r.Context(), key)
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toAdminFile(f))
}

func (s *Server) uploadError(w http.ResponseWriter, r *http.Request, err error) {
	var mbe *http.MaxBytesError
	if errors.Is(err, errFileTooLarge) || errors.As(err, &mbe) {
		writeError(w, http.StatusRequestEntityTooLarge, "file too large")
		return
	}
	if strings.Contains(err.Error(), "multipart") {
		badRequest(w, "malformed multipart body")
		return
	}
	s.fail(w, r, err)
}

func (s *Server) removeObject(ctx context.Context, key string) {
	if err := s.objects.Remove(context.WithoutCancel(ctx), key); err != nil {
		s.log.WarnContext(ctx, "remove object", "key", key, "err", err)
	}
}

func (s *Server) adminDeleteFile(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(r, "id")
	if !ok {
		notFound(w, r)
		return
	}
	f, err := s.store.FileByID(r.Context(), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if err := s.objects.Remove(r.Context(), f.S3Key); err != nil && !errors.Is(err, objstore.ErrNotFound) {
		s.fail(w, r, err)
		return
	}
	if err := s.store.DeleteFile(r.Context(), id); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SanitizeFileName keeps the base name, drops control characters and
// limits the length.
func SanitizeFileName(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = filepath.Base(name)
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || r == '/' {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "file"
	}
	for len(name) > maxFileNameLen {
		_, size := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-size]
	}
	return name
}

// DetectContentType picks the declared type, then the type by extension,
// then sniffs the first bytes.
func DetectContentType(declared, name string, head []byte) string {
	if declared != "" && declared != "application/octet-stream" {
		if mt, params, err := mime.ParseMediaType(declared); err == nil {
			return mime.FormatMediaType(mt, params)
		}
	}
	if t := mime.TypeByExtension(strings.ToLower(filepath.Ext(name))); t != "" {
		return t
	}
	return http.DetectContentType(head)
}
