package source

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// 常见分享链接 scheme（Clash / 客户端通用）
var shareURISchemes = []string{
	"vmess://", "vless://", "ss://", "ssr://", "trojan://",
	"hysteria://", "hysteria2://", "hy2://", "tuic://", "wireguard://",
}

// ParsedProxy 解析出的节点
type ParsedProxy struct {
	Name   string
	Type   string
	Server string
	Port   int
	Raw    map[string]interface{}
}

// ParseStats 解析统计（帮助排查「为什么少了节点」）
type ParseStats struct {
	Total   int            `json:"total"`
	Valid   int            `json:"valid"`
	Dropped map[string]int `json:"dropped,omitempty"` // 原因 → 数量
}

func (s *ParseStats) addDrop(reason string) {
	if s.Dropped == nil {
		s.Dropped = map[string]int{}
	}
	s.Dropped[reason]++
}

// DecodeSubscriptionBody 规范化上游订阅正文。
// 机场常见：整份 Clash YAML 再 Base64 一次；也可能是明文 YAML 或分享链接列表。
func DecodeSubscriptionBody(body []byte) ([]byte, error) {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil, fmt.Errorf("empty subscription body")
	}
	// UTF-8 BOM
	body = bytes.TrimPrefix(body, []byte{0xEF, 0xBB, 0xBF})

	// 已是 YAML 或明文 URI 列表则直接用
	if looksLikeYAML(body) || looksLikeURIList(body) {
		return body, nil
	}

	// 尝试 Base64（标准 / URL / 无填充）。
	// tryBase64Decode 已保证解码结果为「基本可打印文本」，此处直接采用；
	// 后续 ParseClashProxiesDetailed 会再判定它是 YAML 还是 URI 列表。
	if decoded, ok := tryBase64Decode(body); ok {
		decoded = bytes.TrimSpace(decoded)
		decoded = bytes.TrimPrefix(decoded, []byte{0xEF, 0xBB, 0xBF})
		if len(decoded) > 0 {
			return decoded, nil
		}
	}

	// 原样返回，交给后续解析器给出明确错误
	return body, nil
}

func looksLikeYAML(b []byte) bool {
	s := strings.TrimLeftFunc(string(b), unicode.IsSpace)
	if s == "" {
		return false
	}
	// 注释或文档开始
	if strings.HasPrefix(s, "#") || strings.HasPrefix(s, "---") {
		return true
	}
	lower := strings.ToLower(s)
	for _, p := range []string{
		"proxies:", "proxy-groups:", "rules:", "port:", "mixed-port:",
		"socks-port:", "mode:", "allow-lan:", "log-level:", "dns:",
	} {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	// 任意一行以 proxies: 开头也算
	for _, line := range strings.Split(lower, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "proxies:") {
			return true
		}
	}
	return false
}

func tryBase64Decode(raw []byte) ([]byte, bool) {
	// 去掉空白/换行（部分订阅会折行）
	compact := make([]byte, 0, len(raw))
	for _, c := range raw {
		if c == ' ' || c == '\n' || c == '\r' || c == '\t' {
			continue
		}
		compact = append(compact, c)
	}
	if len(compact) < 8 {
		return nil, false
	}
	// 仅允许 base64 字符
	for _, c := range compact {
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') || c == '+' || c == '/' || c == '=' ||
			c == '-' || c == '_' {
			continue
		}
		return nil, false
	}

	// 补齐 padding
	pad := len(compact) % 4
	if pad > 0 {
		compact = append(compact, bytes.Repeat([]byte("="), 4-pad)...)
	}

	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.RawURLEncoding,
	}
	for _, enc := range encodings {
		out, err := enc.DecodeString(string(compact))
		if err != nil || len(out) == 0 {
			continue
		}
		// 解码结果应主要是可打印文本
		if !mostlyPrintable(out) {
			continue
		}
		return out, true
	}
	return nil, false
}

func mostlyPrintable(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	bad := 0
	for _, c := range b {
		// 允许可打印字符与常见空白（\t=0x09 \n=0x0a \r=0x0d）；
		// 其余控制字符（0x00–0x08、0x0b、0x0c、0x0e–0x1f、0x7f）计为坏字节
		if c == '\t' || c == '\n' || c == '\r' {
			continue
		}
		if c < 0x20 || c == 0x7f {
			bad++
		}
	}
	return bad*20 < len(b) // 允许少量二进制噪声
}

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

