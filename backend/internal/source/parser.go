package source

import (
	"encoding/json"
	"fmt"

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
