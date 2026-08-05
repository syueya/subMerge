package source

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestParseClashProxies(t *testing.T) {
	yamlBody := []byte(`
proxies:
  - name: node-a
    type: ss
    server: 1.2.3.4
    port: 8388
    cipher: aes-256-gcm
    password: secret
  - name: node-a
    type: ss
    server: 1.2.3.4
    port: 8388
  - name: node-b
    type: vmess
    server: example.com
    port: 443
`)
	list, err := ParseClashProxies(yamlBody)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 unique proxies, got %d", len(list))
	}
	if list[0].Name != "node-a" || list[1].Name != "node-b" {
		t.Fatalf("unexpected names: %+v", list)
	}
}

func TestParseClashProxiesSkipsInvalidPorts(t *testing.T) {
	list, err := ParseClashProxies([]byte(`
proxies:
  - name: invalid
    type: ss
    server: example.com
    port: 70000
  - name: valid
    type: ss
    server: example.com
    port: 443
`))
	if err != nil || len(list) != 1 || list[0].Name != "valid" {
		t.Fatalf("unexpected parsed proxies: %v, %v", list, err)
	}
}

func TestParseClashProxiesStringPortAndStats(t *testing.T) {
	list, stats, err := ParseClashProxiesDetailed([]byte(`
proxies:
  - name: 剩余流量：1GB
    type: vmess
  - name: ok-string-port
    type: vmess
    server: a.example.com
    port: "443"
  - name: ok-float-port
    type: anytls
    server: b.example.com
    port: 8443.0
  - name: no-server
    type: vmess
    port: 443
`))
	if err != nil {
		t.Fatal(err)
	}
	if stats.Total != 4 {
		t.Fatalf("total=%d", stats.Total)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 valid, got %d: %+v dropped=%v", len(list), list, stats.Dropped)
	}
	if stats.Dropped["missing_server"] < 1 || stats.Dropped["missing_type"]+stats.Dropped["missing_server"] < 1 {
		// first item missing type+server; third missing server only depending on coerce
		if stats.Dropped["missing_server"] < 1 {
			t.Fatalf("expected missing_server drops, got %v", stats.Dropped)
		}
	}
	if list[0].Port != 443 || list[1].Port != 8443 {
		t.Fatalf("ports: %+v", list)
	}
}

func TestEnsureRegionPrefix(t *testing.T) {
	cases := map[string]struct {
		name, region, want string
	}{
		"plain":     {"foo", "US", "US-foo"},
		"same":      {"US-bar", "US", "US-bar"},
		"replace":   {"PH-baz", "US", "US-baz"},
		"emoji":     {"🇯🇵日本01", "JP", "JP-🇯🇵日本01"},
		"keep-flag": {"HK-🇭🇰香港01", "HK", "HK-🇭🇰香港01"},
	}
	for label, c := range cases {
		if got := EnsureRegionPrefix(c.name, c.region); got != c.want {
			t.Fatalf("%s: got %q want %q", label, got, c.want)
		}
	}
}

func TestParseClashProxiesBase64(t *testing.T) {
	plain := []byte(`
proxies:
  - name: node-a
    type: ss
    server: 1.2.3.4
    port: 8388
    cipher: aes-256-gcm
    password: secret
`)
	// 标准 Base64（模拟机场订阅）
	enc := make([]byte, base64.StdEncoding.EncodedLen(len(plain)))
	base64.StdEncoding.Encode(enc, plain)
	list, err := ParseClashProxies(enc)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "node-a" {
		t.Fatalf("unexpected: %+v", list)
	}
}

func TestDecodeSubscriptionBodyPlainYAML(t *testing.T) {
	in := []byte("proxies:\n  - name: a\n    type: ss\n    server: 1.1.1.1\n    port: 443\n")
	out, err := DecodeSubscriptionBody(in)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out, []byte("proxies:")) {
		t.Fatalf("got %s", out)
	}
}

func TestFormatProxyNameWithSourceSuffix(t *testing.T) {
	if got := FormatProxyName("香港01", "US", "良心云"); got != "US-香港01-良心云" {
		t.Fatalf("got %s", got)
	}
	// 已有同后缀不重复追加
	if got := FormatProxyName("US-香港01-良心云", "US", "良心云"); got != "US-香港01-良心云" {
		t.Fatalf("got %s", got)
	}
	if got := SanitizeSourceSuffix("  良心 云  "); got != "良心-云" {
		t.Fatalf("sanitize got %s", got)
	}
}

