package config

import (
	"strings"
	"testing"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func base() map[string]string {
	return map[string]string{
		"DATABASE_URL":   "postgres://x",
		"JWT_SECRET":     strings.Repeat("j", 32),
		"IP_HASH_SECRET": strings.Repeat("i", 32),
		"S3_ENDPOINT":    "garage:3900",
	}
}

func TestDefaults(t *testing.T) {
	c, err := LoadFrom(env(base()))
	if err != nil {
		t.Fatal(err)
	}
	if c.HTTPAddr != ":8080" || c.S3Region != "garage" || c.S3Bucket != "nilptr" ||
		!c.CookieSecure || c.S3UseSSL || c.MaxUploadBytes != 1<<30 || len(c.TrustedProxies) != 0 {
		t.Fatalf("defaults %+v", c)
	}
}

func TestErrors(t *testing.T) {
	m := base()
	m["JWT_SECRET"] = "short"
	m["COOKIE_SECURE"] = "maybe"
	m["ADMIN_LOGIN"] = "admin"
	m["TRUSTED_PROXIES"] = "10.0.0.0/33"
	_, err := LoadFrom(env(m))
	if err == nil {
		t.Fatal("expected errors")
	}
	for _, s := range []string{"JWT_SECRET", "COOKIE_SECURE", "ADMIN_PASSWORD", "TRUSTED_PROXIES"} {
		if !strings.Contains(err.Error(), s) {
			t.Errorf("error %q does not mention %s", err, s)
		}
	}
}

func TestParsePrefixes(t *testing.T) {
	ps, err := ParsePrefixes("10.42.0.0/16, 127.0.0.1/32,::1, 192.168.1.7")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"10.42.0.0/16", "127.0.0.1/32", "::1/128", "192.168.1.7/32"}
	if len(ps) != len(want) {
		t.Fatalf("got %v", ps)
	}
	for i, p := range ps {
		if p.String() != want[i] {
			t.Errorf("%d: got %s want %s", i, p, want[i])
		}
	}
}
