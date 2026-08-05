package source

import (
	"fmt"
	"strings"
)

func looksLikeURIList(b []byte) bool {
	s := strings.TrimSpace(string(b))
	if s == "" {
		return false
	}
	// 整段或首行是分享链接
	if hasShareURIPrefix(s) {
		return true
	}
	// 多行里有足够多的分享链接（避免把含偶然 URL 的 YAML 误判）
	n := 0
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if hasShareURIPrefix(line) {
			n++
			if n >= 1 && !looksLikeYAML(b) {
				return true
			}
		}
	}
	return n >= 2
}

func hasShareURIPrefix(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	for _, p := range shareURISchemes {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	return false
}

// parseURIListDetailed 从分享链接列表解析节点（每行一条；忽略空行与 # 注释）
func parseURIListDetailed(body []byte) ([]ParsedProxy, ParseStats, error) {
	stats := ParseStats{}
	lines := splitURILines(string(body))
	out := make([]ParsedProxy, 0, len(lines))
	seen := map[string]struct{}{}
	for _, line := range lines {
		stats.Total++
		m, err := parseShareURI(line)
		if err != nil {
			stats.addDrop("uri_parse")
			continue
		}
		p, reason := normalizeParsedProxy(m)
		if reason != "" {
			stats.addDrop(reason)
			continue
		}
		// 同名去重：追加序号，避免整批丢弃（URI 列表常缺唯一 name）
		base := p.Name
		if _, dup := seen[p.Name]; dup {
			for i := 2; ; i++ {
				cand := fmt.Sprintf("%s-%d", base, i)
				if _, ok := seen[cand]; !ok {
					p.Name = cand
					if p.Raw != nil {
						p.Raw["name"] = cand
					}
					break
				}
			}
		}
		seen[p.Name] = struct{}{}
		out = append(out, p)
	}
	stats.Valid = len(out)
	if len(out) == 0 {
		return nil, stats, fmt.Errorf("no valid share links found (total=%d dropped=%v)", stats.Total, stats.Dropped)
	}
	return out, stats, nil
}

func splitURILines(s string) []string {
	// 兼容 \r\n / 仅 \r；部分订阅用空白分隔多条
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	raw := strings.Split(s, "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// 一行内空格分隔的多条链接
		if strings.Contains(line, "://") && strings.ContainsAny(line, " \t") {
			for _, part := range strings.Fields(line) {
				part = strings.TrimSpace(part)
				if hasShareURIPrefix(part) {
					out = append(out, part)
				}
			}
			continue
		}
		if hasShareURIPrefix(line) {
			out = append(out, line)
		}
	}
	return out
}

// parseShareURI 将单条分享链接转为 Clash 风格 map（name/type/server/port + 协议字段）
func parseShareURI(raw string) (map[string]interface{}, error) {
	raw = strings.TrimSpace(raw)
	lower := strings.ToLower(raw)
	switch {
	case strings.HasPrefix(lower, "vmess://"):
		return parseVmessURI(raw)
	case strings.HasPrefix(lower, "ss://"):
		return parseShadowsocksURI(raw)
	case strings.HasPrefix(lower, "trojan://"):
		return parseTrojanURI(raw)
	case strings.HasPrefix(lower, "vless://"):
		return parseVlessURI(raw)
	case strings.HasPrefix(lower, "hysteria2://") || strings.HasPrefix(lower, "hy2://"):
		return parseHysteria2URI(raw)
	default:
		return nil, fmt.Errorf("unsupported share uri scheme")
	}
}
