// Package config loads the backend configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"net/netip"
	"os"
	"strconv"
	"strings"
)

// Config holds all settings for the serve subcommand.
type Config struct {
	HTTPAddr       string
	DatabaseURL    string
	JWTSecret      []byte
	IPHashSecret   []byte
	AdminLogin     string
	AdminPassword  string
	S3Endpoint     string
	S3Bucket       string
	S3AccessKey    string
	S3SecretKey    string
	S3Region       string
	S3UseSSL       bool
	TrustedProxies []netip.Prefix
	CookieSecure   bool
	MaxUploadBytes int64
}

const minSecretLen = 32

// Load reads the configuration using os.Getenv.
func Load() (Config, error) {
	return LoadFrom(os.Getenv)
}

// LoadFrom reads the configuration using the given lookup function.
func LoadFrom(getenv func(string) string) (Config, error) {
	get := func(name, def string) string {
		if v := strings.TrimSpace(getenv(name)); v != "" {
			return v
		}
		return def
	}

	var errs []error
	c := Config{
		HTTPAddr:      get("HTTP_ADDR", ":8080"),
		DatabaseURL:   get("DATABASE_URL", ""),
		JWTSecret:     []byte(getenv("JWT_SECRET")),
		IPHashSecret:  []byte(getenv("IP_HASH_SECRET")),
		AdminLogin:    get("ADMIN_LOGIN", ""),
		AdminPassword: getenv("ADMIN_PASSWORD"),
		S3Endpoint:    get("S3_ENDPOINT", ""),
		S3Bucket:      get("S3_BUCKET", "nilptr"),
		S3AccessKey:   get("S3_ACCESS_KEY", ""),
		S3SecretKey:   get("S3_SECRET_KEY", ""),
		S3Region:      get("S3_REGION", "garage"),
	}

	if c.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if len(c.JWTSecret) < minSecretLen {
		errs = append(errs, fmt.Errorf("JWT_SECRET must be at least %d bytes", minSecretLen))
	}
	if len(c.IPHashSecret) < minSecretLen {
		errs = append(errs, fmt.Errorf("IP_HASH_SECRET must be at least %d bytes", minSecretLen))
	}
	if c.S3Endpoint == "" {
		errs = append(errs, errors.New("S3_ENDPOINT is required"))
	}
	if (c.AdminLogin == "") != (c.AdminPassword == "") {
		errs = append(errs, errors.New("ADMIN_LOGIN and ADMIN_PASSWORD must be set together"))
	}

	var err error
	if c.S3UseSSL, err = parseBool(get("S3_USE_SSL", "false")); err != nil {
		errs = append(errs, fmt.Errorf("S3_USE_SSL: %w", err))
	}
	if c.CookieSecure, err = parseBool(get("COOKIE_SECURE", "true")); err != nil {
		errs = append(errs, fmt.Errorf("COOKIE_SECURE: %w", err))
	}
	if c.MaxUploadBytes, err = strconv.ParseInt(get("MAX_UPLOAD_BYTES", "1073741824"), 10, 64); err != nil || c.MaxUploadBytes <= 0 {
		errs = append(errs, errors.New("MAX_UPLOAD_BYTES must be a positive integer"))
	}
	if c.TrustedProxies, err = ParsePrefixes(get("TRUSTED_PROXIES", "")); err != nil {
		errs = append(errs, fmt.Errorf("TRUSTED_PROXIES: %w", err))
	}

	return c, errors.Join(errs...)
}

// ParsePrefixes parses a comma separated list of CIDRs or bare addresses.
func ParsePrefixes(s string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for part := range strings.SplitSeq(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if !strings.Contains(part, "/") {
			addr, err := netip.ParseAddr(part)
			if err != nil {
				return nil, err
			}
			addr = addr.Unmap()
			out = append(out, netip.PrefixFrom(addr, addr.BitLen()))
			continue
		}
		p, err := netip.ParsePrefix(part)
		if err != nil {
			return nil, err
		}
		if p.Addr().Is4In6() {
			p = netip.PrefixFrom(p.Addr().Unmap(), max(p.Bits()-96, 0))
		}
		out = append(out, p.Masked())
	}
	return out, nil
}

func parseBool(s string) (bool, error) {
	return strconv.ParseBool(strings.ToLower(s))
}
