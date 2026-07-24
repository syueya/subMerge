package publish

import (
	"fmt"
	"strings"
)

// sanitizeProxiesForMeta 清洗节点 map，保证 mihomo 可解析。
// 第二返回值：因非法 REALITY short-id 丢弃的节点数。
func sanitizeProxiesForMeta(in []map[string]interface{}) ([]map[string]interface{}, int) {
	out := make([]map[string]interface{}, 0, len(in))
	droppedReality := 0
	for _, raw := range in {
		if raw == nil {
			continue
		}
		m := make(map[string]interface{}, len(raw)+4)
		for k, v := range raw {
			m[k] = v
		}
		name := strings.TrimSpace(fmt.Sprint(m["name"]))
		typ := strings.TrimSpace(strings.ToLower(fmt.Sprint(m["type"])))
		server := strings.TrimSpace(fmt.Sprint(m["server"]))
		if name == "" || typ == "" || typ == "<nil>" || server == "" || server == "<nil>" {
			continue
		}
		m["name"] = name
		m["type"] = typ
		m["server"] = server
		if port, ok := coerceYAMLPort(m["port"]); ok {
			m["port"] = port
		} else {
			continue
		}
		// 常见布尔字段规范化
		for _, bk := range []string{"udp", "tls", "skip-cert-verify", "tfo", "mptcp"} {
			if v, exists := m[bk]; exists {
				m[bk] = coerceYAMLBool(v)
			}
		}
		// 旧式 ws-path / ws-headers 与 ws-opts 并存时，Meta 以 ws-opts 为准；去掉重复字段避免解析歧义
		normalizeTransportOpts(m)
		// REALITY：short-id 必须当字符串输出（防 6314e825 → .inf），并校验 hex
		if !normalizeRealityOpts(m) {
			droppedReality++
			continue
		}
		// 剥离内部元数据，避免写入 Clash YAML
		stripInternalProxyMeta(m)
		out = append(out, m)
	}
	return out, droppedReality
}

// normalizeRealityOpts 规范化 reality-opts；非法 short-id 返回 false（调用方应丢弃节点）
func normalizeRealityOpts(m map[string]interface{}) bool {
	raw, ok := m["reality-opts"]
	if !ok || raw == nil {
		return true
	}
	opts, ok := raw.(map[string]interface{})
	if !ok {
		// 兼容 json 反序列化后偶发的 map[string]string
		if ms, ok := raw.(map[string]string); ok {
			opts = make(map[string]interface{}, len(ms))
			for k, v := range ms {
				opts[k] = v
			}
		} else {
			delete(m, "reality-opts")
			return true
		}
	}
	out := make(map[string]interface{}, len(opts)+2)
	for k, v := range opts {
		out[k] = v
	}

	if v, exists := out["public-key"]; exists && v != nil {
		pk := strings.TrimSpace(fmt.Sprint(v))
		if pk == "" || pk == "<nil>" {
			delete(out, "public-key")
		} else {
			// 强制引号，避免 base64 中的特殊字符/歧义
			out["public-key"] = yamlQuotedString(pk)
		}
	}

	if v, exists := out["short-id"]; exists && v != nil {
		sid := normalizeRealityShortID(v)
		if sid == "" && fmt.Sprint(v) != "" && fmt.Sprint(v) != "<nil>" {
			// 有值但非法（含 .inf / 非 hex / 过长）
			return false
		}
		// 空 short-id 合法（部分节点允许）；仍强制字符串形态
		out["short-id"] = yamlQuotedString(sid)
	}

	if len(out) == 0 {
		delete(m, "reality-opts")
		return true
	}
	m["reality-opts"] = out
	return true
}

