package geo

import (
	"errors"
	"net"
	"strconv"
	"strings"
)

const maxMatchRules = 5000

// MatchRule is one Clash-style rule line for panel simulation.
type MatchRule struct {
	Type    string `json:"type"`
	Payload string `json:"payload,omitempty"`
	Target  string `json:"target"`
	Enabled *bool  `json:"enabled,omitempty"`
	Raw     string `json:"raw,omitempty"`
}

// MatchGeoHit is the geosite/geoip entry that caused a geo rule hit.
type MatchGeoHit struct {
	Category string `json:"category"`
	Type     string `json:"type,omitempty"`
	Value    string `json:"value,omitempty"`
	CIDR     string `json:"cidr,omitempty"`
	IP       string `json:"ip,omitempty"`
}

// MatchResult is the panel rule-test response shape.
type MatchResult struct {
	Input          string        `json:"input"`
	Host           string        `json:"host"`
	Kind           string        `json:"kind"` // domain | ipv4 | ipv6 | empty | invalid
	Matched        bool          `json:"matched"`
	FallbackMatch  bool          `json:"fallbackMatch"`
	Rule           *MatchRule    `json:"rule"`
	Skipped        int           `json:"skipped"`
	Note           string        `json:"note"`
	GeoHit         *MatchGeoHit  `json:"geoHit,omitempty"`
	IPs            []string      `json:"ips,omitempty"`
	ResolveSkipped bool          `json:"resolveSkipped"`
	ResolveError   string        `json:"resolveError,omitempty"`
}

