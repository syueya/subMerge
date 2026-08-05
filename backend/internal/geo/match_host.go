package geo

import (
	"net"
	"strconv"
	"strings"
)

func normalizeMatchHost(raw string) (host string, kind string) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", "empty"
	}

	// URL without scheme but with path
	if !hasURLScheme(s) && strings.Contains(s, "/") {
		s = "http://" + s
	}

	if hasURLScheme(s) {
		// Minimal URL host extract without importing net/url failure modes for weird inputs
		rest := s
		if i := strings.Index(rest, "://"); i >= 0 {
			rest = rest[i+3:]
		}
		if i := strings.IndexAny(rest, "/?#"); i >= 0 {
			rest = rest[:i]
		}
		// userinfo@host
		if i := strings.LastIndex(rest, "@"); i >= 0 {
			rest = rest[i+1:]
		}
		s = rest
	}

	s = strings.Split(s, "/")[0]
	s = strings.Split(s, "?")[0]
	s = strings.Split(s, "#")[0]

	if strings.HasPrefix(s, "[") {
		end := strings.Index(s, "]")
		if end > 0 {
			s = s[1:end]
		}
	} else if isIPv4WithPort(s) {
		if i := strings.LastIndex(s, ":"); i > 0 {
			s = s[:i]
		}
	} else if looksLikeHostPort(s) {
		if i := strings.LastIndex(s, ":"); i > 0 {
			s = s[:i]
		}
	}

	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.TrimSuffix(s, ".")
	if s == "" {
		return "", "empty"
	}

	if ip := net.ParseIP(s); ip != nil {
		if ip.To4() != nil {
			return ip.String(), "ipv4"
		}
		return ip.String(), "ipv6"
	}

	// Basic domain check (allow localhost)
	if s != "localhost" && !isLikelyDomain(s) {
		if !containsAlnum(s) {
			return s, "invalid"
		}
	}
	return s, "domain"
}

func hasURLScheme(s string) bool {
	i := strings.Index(s, "://")
	if i <= 0 {
		return false
	}
	for _, c := range s[:i] {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '+' || c == '-' || c == '.' {
			continue
		}
		return false
	}
	return true
}

func isIPv4WithPort(s string) bool {
	// 1.2.3.4:443
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return false
	}
	if net.ParseIP(parts[0]) == nil || net.ParseIP(parts[0]).To4() == nil {
		return false
	}
	_, err := strconv.Atoi(parts[1])
	return err == nil
}

func looksLikeHostPort(s string) bool {
	// host:port without IPv6 brackets
	if strings.Count(s, ":") != 1 {
		return false
	}
	i := strings.LastIndex(s, ":")
	if i <= 0 {
		return false
	}
	port := s[i+1:]
	if port == "" {
		return false
	}
	for _, c := range port {
		if c < '0' || c > '9' {
			return false
		}
	}
	host := s[:i]
	return host != "" && !strings.Contains(host, ":")
}

func isLikelyDomain(s string) bool {
	if len(s) == 0 || len(s) > maxDomainLength {
		return false
	}
	// label.label…
	for _, label := range strings.Split(s, ".") {
		if label == "" || len(label) > 63 {
			return false
		}
		for i, c := range label {
			ok := (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-'
			if !ok {
				return false
			}
			if (i == 0 || i == len(label)-1) && c == '-' {
				return false
			}
		}
	}
	return true
}

func containsAlnum(s string) bool {
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			return true
		}
	}
	return false
}