// annotateProxyIndexes 在 proxies 列表每项前加「# N」注释（1-based，对齐 mihomo proxy N 日志）
func annotateProxyIndexes(yamlText string) string {
	lines := strings.Split(yamlText, "\n")
	out := make([]string, 0, len(lines)+32)
	inProxies := false
	idx := 0
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		// 进入 proxies:
		if !inProxies {
			if trimmed == "proxies:" {
				inProxies = true
				out = append(out, line)
				continue
			}
			out = append(out, line)
			continue
		}
		// 离开 proxies：下一顶层 key（无缩进且非空、非列表项）
		if trimmed != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && !strings.HasPrefix(trimmed, "#") {
			inProxies = false
			out = append(out, line)
			continue
		}
		// 列表项起始：两个空格 + "- "（yaml.v3 默认缩进）
		if strings.HasPrefix(line, "  - ") || strings.HasPrefix(line, "    - ") {
			idx++
			// 注释缩进与列表项对齐
			indent := "  "
			if strings.HasPrefix(line, "    - ") {
				indent = "    "
			}
			out = append(out, fmt.Sprintf("%s# %d", indent, idx))
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// normalizeRealityShortID 将 short-id 规范为 hex 字符串；非法返回 "" 且调用方应区分「空」与「非法」
func normalizeRealityShortID(v interface{}) string {
	// 若已被 YAML 误解析成 float/inf，无法还原原始 hex，只能判非法
	switch t := v.(type) {
	case float64, float32:
		return ""
	case yamlQuotedString:
		return normalizeRealityShortID(string(t))
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return ""
		}
		// 客户端已炸成 .inf / +Inf 等
		low := strings.ToLower(s)
		if low == ".inf" || low == "-.inf" || low == "+.inf" || low == "inf" || low == "+inf" || low == "-inf" || low == ".nan" || low == "nan" {
			return ""
		}
		if !realityShortIDRe.MatchString(s) {
			return ""
		}
		// mihomo hex.Decode：奇数长度会失败
		if len(s)%2 != 0 {
			return ""
		}
		return strings.ToLower(s)
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" || s == "<nil>" {
			return ""
		}
		return normalizeRealityShortID(s)
	}
}

// normalizeTransportOpts 统一传输层字段：保留 *-opts，去掉并行的旧字段
func normalizeTransportOpts(m map[string]interface{}) {
	if _, has := m["ws-opts"]; has {
		delete(m, "ws-path")
		delete(m, "ws-headers")
	} else if path, ok := m["ws-path"]; ok || m["ws-headers"] != nil {
		// 仅有旧字段时提升为 ws-opts
		opts := map[string]interface{}{}
		if ok && path != nil && fmt.Sprint(path) != "" && fmt.Sprint(path) != "<nil>" {
			opts["path"] = path
		}
		if h, ok := m["ws-headers"]; ok && h != nil {
			opts["headers"] = h
		}
		if len(opts) > 0 {
			m["ws-opts"] = opts
		}
		delete(m, "ws-path")
		delete(m, "ws-headers")
	}
	if _, has := m["http-opts"]; has {
		delete(m, "http-path")
		delete(m, "http-headers")
	}
	if _, has := m["h2-opts"]; has {
		delete(m, "h2-path")
		delete(m, "h2-headers")
	}
	// hysteria2：ports / mport 并存时保留 ports（Meta 主字段），去掉重复 mport
	if strings.EqualFold(fmt.Sprint(m["type"]), "hysteria2") {
		if _, hasPorts := m["ports"]; hasPorts {
			delete(m, "mport")
		} else if mp, ok := m["mport"]; ok && mp != nil {
			m["ports"] = mp
			delete(m, "mport")
		}
	}
}

func coerceYAMLPort(v interface{}) (int, bool) {
	switch t := v.(type) {
	case int:
		if t >= 1 && t <= 65535 {
			return t, true
		}
	case int64:
		if t >= 1 && t <= 65535 {
			return int(t), true
		}
	case float64:
		if t == float64(int(t)) {
			p := int(t)
			if p >= 1 && p <= 65535 {
				return p, true
			}
		}
	case string:
		s := strings.TrimSpace(t)
		n := 0
		for _, c := range s {
			if c < '0' || c > '9' {
				return 0, false
			}
			n = n*10 + int(c-'0')
			if n > 65535 {
				return 0, false
			}
		}
		if n >= 1 {
			return n, true
		}
	}
	return 0, false
}

func coerceYAMLBool(v interface{}) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		return s == "true" || s == "1" || s == "yes" || s == "on"
	case int:
		return t != 0
	case float64:
		return t != 0
	default:
		return false
	}
}

// stripInternalProxyMeta 去掉仅供生成器使用的内部字段
func stripInternalProxyMeta(m map[string]interface{}) {
	if m == nil {
		return
	}
	delete(m, "_source_id")
	delete(m, "_source_name")
}