// MatchRules simulates Clash top-down matching against the given rule snapshot.
// GEOSITE/GEOIP use the in-memory geosite.dat / geoip.dat (literal category names).
func (s *Service) MatchRules(rawInput string, rules []MatchRule, resolve bool) (MatchResult, error) {
	if len(rules) > maxMatchRules {
		return MatchResult{}, errors.New("too many rules (max 5000)")
	}

	host, kind := normalizeMatchHost(rawInput)
	result := MatchResult{
		Input:          rawInput,
		Host:           host,
		Kind:           kind,
		Rule:           nil,
		IPs:            []string{},
		ResolveSkipped: true,
	}
	if kind == "empty" {
		result.Note = "请输入域名或 URL，例如 chat.openai.com 或 https://www.google.com"
		return result, nil
	}
	if kind == "invalid" {
		result.Note = "无法解析输入，请检查格式"
		return result, nil
	}

	enabled := make([]MatchRule, 0, len(rules))
	needSite := false
	needIP := false
	for _, r := range rules {
		if r.Enabled != nil && !*r.Enabled {
			continue
		}
		typ := strings.ToUpper(strings.TrimSpace(r.Type))
		r.Type = typ
		enabled = append(enabled, r)
		switch typ {
		case "GEOSITE":
			needSite = true
		case "GEOIP":
			needIP = true
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	siteByCat := map[string]GeoSiteHit{}
	ipByCat := map[string]IPHit{}
	var resolveErr string

	if needSite && kind == "domain" {
		for _, entry := range s.snap.domains {
			if !domainMatch(entry, host) {
				continue
			}
			cat := entry.Category
			if _, ok := siteByCat[cat]; ok {
				continue
			}
			siteByCat[cat] = GeoSiteHit{Category: cat, Type: entry.Type, Value: entry.Value}
		}
	}

	if needIP {
		switch kind {
		case "ipv4", "ipv6":
			ip := net.ParseIP(host)
			if ip != nil {
				qr := QueryResponse{IPs: []string{}, GeoIP: []IPHit{}}
				s.appendIPLookups(&qr, ip)
				result.IPs = qr.IPs
				for _, hit := range qr.GeoIP {
					if _, ok := ipByCat[hit.Category]; !ok {
						ipByCat[hit.Category] = hit
					}
				}
			}
			result.ResolveSkipped = true
		case "domain":
			if !resolve {
				result.ResolveSkipped = true
			} else {
				result.ResolveSkipped = false
				ips, err := net.LookupIP(host)
				if err != nil {
					resolveErr = err.Error()
					result.ResolveError = resolveErr
				} else {
					seen := map[string]bool{}
					for _, ip := range ips {
						ipText := ip.String()
						if seen[ipText] {
							continue
						}
						seen[ipText] = true
						result.IPs = append(result.IPs, ipText)
						qr := QueryResponse{IPs: []string{}, GeoIP: []IPHit{}}
						s.appendIPLookups(&qr, ip)
						for _, hit := range qr.GeoIP {
							if _, ok := ipByCat[hit.Category]; !ok {
								ipByCat[hit.Category] = hit
							}
						}
					}
				}
			}
		}
	}

	skipped := 0
	for i := range enabled {
		rule := enabled[i]
		typ := rule.Type
		payload := strings.TrimSpace(rule.Payload)

		if typ == "MATCH" {
			cp := rule
			result.Matched = false
			result.FallbackMatch = true
			result.Rule = &cp
			result.Skipped = skipped
			result.Note = "未命中其它规则，落入最终匹配 (MATCH)"
			return result, nil
		}

		switch kind {
		case "domain":
			switch typ {
			case "DOMAIN":
				if matchDomainExact(host, payload) {
					return hitResult(result, rule, skipped, "命中域名精确匹配", nil)
				}
			case "DOMAIN-SUFFIX":
				if matchDomainSuffix(host, payload) {
					return hitResult(result, rule, skipped, "命中域名后缀", nil)
				}
			case "DOMAIN-KEYWORD":
				if matchDomainKeyword(host, payload) {
					return hitResult(result, rule, skipped, "命中域名关键词", nil)
				}
			case "GEOSITE":
				cat := strings.ToLower(payload)
				if cat == "" {
					skipped++
					continue
				}
				if hit, ok := siteByCat[cat]; ok {
					gh := &MatchGeoHit{Category: hit.Category, Type: hit.Type, Value: hit.Value}
					return hitResult(result, rule, skipped, "命中 GeoSite 分类", gh)
				}
				skipped++
				continue
			case "GEOIP":
				if !resolve {
					skipped++
					continue
				}
				if resolveErr != "" {
					skipped++
					continue
				}
				cat := strings.ToLower(payload)
				if hit, ok := ipByCat[cat]; ok {
					gh := &MatchGeoHit{Category: hit.Category, CIDR: hit.CIDR, IP: hit.IP}
					return hitResult(result, rule, skipped, "命中 GeoIP 分类（经 DNS）", gh)
				}
				skipped++
				continue
			case "IP-CIDR", "IP-CIDR6":
				skipped++
				continue
			default:
				skipped++
				continue
			}
		case "ipv4":
			switch typ {
			case "IP-CIDR":
				if matchIPv4CIDR(host, payload) {
					return hitResult(result, rule, skipped, "命中 IPv4 CIDR", nil)
				}
			case "GEOIP":
				cat := strings.ToLower(payload)
				if hit, ok := ipByCat[cat]; ok {
					gh := &MatchGeoHit{Category: hit.Category, CIDR: hit.CIDR, IP: hit.IP}
					return hitResult(result, rule, skipped, "命中 GeoIP 分类", gh)
				}
			case "GEOSITE", "DOMAIN", "DOMAIN-SUFFIX", "DOMAIN-KEYWORD", "IP-CIDR6":
				skipped++
				continue
			default:
				skipped++
				continue
			}
		case "ipv6":
			switch typ {
			case "IP-CIDR6":
				if matchIPv6CIDRSimple(host, payload) {
					return hitResult(result, rule, skipped, "命中 IPv6 规则（简化匹配，正式以 Clash 为准）", nil)
				}
			case "GEOIP":
				cat := strings.ToLower(payload)
				if hit, ok := ipByCat[cat]; ok {
					gh := &MatchGeoHit{Category: hit.Category, CIDR: hit.CIDR, IP: hit.IP}
					return hitResult(result, rule, skipped, "命中 GeoIP 分类", gh)
				}
			default:
				skipped++
				continue
			}
		}
		skipped++
	}

	result.Skipped = skipped
	result.Note = "没有启用规则命中（请确认是否存在 MATCH 兜底规则）"
	return result, nil
}

func hitResult(base MatchResult, rule MatchRule, skipped int, note string, geo *MatchGeoHit) (MatchResult, error) {
	cp := rule
	base.Matched = true
	base.FallbackMatch = false
	base.Rule = &cp
	base.Skipped = skipped
	base.Note = note
	base.GeoHit = geo
	return base, nil
}

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

func matchDomainExact(host, payload string) bool {
	p := strings.ToLower(strings.TrimSpace(strings.TrimSuffix(payload, ".")))
	return p != "" && host == p
}

func matchDomainSuffix(host, payload string) bool {
	p := strings.ToLower(strings.TrimSpace(payload))
	p = strings.TrimPrefix(p, ".")
	if p == "" {
		return false
	}
	return host == p || strings.HasSuffix(host, "."+p)
}

func matchDomainKeyword(host, payload string) bool {
	p := strings.ToLower(strings.TrimSpace(payload))
	return p != "" && strings.Contains(host, p)
}

func matchIPv4CIDR(ip, cidr string) bool {
	cidr = strings.TrimSpace(cidr)
	if cidr == "" {
		return false
	}
	if !strings.Contains(cidr, "/") {
		parsed := net.ParseIP(cidr)
		return parsed != nil && parsed.To4() != nil && parsed.String() == ip
	}
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	parsed := net.ParseIP(ip)
	if parsed == nil || parsed.To4() == nil {
		return false
	}
	return network.Contains(parsed)
}

func matchIPv6CIDRSimple(ip, payload string) bool {
	p := strings.ToLower(strings.TrimSpace(payload))
	if p == "" {
		return false
	}
	base := strings.Split(p, "/")[0]
	base = strings.TrimSuffix(base, ":")
	host := strings.ToLower(ip)
	return host == base || strings.HasPrefix(host, base)
}
