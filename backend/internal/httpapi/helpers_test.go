package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/config"
)

func TestClientIP(t *testing.T) {
	trusted, err := config.ParsePrefixes("10.42.0.0/16, 127.0.0.1, ::1/128")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		remote string
		xff    []string
		want   string
	}{
		{"untrusted peer ignores xff", "203.0.113.9:4000", []string{"1.1.1.1"}, "203.0.113.9"},
		{"trusted peer without xff", "10.42.0.5:4000", nil, "10.42.0.5"},
		{"trusted peer one hop", "10.42.0.5:4000", []string{"198.51.100.7"}, "198.51.100.7"},
		{"skips trusted hops from right", "10.42.0.5:4000", []string{"198.51.100.7, 10.42.1.1, 10.42.0.9"}, "198.51.100.7"},
		{"spoofed left part ignored", "10.42.0.5:4000", []string{"6.6.6.6, 198.51.100.7"}, "198.51.100.7"},
		{"multiple headers", "10.42.0.5:4000", []string{"6.6.6.6", "198.51.100.7, 10.42.0.3"}, "198.51.100.7"},
		{"all trusted returns leftmost", "10.42.0.5:4000", []string{"10.42.3.3, 10.42.2.2"}, "10.42.3.3"},
		{"garbage stops walk", "10.42.0.5:4000", []string{"198.51.100.7, garbage, 10.42.2.2"}, "10.42.2.2"},
		{"ipv6 peer trusted", "[::1]:4000", []string{"2001:db8::1"}, "2001:db8::1"},
		{"ipv4 mapped", "[::ffff:127.0.0.1]:4000", []string{"198.51.100.7"}, "198.51.100.7"},
		{"xff with port", "127.0.0.1:1", []string{"198.51.100.7:5555"}, "198.51.100.7"},
		{"bad remote", "pipe", nil, "0.0.0.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = tt.remote
			for _, v := range tt.xff {
				r.Header.Add("X-Forwarded-For", v)
			}
			if got := ClientIP(r, trusted).String(); got != tt.want {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestIPHash(t *testing.T) {
	secret := []byte("s")
	ip := netip.MustParseAddr("198.51.100.7")
	day := time.Date(2026, 10, 6, 23, 59, 0, 0, time.UTC)

	m := hmac.New(sha256.New, secret)
	m.Write([]byte("198.51.100.7|2026-10-06"))
	want := hex.EncodeToString(m.Sum(nil))
	got := IPHash(secret, ip, day)
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
	if got != IPHash(secret, ip, day.Add(-23*time.Hour)) {
		t.Fatal("same UTC day must give the same hash")
	}
	if got == IPHash(secret, ip, day.Add(time.Minute)) {
		t.Fatal("next UTC day must differ")
	}
	// Local time zone must not matter.
	msk := time.FixedZone("MSK", 3*3600)
	if got != IPHash(secret, ip, day.In(msk)) {
		t.Fatal("hash depends on time zone")
	}
	if got == IPHash([]byte("other"), ip, day) || got == IPHash(secret, netip.MustParseAddr("198.51.100.8"), day) {
		t.Fatal("hash must depend on secret and ip")
	}
}

func TestValidSlug(t *testing.T) {
	for _, s := range []string{"hello", "a", "hello-world", "post-2026-10-06", "x1-y2"} {
		if !ValidSlug(s) {
			t.Errorf("%q should be valid", s)
		}
	}
	for _, s := range []string{"", "-a", "a-", "a--b", "Hello", "a_b", "a b", "a/b", "привет", "a.b"} {
		if ValidSlug(s) {
			t.Errorf("%q should be invalid", s)
		}
	}
}

func TestValidNavURL(t *testing.T) {
	for _, u := range []string{"/cv", "/", "https://github.com/x", "http://a.b", "mailto:me@x.y"} {
		if !ValidNavURL(u) {
			t.Errorf("%q should be valid", u)
		}
	}
	for _, u := range []string{"", "//evil.com", "javascript:alert(1)", "https://", "cv", "/a b"} {
		if ValidNavURL(u) {
			t.Errorf("%q should be invalid", u)
		}
	}
}

func TestValidViewPath(t *testing.T) {
	for _, p := range []string{"/", "/posts/hello", "/cv"} {
		if !validViewPath(p) {
			t.Errorf("%q should be valid", p)
		}
	}
	long := "/" + string(make([]byte, 512))
	for _, p := range []string{"", "posts", "/admin", "/admin/posts", long} {
		if validViewPath(p) {
			t.Errorf("%q should be invalid", p)
		}
	}
}

func TestRateLimiter(t *testing.T) {
	now := time.Unix(0, 0)
	l := newRateLimiter(2, time.Minute, func() time.Time { return now })
	for i, want := range []bool{true, true, false} {
		if got := l.Allow("a"); got != want {
			t.Fatalf("hit %d: got %v, want %v", i, got, want)
		}
	}
	if !l.Allow("b") {
		t.Fatal("keys must be independent")
	}
	now = now.Add(time.Minute)
	if !l.Allow("a") {
		t.Fatal("window did not reset")
	}
}

func TestSanitizeFileName(t *testing.T) {
	tests := map[string]string{
		"photo.jpg":           "photo.jpg",
		"C:\\Users\\x\\a.png": "a.png",
		"../../etc/passwd":    "passwd",
		"":                    "file",
		"..":                  "file",
		"a\x00b\nc.txt":       "abc.txt",
		"отчет 2026.pdf":      "отчет 2026.pdf",
	}
	for in, want := range tests {
		if got := SanitizeFileName(in); got != want {
			t.Errorf("SanitizeFileName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestContentDisposition(t *testing.T) {
	tests := []struct{ ct, name, want string }{
		{"image/png", "a.png", "inline; filename=a.png"},
		{"video/mp4", "v.mp4", "inline; filename=v.mp4"},
		{"audio/mpeg", "s.mp3", "inline; filename=s.mp3"},
		{"application/pdf", "doc.pdf", "attachment; filename=doc.pdf"},
		{"application/pdf", "my doc.pdf", `attachment; filename="my doc.pdf"`},
		{"text/plain", "файл.txt", "attachment; filename*=utf-8''%D1%84%D0%B0%D0%B9%D0%BB.txt"},
	}
	for _, tt := range tests {
		if got := ContentDisposition(tt.ct, tt.name); got != tt.want {
			t.Errorf("ContentDisposition(%q, %q) = %q, want %q", tt.ct, tt.name, got, tt.want)
		}
	}
}

func TestDetectContentType(t *testing.T) {
	if got := DetectContentType("image/png", "x.bin", nil); got != "image/png" {
		t.Errorf("declared: %s", got)
	}
	if got := DetectContentType("application/octet-stream", "x.pdf", nil); got != "application/pdf" {
		t.Errorf("by extension: %s", got)
	}
	if got := DetectContentType("", "noext", []byte("\x89PNG\r\n\x1a\n")); got != "image/png" {
		t.Errorf("sniffed: %s", got)
	}
}

func TestParseStatsRange(t *testing.T) {
	now := time.Date(2026, 10, 6, 15, 0, 0, 0, time.UTC)
	from, to, msg := parseStatsRange("", "", now)
	if msg != "" || !to.Equal(time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC)) ||
		!from.Equal(time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("default range %v %v %s", from, to, msg)
	}
	from, to, msg = parseStatsRange("2026-10-01", "2026-10-06", now)
	if msg != "" || from.Day() != 1 || to.Day() != 6 {
		t.Fatalf("explicit range %v %v %s", from, to, msg)
	}
	if _, _, msg = parseStatsRange("2026-10-07", "2026-10-06", now); msg == "" {
		t.Fatal("inverted range accepted")
	}
	if _, _, msg = parseStatsRange("bad", "", now); msg == "" {
		t.Fatal("bad date accepted")
	}
}