func TestParseURIListVmessPlain(t *testing.T) {
	// v2rayN vmess JSON → base64
	vmessJSON := `{"v":"2","ps":"香港01","add":"hk.example.com","port":"443","id":"11111111-1111-1111-1111-111111111111","aid":"0","scy":"auto","net":"ws","type":"none","host":"hk.example.com","path":"/v","tls":"tls","sni":"hk.example.com"}`
	enc := base64.StdEncoding.EncodeToString([]byte(vmessJSON))
	body := []byte("vmess://" + enc + "\n")
	list, stats, err := ParseClashProxiesDetailed(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 proxy, got %d stats=%+v", len(list), stats)
	}
	p := list[0]
	if p.Name != "香港01" || p.Type != "vmess" || p.Server != "hk.example.com" || p.Port != 443 {
		t.Fatalf("unexpected proxy: %+v raw=%v", p, p.Raw)
	}
	if p.Raw["uuid"] != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("uuid missing: %v", p.Raw)
	}
	if p.Raw["tls"] != true {
		t.Fatalf("tls expected true: %v", p.Raw["tls"])
	}
}

func TestParseURIListMixedAndBase64(t *testing.T) {
	// ss SIP002
	// method:password = aes-256-gcm:secret → base64
	user := base64.StdEncoding.EncodeToString([]byte("aes-256-gcm:secret"))
	ss := "ss://" + user + "@1.2.3.4:8388#%E7%BE%8E%E5%9B%BD01"
	trojan := "trojan://pass@trojan.example.com:443?sni=trojan.example.com&allowInsecure=1#Trojan-Node"
	vless := "vless://22222222-2222-2222-2222-222222222222@vless.example.com:443?encryption=none&security=tls&type=ws&host=vless.example.com&path=%2Fws&sni=vless.example.com#VLESS-WS"
	plain := stringsJoinLines(ss, trojan, vless)

	// 明文多行
	list, err := ParseClashProxies([]byte(plain))
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3, got %d: %+v", len(list), namesOf(list))
	}
	byType := map[string]ParsedProxy{}
	for _, p := range list {
		byType[p.Type] = p
	}
	if byType["ss"].Server != "1.2.3.4" || byType["ss"].Port != 8388 || byType["ss"].Name != "美国01" {
		t.Fatalf("ss: %+v", byType["ss"])
	}
	if byType["trojan"].Server != "trojan.example.com" || byType["trojan"].Raw["password"] != "pass" {
		t.Fatalf("trojan: %+v", byType["trojan"])
	}
	if byType["vless"].Server != "vless.example.com" || byType["vless"].Raw["uuid"] == nil {
		t.Fatalf("vless: %+v", byType["vless"])
	}

	// 整包 Base64（机场常见）
	enc := make([]byte, base64.StdEncoding.EncodedLen(len(plain)))
	base64.StdEncoding.Encode(enc, []byte(plain))
	list2, err := ParseClashProxies(enc)
	if err != nil {
		t.Fatal(err)
	}
	if len(list2) != 3 {
		t.Fatalf("base64 uri list expected 3, got %d", len(list2))
	}
}

func TestParseURIListDoesNotBreakYAML(t *testing.T) {
	list, err := ParseClashProxies([]byte(`
proxies:
  - name: node-a
    type: ss
    server: 1.2.3.4
    port: 8388
    cipher: aes-256-gcm
    password: secret
`))
	if err != nil || len(list) != 1 || list[0].Name != "node-a" {
		t.Fatalf("yaml still works: %v %+v", err, list)
	}
}

func stringsJoinLines(lines ...string) string {
	return stringsJoin(lines, "\n")
}

func stringsJoin(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	n := len(sep) * (len(parts) - 1)
	for _, p := range parts {
		n += len(p)
	}
	b := make([]byte, 0, n)
	for i, p := range parts {
		if i > 0 {
			b = append(b, sep...)
		}
		b = append(b, p...)
	}
	return string(b)
}

func namesOf(list []ParsedProxy) []string {
	out := make([]string, len(list))
	for i, p := range list {
		out[i] = p.Name
	}
	return out
}
