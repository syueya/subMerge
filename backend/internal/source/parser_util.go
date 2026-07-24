package source

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func splitComma(s string) []interface{} {
	parts := strings.Split(s, ",")
	out := make([]interface{}, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func splitMethodPassword(s string) (method, password string, ok bool) {
	s = strings.TrimSpace(s)
	i := strings.Index(s, ":")
	if i <= 0 || i >= len(s)-1 {
		return "", "", false
	}
	return s[:i], s[i+1:], true
}

func splitHostPort(hostport string) (string, int, error) {
	hostport = strings.TrimSpace(hostport)
	// IPv6 in brackets
	host, portStr, err := net.SplitHostPort(hostport)
	if err != nil {
		// 无端口？
		return "", 0, fmt.Errorf("invalid host:port: %w", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return "", 0, fmt.Errorf("invalid port")
	}
	return host, port, nil
}

// asStringKeyMap 兼容 yaml 解析出的 map[string]interface{} / map[interface{}]interface{}
func asStringKeyMap(v interface{}) (map[string]interface{}, bool) {
	switch m := v.(type) {
	case map[string]interface{}:
		return m, true
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(m))
		for k, val := range m {
			out[fmt.Sprint(k)] = val
		}
		return out, true
	default:
		return nil, false
	}
}

func coerceString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case fmt.Stringer:
		return strings.TrimSpace(t.String())
	case int:
		return fmt.Sprintf("%d", t)
	case int64:
		return fmt.Sprintf("%d", t)
	case float64:
		// YAML 数字；端口等不走这里
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	case bool:
		if t {
			return "true"
		}
		return "false"
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	}
}

func coercePort(v interface{}) (int, bool) {
	switch t := v.(type) {
	case int:
		if t >= 1 && t <= 65535 {
			return t, true
		}
	case int64:
		if t >= 1 && t <= 65535 {
			return int(t), true
		}
	case uint64:
		if t >= 1 && t <= 65535 {
			return int(t), true
		}
	case float64:
		// yaml.v3 常把数字解成 float64
		if t == float64(int(t)) {
			p := int(t)
			if p >= 1 && p <= 65535 {
				return p, true
			}
		}
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return 0, false
		}
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

// EnsureRegionPrefix 保证节点名带目标地区前缀
// 若已有其它地区码前缀（1–16 位 alnum + '-'）则剥掉再加目标前缀
func EnsureRegionPrefix(name, region string) string {
	region = strings.ToUpper(strings.TrimSpace(region))
	name = strings.TrimSpace(name)
	if region == "" {
		return name
	}
	prefix := region + "-"
	if strings.HasPrefix(strings.ToUpper(name), strings.ToUpper(prefix)) {
		return name
	}
	rest := stripRegionPrefix(name)
	if rest == "" {
		rest = name
	}
	return prefix + rest
}

// SanitizeSourceSuffix 将订阅源名称整理为节点名后缀（如「良心云」）
// 去掉路径分隔与控制字符；空白压成单个连字符；过长截断
func SanitizeSourceSuffix(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(name))
	prevDash := false
	for _, r := range name {
		switch {
		case r < 0x20 || r == 0x7f:
			continue
		case r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' ||
			r == '"' || r == '<' || r == '>' || r == '|' || r == '#' || r == '%' || r == '@':
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		default:
			b.WriteRune(r)
			prevDash = false
		}
	}
	out := strings.Trim(b.String(), "-")
	// 限制后缀长度，避免节点名过长
	runes := []rune(out)
	if len(runes) > 24 {
		out = string(runes[:24])
		out = strings.TrimRight(out, "-")
	}
	return out
}

// FormatProxyName 生成入库节点名：地区前缀 + 原名 + 源后缀
// 例：US-香港01-良心云
func FormatProxyName(rawName, region, sourceName string) string {
	base := EnsureRegionPrefix(rawName, region)
	suffix := SanitizeSourceSuffix(sourceName)
	if suffix == "" {
		return base
	}
	// 若已以 -源名 结尾（改地区/改名重写时），先去掉再拼，避免重复后缀
	base = stripSourceSuffix(base, sourceName)
	if base == "" {
		base = EnsureRegionPrefix(rawName, region)
	}
	return base + "-" + suffix
}

// stripRegionPrefix 去掉开头的已知地区码前缀（JP-/HK-…），避免误剥 Japan- 等整词
func stripRegionPrefix(name string) string {
	ensureRegionDict()
	i := strings.Index(name, "-")
	if i < 2 || i > 16 {
		return name
	}
	code := strings.ToUpper(name[:i])
	if !isRegionCode(code) {
		return name
	}
	if _, ok := knownRegionCodes[code]; !ok {
		return name
	}
	rest := strings.TrimSpace(name[i+1:])
	if rest == "" {
		return name
	}
	return rest
}
