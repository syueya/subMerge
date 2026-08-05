package geo

import (
	"net"
	"strings"
)

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
