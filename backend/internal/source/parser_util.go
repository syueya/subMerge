package source

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// parsePortOr443 解析 URL 端口，缺省 443；越界返回错误（proto 仅用于错误信息）。
func parsePortOr443(u *url.URL, proto string) (int, error) {
	p := u.Port()
	if p == "" {
		return 443, nil
	}
	n, err := strconv.Atoi(p)
	if err != nil || n < 1 || n > 65535 {
		return 0, fmt.Errorf("%s invalid port", proto)
	}
	return n, nil
}

// fragmentName 取 URL fragment 作节点名（URL 解码），为空时用 "<prefix>-<host>-<port>" 兜底。
func fragmentName(u *url.URL, prefix, host string, port int) string {
	name := strings.TrimSpace(u.Fragment)
	if name == "" {
		return fmt.Sprintf("%s-%s-%d", prefix, host, port)
	}
	if unesc, err := url.QueryUnescape(name); err == nil {
		return strings.TrimSpace(unesc)
	}
	return name
}

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
	host, portStr, err := net.SplitHostPort(hostport)
	if err != nil {
		return "", 0, fmt.Errorf("invalid host:port: %w", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return "", 0, fmt.Errorf("invalid port")
	}
	return host, port, nil
}

// asStringKeyMap 兼容 yaml 解析出的 map[string]interface{} / map[interface{}]interface{}。
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
