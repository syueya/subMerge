package geo

import (
	"errors"
	"net"
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
	Input          string       `json:"input"`
	Host           string       `json:"host"`
	Kind           string       `json:"kind"` // domain | ipv4 | ipv6 | empty | invalid
	Matched        bool         `json:"matched"`
	FallbackMatch  bool         `json:"fallbackMatch"`
	Rule           *MatchRule   `json:"rule"`
	Skipped        int          `json:"skipped"`
	Note           string       `json:"note"`
	GeoHit         *MatchGeoHit `json:"geoHit,omitempty"`
	IPs            []string     `json:"ips,omitempty"`
	ResolveSkipped bool         `json:"resolveSkipped"`
	ResolveError   string       `json:"resolveError,omitempty"`
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
	switch kind {
	case "empty":
		result.Note = "请输入域名或 URL，例如 chat.openai.com 或 https://www.google.com"
		return result, nil
	case "invalid":
		result.Note = "无法解析输入，请检查格式"
		return result, nil
	}

	enabled, needSite, needIP := filterEnabledMatchRules(rules)

	s.mu.RLock()
	defer s.mu.RUnlock()

	siteByCat := map[string]GeoSiteHit{}
	if needSite && kind == "domain" {
		siteByCat = s.collectSiteHits(host)
	}
	ipByCat := map[string]IPHit{}
	var resolveErr string
	if needIP {
		ipByCat, resolveErr = s.collectIPHits(&result, host, kind, resolve)
	}

	skipped := 0
	for i := range enabled {
		rule := enabled[i]
		if rule.Type == "MATCH" {
			cp := rule
			result.Matched = false
			result.FallbackMatch = true
			result.Rule = &cp
			result.Skipped = skipped
			result.Note = "未命中其它规则，落入最终匹配 (MATCH)"
			return result, nil
		}
		if hit, note, geo := evalMatchRule(kind, host, rule, siteByCat, ipByCat, resolve, resolveErr); hit {
			return hitResult(result, rule, skipped, note, geo)
		}
		skipped++
	}

	result.Skipped = skipped
	result.Note = "没有启用规则命中（请确认是否存在 MATCH 兜底规则）"
	return result, nil
}

func filterEnabledMatchRules(rules []MatchRule) (enabled []MatchRule, needSite, needIP bool) {
	enabled = make([]MatchRule, 0, len(rules))
	for _, r := range rules {
		if r.Enabled != nil && !*r.Enabled {
			continue
		}
		r.Type = strings.ToUpper(strings.TrimSpace(r.Type))
		enabled = append(enabled, r)
		switch r.Type {
		case "GEOSITE":
			needSite = true
		case "GEOIP":
			needIP = true
		}
	}
	return enabled, needSite, needIP
}

func (s *Service) collectSiteHits(host string) map[string]GeoSiteHit {
	out := map[string]GeoSiteHit{}
	for _, entry := range s.snap.domains {
		if !domainMatch(entry, host) {
			continue
		}
		if _, ok := out[entry.Category]; ok {
			continue
		}
		out[entry.Category] = GeoSiteHit{Category: entry.Category, Type: entry.Type, Value: entry.Value}
	}
	return out
}

func (s *Service) collectIPHits(result *MatchResult, host, kind string, resolve bool) (map[string]IPHit, string) {
	out := map[string]IPHit{}
	addHits := func(ip net.IP) {
		qr := QueryResponse{IPs: []string{}, GeoIP: []IPHit{}}
		s.appendIPLookups(&qr, ip)
		if len(result.IPs) == 0 {
			result.IPs = qr.IPs
		} else {
			result.IPs = append(result.IPs, qr.IPs...)
		}
		for _, hit := range qr.GeoIP {
			if _, ok := out[hit.Category]; !ok {
				out[hit.Category] = hit
			}
		}
	}

	switch kind {
	case "ipv4", "ipv6":
		if ip := net.ParseIP(host); ip != nil {
			addHits(ip)
		}
		result.ResolveSkipped = true
		return out, ""
	case "domain":
		if !resolve {
			result.ResolveSkipped = true
			return out, ""
		}
		result.ResolveSkipped = false
		ips, err := net.LookupIP(host)
		if err != nil {
			msg := err.Error()
			result.ResolveError = msg
			return out, msg
		}
		seen := map[string]bool{}
		result.IPs = result.IPs[:0]
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
				if _, ok := out[hit.Category]; !ok {
					out[hit.Category] = hit
				}
			}
		}
		return out, ""
	default:
		return out, ""
	}
}

// evalMatchRule 返回是否命中、说明、可选 geo 细节。
func evalMatchRule(
	kind, host string,
	rule MatchRule,
	siteByCat map[string]GeoSiteHit,
	ipByCat map[string]IPHit,
	resolve bool,
	resolveErr string,
) (bool, string, *MatchGeoHit) {
	typ := rule.Type
	payload := strings.TrimSpace(rule.Payload)
	switch kind {
	case "domain":
		switch typ {
		case "DOMAIN":
			if matchDomainExact(host, payload) {
				return true, "命中域名精确匹配", nil
			}
		case "DOMAIN-SUFFIX":
			if matchDomainSuffix(host, payload) {
				return true, "命中域名后缀", nil
			}
		case "DOMAIN-KEYWORD":
			if matchDomainKeyword(host, payload) {
				return true, "命中域名关键词", nil
			}
		case "GEOSITE":
			cat := strings.ToLower(payload)
			if cat == "" {
				return false, "", nil
			}
			if hit, ok := siteByCat[cat]; ok {
				return true, "命中 GeoSite 分类", &MatchGeoHit{Category: hit.Category, Type: hit.Type, Value: hit.Value}
			}
		case "GEOIP":
			if !resolve || resolveErr != "" {
				return false, "", nil
			}
			cat := strings.ToLower(payload)
			if hit, ok := ipByCat[cat]; ok {
				return true, "命中 GeoIP 分类（经 DNS）", &MatchGeoHit{Category: hit.Category, CIDR: hit.CIDR, IP: hit.IP}
			}
		}
	case "ipv4":
		switch typ {
		case "IP-CIDR":
			if matchIPv4CIDR(host, payload) {
				return true, "命中 IPv4 CIDR", nil
			}
		case "GEOIP":
			cat := strings.ToLower(payload)
			if hit, ok := ipByCat[cat]; ok {
				return true, "命中 GeoIP 分类", &MatchGeoHit{Category: hit.Category, CIDR: hit.CIDR, IP: hit.IP}
			}
		}
	case "ipv6":
		switch typ {
		case "IP-CIDR6":
			if matchIPv6CIDRSimple(host, payload) {
				return true, "命中 IPv6 规则（简化匹配，正式以 Clash 为准）", nil
			}
		case "GEOIP":
			cat := strings.ToLower(payload)
			if hit, ok := ipByCat[cat]; ok {
				return true, "命中 GeoIP 分类", &MatchGeoHit{Category: hit.Category, CIDR: hit.CIDR, IP: hit.IP}
			}
		}
	}
	return false, "", nil
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
