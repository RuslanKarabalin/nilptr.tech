package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"
)

// ClientIP returns the address of the client that made the request.
//
// If the direct peer is not in trusted, the peer address is returned.
// Otherwise X-Forwarded-For is walked from the right, skipping trusted
// addresses, and the first untrusted address is returned. If every hop is
// trusted, or a malformed entry is met, the last trusted address seen is
// returned.
func ClientIP(r *http.Request, trusted []netip.Prefix) netip.Addr {
	peer := parseAddr(r.RemoteAddr)
	if !peer.IsValid() {
		return netip.IPv4Unspecified()
	}
	if !isTrusted(peer, trusted) {
		return peer
	}
	var hops []string
	for _, v := range r.Header.Values("X-Forwarded-For") {
		hops = append(hops, strings.Split(v, ",")...)
	}
	for i := len(hops) - 1; i >= 0; i-- {
		a := parseAddr(strings.TrimSpace(hops[i]))
		if !a.IsValid() {
			break
		}
		if !isTrusted(a, trusted) {
			return a
		}
		peer = a
	}
	return peer
}

// parseAddr accepts "ip", "ip:port" and "[ipv6]:port".
func parseAddr(s string) netip.Addr {
	if s == "" {
		return netip.Addr{}
	}
	if a, err := netip.ParseAddr(s); err == nil {
		return a.Unmap().WithZone("")
	}
	if host, _, err := net.SplitHostPort(s); err == nil {
		if a, err := netip.ParseAddr(host); err == nil {
			return a.Unmap().WithZone("")
		}
	}
	return netip.Addr{}
}

func isTrusted(a netip.Addr, trusted []netip.Prefix) bool {
	for _, p := range trusted {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

// IPHash returns hex(HMAC-SHA256(secret, ip + "|" + YYYY-MM-DD)) where the
// date is the UTC day of now. It enforces one comment per IP per day.
func IPHash(secret []byte, ip netip.Addr, now time.Time) string {
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(ip.String() + "|" + now.UTC().Format(time.DateOnly)))
	return hex.EncodeToString(m.Sum(nil))
}
