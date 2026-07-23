package source

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// ParsedProxy 解析出的节点
type ParsedProxy struct {
	Name   string
	Type   string
	Server string
	Port   int
	Raw    map[string]interface{}
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

// 常见分享链接 scheme（Clash / 客户端通用）
var shareURISchemes = []string{
	"vmess://", "vless://", "ss://", "ssr://", "trojan://",
	"hysteria://", "hysteria2://", "hy2://", "tuic://", "wireguard://",
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

func parseVmessURI(raw string) (map[string]interface{}, error) {
	payload := raw[len("vmess://"):]
	// 去掉可能的 fragment 前先 base64
	payload = strings.TrimSpace(payload)
	decoded, ok := tryBase64Decode([]byte(payload))
	if !ok {
		// 部分实现 payload 本身是 JSON 明文（少见）
		decoded = []byte(payload)
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(decoded, &obj); err != nil {
		return nil, fmt.Errorf("vmess json: %w", err)
	}
	// v2rayN 字段：v,ps,add,port,id,aid,scy,net,type,host,path,tls,sni,alpn
	name := firstNonEmpty(
		coerceString(obj["ps"]),
		coerceString(obj["remark"]),
		coerceString(obj["name"]),
	)
	server := firstNonEmpty(coerceString(obj["add"]), coerceString(obj["host"]), coerceString(obj["server"]))
	// host 在 vmess 里常作 sniffer 域名；server 优先 add
	if add := coerceString(obj["add"]); add != "" {
		server = add
	}
	port, ok := coercePort(obj["port"])
	if !ok {
		return nil, fmt.Errorf("vmess invalid port")
	}
	uuid := firstNonEmpty(coerceString(obj["id"]), coerceString(obj["uuid"]))
	if server == "" || uuid == "" {
		return nil, fmt.Errorf("vmess missing server/uuid")
	}
	if name == "" {
		name = fmt.Sprintf("vmess-%s-%d", server, port)
	}
	alterID := 0
	if v, ok := coercePort(obj["aid"]); ok {
		alterID = v
	} else if n, err := strconv.Atoi(coerceString(obj["aid"])); err == nil && n >= 0 {
		alterID = n
	}
	cipher := firstNonEmpty(coerceString(obj["scy"]), coerceString(obj["security"]), "auto")
	network := firstNonEmpty(coerceString(obj["net"]), coerceString(obj["network"]), "tcp")
	tls := strings.ToLower(coerceString(obj["tls"]))
	m := map[string]interface{}{
		"name":    name,
		"type":    "vmess",
		"server":  server,
		"port":    port,
		"uuid":    uuid,
		"alterId": alterID,
		"cipher":  cipher,
		"udp":     true,
	}
	if network != "" && network != "tcp" {
		m["network"] = network
	} else if network == "tcp" {
		m["network"] = "tcp"
	}
	if tls == "tls" || tls == "true" || tls == "1" {
		m["tls"] = true
	}
	sni := firstNonEmpty(coerceString(obj["sni"]), coerceString(obj["servername"]))
	if sni != "" {
		m["servername"] = sni
	}
	host := coerceString(obj["host"])
	path := coerceString(obj["path"])
	switch strings.ToLower(network) {
	case "ws":
		opts := map[string]interface{}{}
		if path != "" {
			opts["path"] = path
		}
		if host != "" {
			opts["headers"] = map[string]interface{}{"Host": host}
		}
		if len(opts) > 0 {
			m["ws-opts"] = opts
		}
	case "http":
		opts := map[string]interface{}{}
		if path != "" {
			opts["path"] = []interface{}{path}
		}
		if host != "" {
			opts["headers"] = map[string]interface{}{"Host": []interface{}{host}}
		}
		if len(opts) > 0 {
			m["http-opts"] = opts
		}
	case "grpc":
		opts := map[string]interface{}{}
		if path != "" {
			opts["grpc-service-name"] = path
		}
		if len(opts) > 0 {
			m["grpc-opts"] = opts
		}
	case "h2":
		opts := map[string]interface{}{}
		if path != "" {
			opts["path"] = path
		}
		if host != "" {
			opts["host"] = []interface{}{host}
		}
		if len(opts) > 0 {
			m["h2-opts"] = opts
		}
	}
	if alpn := coerceString(obj["alpn"]); alpn != "" {
		parts := splitComma(alpn)
		if len(parts) > 0 {
			m["alpn"] = parts
		}
	}
	return m, nil
}

func parseShadowsocksURI(raw string) (map[string]interface{}, error) {
	// SIP002: ss://BASE64(method:password)@host:port#name
	// 旧式: ss://BASE64(method:password@host:port)#name
	u := raw
	name := ""
	if i := strings.Index(u, "#"); i >= 0 {
		frag, err := url.QueryUnescape(u[i+1:])
		if err == nil {
			name = strings.TrimSpace(frag)
		} else {
			name = strings.TrimSpace(u[i+1:])
		}
		u = u[:i]
	}
	body := u
	if strings.HasPrefix(strings.ToLower(body), "ss://") {
		body = body[5:]
	}
	var method, password, host string
	var port int

	if strings.Contains(body, "@") {
		// SIP002
		at := strings.LastIndex(body, "@")
		userinfo := body[:at]
		hostport := body[at+1:]
		// userinfo 可能是 base64(method:password) 或明文 method:password
		decoded, ok := tryBase64Decode([]byte(userinfo))
		userDecoded := userinfo
		if ok {
			userDecoded = string(decoded)
		} else if unesc, err := url.QueryUnescape(userinfo); err == nil {
			userDecoded = unesc
		}
		method, password, ok = splitMethodPassword(userDecoded)
		if !ok {
			return nil, fmt.Errorf("ss invalid userinfo")
		}
		h, p, err := splitHostPort(hostport)
		if err != nil {
			return nil, err
		}
		host, port = h, p
	} else {
		decoded, ok := tryBase64Decode([]byte(body))
		if !ok {
			return nil, fmt.Errorf("ss invalid legacy body")
		}
		// method:password@host:port
		legacy := string(decoded)
		at := strings.LastIndex(legacy, "@")
		if at < 0 {
			return nil, fmt.Errorf("ss legacy missing @")
		}
		var ok2 bool
		method, password, ok2 = splitMethodPassword(legacy[:at])
		if !ok2 {
			return nil, fmt.Errorf("ss legacy userinfo")
		}
		h, p, err := splitHostPort(legacy[at+1:])
		if err != nil {
			return nil, err
		}
		host, port = h, p
	}
	if name == "" {
		name = fmt.Sprintf("ss-%s-%d", host, port)
	}
	return map[string]interface{}{
		"name":     name,
		"type":     "ss",
		"server":   host,
		"port":     port,
		"cipher":   method,
		"password": password,
		"udp":      true,
	}, nil
}

func parseTrojanURI(raw string) (map[string]interface{}, error) {
	// trojan://password@host:port?allowInsecure=1&peer=sni&sni=...#name
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	password, _ := url.QueryUnescape(u.User.String())
	if password == "" {
		password = u.User.Username()
	}
	host := u.Hostname()
	port := 443
	if p := u.Port(); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 65535 {
			return nil, fmt.Errorf("trojan invalid port")
		}
		port = n
	}
	if host == "" || password == "" {
		return nil, fmt.Errorf("trojan missing host/password")
	}
	name := strings.TrimSpace(u.Fragment)
	if name == "" {
		name = fmt.Sprintf("trojan-%s-%d", host, port)
	} else if unesc, err := url.QueryUnescape(name); err == nil {
		name = strings.TrimSpace(unesc)
	}
	q := u.Query()
	m := map[string]interface{}{
		"name":     name,
		"type":     "trojan",
		"server":   host,
		"port":     port,
		"password": password,
		"udp":      true,
	}
	sni := firstNonEmpty(q.Get("sni"), q.Get("peer"))
	if sni != "" {
		m["sni"] = sni
	}
	if network := q.Get("type"); network != "" && network != "tcp" {
		m["network"] = network
		if network == "ws" {
			opts := map[string]interface{}{}
			if path := q.Get("path"); path != "" {
				opts["path"] = path
			}
			if h := firstNonEmpty(q.Get("host"), q.Get("Host")); h != "" {
				opts["headers"] = map[string]interface{}{"Host": h}
			}
			if len(opts) > 0 {
				m["ws-opts"] = opts
			}
		}
	}
	if alpn := q.Get("alpn"); alpn != "" {
		if parts := splitComma(alpn); len(parts) > 0 {
			m["alpn"] = parts
		}
	}
	if insecure := q.Get("allowInsecure"); insecure == "1" || strings.EqualFold(insecure, "true") {
		m["skip-cert-verify"] = true
	}
	return m, nil
}

func parseVlessURI(raw string) (map[string]interface{}, error) {
	// vless://uuid@host:port?encryption=none&security=tls&type=ws&host=&path=&sni=#name
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	uuid := u.User.Username()
	host := u.Hostname()
	port := 443
	if p := u.Port(); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 65535 {
			return nil, fmt.Errorf("vless invalid port")
		}
		port = n
	}
	if host == "" || uuid == "" {
		return nil, fmt.Errorf("vless missing host/uuid")
	}
	name := strings.TrimSpace(u.Fragment)
	if name == "" {
		name = fmt.Sprintf("vless-%s-%d", host, port)
	} else if unesc, err := url.QueryUnescape(name); err == nil {
		name = strings.TrimSpace(unesc)
	}
	q := u.Query()
	m := map[string]interface{}{
		"name":   name,
		"type":   "vless",
		"server": host,
		"port":   port,
		"uuid":   uuid,
		"udp":    true,
	}
	if enc := q.Get("encryption"); enc != "" {
		m["cipher"] = enc // Clash Meta 多用 flow/packet-encoding；cipher 作兼容
	}
	security := strings.ToLower(q.Get("security"))
	if security == "tls" || security == "reality" {
		m["tls"] = true
	}
	if security == "reality" {
		m["reality-opts"] = map[string]interface{}{
			"public-key": q.Get("pbk"),
			"short-id":   q.Get("sid"),
		}
	}
	sni := firstNonEmpty(q.Get("sni"), q.Get("peer"))
	if sni != "" {
		m["servername"] = sni
	}
	if fp := q.Get("fp"); fp != "" {
		m["client-fingerprint"] = fp
	}
	if flow := q.Get("flow"); flow != "" {
		m["flow"] = flow
	}
	network := firstNonEmpty(q.Get("type"), "tcp")
	if network != "" {
		m["network"] = network
	}
	switch strings.ToLower(network) {
	case "ws":
		opts := map[string]interface{}{}
		if path := q.Get("path"); path != "" {
			opts["path"] = path
		}
		if h := q.Get("host"); h != "" {
			opts["headers"] = map[string]interface{}{"Host": h}
		}
		if len(opts) > 0 {
			m["ws-opts"] = opts
		}
	case "grpc":
		opts := map[string]interface{}{}
		if sn := firstNonEmpty(q.Get("serviceName"), q.Get("path")); sn != "" {
			opts["grpc-service-name"] = sn
		}
		if len(opts) > 0 {
			m["grpc-opts"] = opts
		}
	}
	if alpn := q.Get("alpn"); alpn != "" {
		if parts := splitComma(alpn); len(parts) > 0 {
			m["alpn"] = parts
		}
	}
	return m, nil
}

func parseHysteria2URI(raw string) (map[string]interface{}, error) {
	// hysteria2://password@host:port?sni=...&insecure=1#name
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	password, _ := url.QueryUnescape(u.User.String())
	if password == "" {
		password = u.User.Username()
	}
	host := u.Hostname()
	port := 443
	if p := u.Port(); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 65535 {
			return nil, fmt.Errorf("hysteria2 invalid port")
		}
		port = n
	}
	if host == "" || password == "" {
		return nil, fmt.Errorf("hysteria2 missing host/password")
	}
	name := strings.TrimSpace(u.Fragment)
	if name == "" {
		name = fmt.Sprintf("hy2-%s-%d", host, port)
	} else if unesc, err := url.QueryUnescape(name); err == nil {
		name = strings.TrimSpace(unesc)
	}
	q := u.Query()
	m := map[string]interface{}{
		"name":     name,
		"type":     "hysteria2",
		"server":   host,
		"port":     port,
		"password": password,
	}
	if sni := q.Get("sni"); sni != "" {
		m["sni"] = sni
	}
	if insecure := q.Get("insecure"); insecure == "1" || strings.EqualFold(insecure, "true") {
		m["skip-cert-verify"] = true
	}
	if obfs := q.Get("obfs"); obfs != "" {
		m["obfs"] = obfs
	}
	if obfsPass := q.Get("obfs-password"); obfsPass != "" {
		m["obfs-password"] = obfsPass
	}
	return m, nil
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