// ParseClashProxies 从 Clash YAML 提取 proxies（自动处理 Base64 订阅）
func ParseClashProxies(yamlBody []byte) ([]ParsedProxy, error) {
	list, _, err := ParseClashProxiesDetailed(yamlBody)
	return list, err
}

// ParseClashProxiesDetailed 解析并返回丢弃统计。
// 支持：Clash YAML、Base64(YAML)、分享链接列表（明文或 Base64）。
func ParseClashProxiesDetailed(yamlBody []byte) ([]ParsedProxy, ParseStats, error) {
	stats := ParseStats{}
	body, err := DecodeSubscriptionBody(yamlBody)
	if err != nil {
		return nil, stats, err
	}

	// 优先识别 URI 列表（含 Base64 解码后的 vmess:// / ss:// 等）
	if looksLikeURIList(body) {
		return parseURIListDetailed(body)
	}

	var doc map[string]interface{}
	if err := yaml.Unmarshal(body, &doc); err != nil {
		// 再试一次：有些正文前后有杂讯，取第一段像 YAML 的内容
		if cleaned := extractYAMLBlock(body); cleaned != nil {
			if err2 := yaml.Unmarshal(cleaned, &doc); err2 == nil {
				goto parsed
			}
		}
		// YAML 失败时再尝试 URI 列表（部分订阅夹杂空行/说明文字）
		if list, st, uriErr := parseURIListDetailed(body); uriErr == nil && len(list) > 0 {
			return list, st, nil
		}
		return nil, stats, fmt.Errorf("invalid yaml (若机场为 Base64 订阅已自动尝试解码): %w", err)
	}
parsed:
	rawProxies, ok := doc["proxies"]
	if !ok {
		// 无 proxies 时尝试 URI 列表（避免把分享链接误报成缺字段）
		if list, st, uriErr := parseURIListDetailed(body); uriErr == nil && len(list) > 0 {
			return list, st, nil
		}
		return nil, stats, fmt.Errorf("yaml missing proxies field（解码后仍无 proxies，请确认是 Clash 订阅或分享链接列表）")
	}
	list, ok := rawProxies.([]interface{})
	if !ok {
		return nil, stats, fmt.Errorf("proxies is not a list")
	}

	stats.Total = len(list)
	out := make([]ParsedProxy, 0, len(list))
	seen := map[string]struct{}{}
	for _, item := range list {
		m, ok := asStringKeyMap(item)
		if !ok {
			stats.addDrop("not_object")
			continue
		}
		p, reason := normalizeParsedProxy(m)
		if reason != "" {
			stats.addDrop(reason)
			continue
		}
		if _, dup := seen[p.Name]; dup {
			stats.addDrop("duplicate_name")
			continue
		}
		seen[p.Name] = struct{}{}
		out = append(out, p)
	}
	stats.Valid = len(out)
	if len(out) == 0 {
		return nil, stats, fmt.Errorf("no valid proxies found (total=%d dropped=%v)", stats.Total, stats.Dropped)
	}
	return out, stats, nil
}

func normalizeParsedProxy(m map[string]interface{}) (ParsedProxy, string) {
	name := coerceString(m["name"])
	typ := coerceString(m["type"])
	server := coerceString(m["server"])
	if name == "" {
		return ParsedProxy{}, "missing_name"
	}
	if typ == "" {
		return ParsedProxy{}, "missing_type"
	}
	if server == "" {
		return ParsedProxy{}, "missing_server"
	}
	port, portOK := coercePort(m["port"])
	if !portOK {
		return ParsedProxy{}, "invalid_port"
	}
	m["name"] = name
	m["type"] = typ
	m["server"] = server
	m["port"] = port
	rawJSON, _ := json.Marshal(m)
	var raw map[string]interface{}
	_ = json.Unmarshal(rawJSON, &raw)
	return ParsedProxy{
		Name:   name,
		Type:   typ,
		Server: server,
		Port:   port,
		Raw:    raw,
	}, ""
}

// extractYAMLBlock 从可能夹杂前后缀的文本中截取 YAML 主体
func extractYAMLBlock(b []byte) []byte {
	s := string(b)
	idx := strings.Index(strings.ToLower(s), "proxies:")
	if idx < 0 {
		return nil
	}
	// 向前找到行首
	start := idx
	for start > 0 && s[start-1] != '\n' {
		start--
	}
	return []byte(strings.TrimSpace(s[start:]))
}
