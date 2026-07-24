package publish

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/submerge/submerge/backend/internal/database"
	"gopkg.in/yaml.v3"
)

func TestGeneratorBuild(t *testing.T) {
	g := NewGenerator()
	res, err := g.Build(BuildInput{
		Proxies: []map[string]interface{}{
			{"name": "US-a", "type": "ss", "server": "1.1.1.1", "port": 443},
			{"name": "JP-b", "type": "ss", "server": "2.2.2.2", "port": 443},
		},
		Groups: []database.ProxyGroup{
			{Name: "直连", Type: "select", Proxies: `["DIRECT"]`, Enabled: true},
			{Name: "拒绝", Type: "select", Proxies: `["REJECT"]`, Enabled: true},
			{Name: "US", Type: "select", Proxies: `["REGION:US"]`, Enabled: true},
			{Name: "JP", Type: "select", Proxies: `["REGION:JP"]`, Enabled: true},
		},
		Rules: []database.Rule{
			{Type: "DOMAIN-SUFFIX", Payload: "openai.com", Target: "US", Enabled: true},
			{Type: "GEOIP", Payload: "CN", Target: "直连", Enabled: true},
			{Type: "MATCH", Payload: "", Target: "直连", Enabled: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ProxyCount != 2 || res.RuleCount != 3 {
		t.Fatalf("counts: %+v", res)
	}
	if !strings.Contains(res.YAML, "MATCH,直连") {
		t.Fatalf("missing MATCH,直连: %s", res.YAML)
	}
	if !strings.Contains(res.YAML, "JP-b") {
		t.Fatalf("missing JP node in yaml: %s", res.YAML)
	}
	// 订阅配置核心字段（不强制 mixed-port/dns，避免覆盖客户端本机设置）
	for _, key := range []string{"mode: rule", "proxies:", "proxy-groups:", "rules:"} {
		if !strings.Contains(res.YAML, key) {
			t.Fatalf("missing field %q in yaml:\n%s", key, res.YAML)
		}
	}
	// 不应写入本机端口/控制器，避免 Clash Verge 导入后测速异常
	for _, key := range []string{"mixed-port:", "external-controller:", "dns:"} {
		if strings.Contains(res.YAML, key) {
			t.Fatalf("subscription yaml should not contain %q:\n%s", key, res.YAML)
		}
	}
}

func TestBuildHashStableAcrossRuns(t *testing.T) {
	g := NewGenerator()
	in := BuildInput{
		Proxies: []map[string]interface{}{
			{"name": "US-a", "type": "ss", "server": "1.1.1.1", "port": 443},
			{"name": "JP-b", "type": "ss", "server": "2.2.2.2", "port": 443},
			{"name": "DE-c", "type": "ss", "server": "3.3.3.3", "port": 443},
			{"name": "FR-d", "type": "ss", "server": "4.4.4.4", "port": 443},
			{"name": "CA-e", "type": "ss", "server": "5.5.5.5", "port": 443},
		},
		Groups: []database.ProxyGroup{
			{Name: "直连", Type: "select", Proxies: `["DIRECT"]`},
			{Name: "美国US", Type: "url-test", Proxies: `["REGION:US"]`, URL: "https://www.gstatic.com/generate_204"},
			{Name: "其他国家", Type: "url-test", Proxies: `["REGION:OTHER"]`, URL: "https://www.gstatic.com/generate_204"},
		},
		Rules: []database.Rule{
			{Type: "MATCH", Target: "直连", Enabled: true},
		},
	}
	first, err := g.Build(in)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 30; i++ {
		res, err := g.Build(in)
		if err != nil {
			t.Fatal(err)
		}
		if res.Hash != first.Hash {
			t.Fatalf("hash unstable at iter %d: first=%s got=%s", i, first.Hash, res.Hash)
		}
		if res.YAML != first.YAML {
			t.Fatalf("yaml unstable at iter %d", i)
		}
	}
}

func TestSanitizeDropsLegacyWSFields(t *testing.T) {
	out, dropped := sanitizeProxiesForMeta([]map[string]interface{}{
		{
			"name":       "a",
			"type":       "vmess",
			"server":     "1.1.1.1",
			"port":       443,
			"ws-path":    "/",
			"ws-headers": map[string]interface{}{"Host": "example.com"},
			"ws-opts":    map[string]interface{}{"path": "/", "headers": map[string]interface{}{"Host": "example.com"}},
		},
	})
	if dropped != 0 {
		t.Fatalf("dropped=%d", dropped)
	}
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
	if _, ok := out[0]["ws-path"]; ok {
		t.Fatal("ws-path should be removed when ws-opts present")
	}
	if _, ok := out[0]["ws-headers"]; ok {
		t.Fatal("ws-headers should be removed when ws-opts present")
	}
	if _, ok := out[0]["ws-opts"]; !ok {
		t.Fatal("ws-opts should remain")
	}
}

func TestSanitizeProxiesPortTypes(t *testing.T) {
	out, dropped := sanitizeProxiesForMeta([]map[string]interface{}{
		{"name": "a", "type": "vmess", "server": "1.1.1.1", "port": "443"},
		{"name": "b", "type": "ss", "server": "2.2.2.2", "port": 8443.0},
		{"name": "bad", "type": "vmess", "server": "3.3.3.3"}, // no port
	})
	if dropped != 0 {
		t.Fatalf("dropped=%d", dropped)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2, got %d", len(out))
	}
	if out[0]["port"] != 443 {
		t.Fatalf("port0=%v", out[0]["port"])
	}
	if out[1]["port"] != 8443 {
		t.Fatalf("port1=%v", out[1]["port"])
	}
}

func TestRealityShortIDQuotedNotScientific(t *testing.T) {
	// 6314e825 未加引号会被 YAML 1.1 解析成 .inf → mihomo invalid REALITY short ID
	g := NewGenerator()
	res, err := g.Build(BuildInput{
		Proxies: []map[string]interface{}{
			{
				"name":               "JP-reality",
				"type":               "vless",
				"server":             "1.2.3.4",
				"port":               443,
				"uuid":               "98bb17d9-9815-4923-a1d7-3d017ffd3f08",
				"tls":                true,
				"client-fingerprint": "chrome",
				"reality-opts": map[string]interface{}{
					"public-key": "VOFSjjWT0wIH3Q0ntyEZd8WwksrIAb5gPt_3PBnEASg",
					"short-id":   "6314e825",
				},
			},
			{"name": "US-a", "type": "ss", "server": "1.1.1.1", "port": 443},
		},
		Groups: []database.ProxyGroup{
			{Name: "直连", Type: "select", Proxies: `["DIRECT"]`},
			{Name: "节点选择", Type: "select", Proxies: `["ALL"]`},
		},
		Rules:     []database.Rule{{Type: "MATCH", Target: "直连", Enabled: true}},
		GroupMode: "auto",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(res.YAML, "short-id: 6314e825") && !strings.Contains(res.YAML, `short-id: "6314e825"`) {
		t.Fatalf("short-id must be double-quoted, got:\n%s", res.YAML)
	}
	if !strings.Contains(res.YAML, `short-id: "6314e825"`) {
		t.Fatalf("expected quoted short-id:\n%s", res.YAML)
	}
	if strings.Contains(res.YAML, ".inf") || strings.Contains(res.YAML, ".Inf") {
		t.Fatalf("yaml must not contain .inf:\n%s", res.YAML)
	}
	// name 应在 type 之前；并有 1-based 序号注释
	jp := strings.Index(res.YAML, "name: JP-reality")
	tp := strings.Index(res.YAML, "type: vless")
	if jp < 0 || tp < 0 || jp > tp {
		t.Fatalf("expected name before type in proxy block:\n%s", res.YAML)
	}
	if !strings.Contains(res.YAML, "# 1") {
		t.Fatalf("expected proxy index comments:\n%s", res.YAML)
	}
	// 再 round-trip：客户端用 YAML 解析后 short-id 仍是字符串 6314e825
	var doc map[string]interface{}
	if err := yaml.Unmarshal([]byte(res.YAML), &doc); err != nil {
		t.Fatal(err)
	}
	list, _ := doc["proxies"].([]interface{})
	found := false
	for _, item := range list {
		m, _ := item.(map[string]interface{})
		if fmt.Sprint(m["name"]) != "JP-reality" {
			continue
		}
		found = true
		ro, _ := m["reality-opts"].(map[string]interface{})
		sid := fmt.Sprint(ro["short-id"])
		if sid != "6314e825" {
			t.Fatalf("round-trip short-id=%q type=%T want 6314e825", ro["short-id"], ro["short-id"])
		}
	}
	if !found {
		t.Fatal("JP-reality missing after unmarshal")
	}
}

func TestSanitizeDropsInvalidRealityShortID(t *testing.T) {
	out, dropped := sanitizeProxiesForMeta([]map[string]interface{}{
		{
			"name":   "bad-inf",
			"type":   "vless",
			"server": "1.1.1.1",
			"port":   443,
			"reality-opts": map[string]interface{}{
				"public-key": "VOFSjjWT0wIH3Q0ntyEZd8WwksrIAb5gPt_3PBnEASg",
				"short-id":   ".inf",
			},
		},
		{
			"name":   "bad-float",
			"type":   "vless",
			"server": "1.1.1.1",
			"port":   443,
			"reality-opts": map[string]interface{}{
				"public-key": "VOFSjjWT0wIH3Q0ntyEZd8WwksrIAb5gPt_3PBnEASg",
				// YAML 1.1 把 6314e825 解析成 +Inf 后的形态
				"short-id": math.Inf(1),
			},
		},
		{
			"name":   "ok",
			"type":   "vless",
			"server": "1.1.1.1",
			"port":   443,
			"reality-opts": map[string]interface{}{
				"public-key": "VOFSjjWT0wIH3Q0ntyEZd8WwksrIAb5gPt_3PBnEASg",
				"short-id":   "9c5b8c53",
			},
		},
	})
	if dropped != 2 {
		t.Fatalf("dropped=%d want 2", dropped)
	}
	if len(out) != 1 || fmt.Sprint(out[0]["name"]) != "ok" {
		t.Fatalf("out=%v", out)
	}
}

func TestGeneratorRejectsUnknownRuleTarget(t *testing.T) {
	g := NewGenerator()
	_, err := g.Build(BuildInput{
		Proxies: []map[string]interface{}{{"name": "US-a", "type": "ss", "server": "1.1.1.1", "port": 443}},
		Groups:  []database.ProxyGroup{{Name: "US", Type: "select", Proxies: `["REGION:US"]`}},
		Rules:   []database.Rule{{Type: "MATCH", Target: "MISSING", Enabled: true}},
	})
	if err == nil {
		t.Fatal("expected unknown rule target to fail")
	}
}

func TestGeneratorAllModeKeepsEmptyGroup(t *testing.T) {
	g := NewGenerator()
	// all：空 JP 保留为 DIRECT
	res, err := g.Build(BuildInput{
		Proxies: []map[string]interface{}{{"name": "US-a", "type": "ss", "server": "1.1.1.1", "port": 443}},
		Groups: []database.ProxyGroup{
			{Name: "直连", Type: "select", Proxies: `["DIRECT"]`},
			{Name: "US", Type: "select", Proxies: `["REGION:US"]`},
			{Name: "JP", Type: "select", Proxies: `["REGION:JP"]`},
		},
		Rules: []database.Rule{
			{Type: "DOMAIN-SUFFIX", Payload: "example.jp", Target: "JP", Enabled: true},
			{Type: "MATCH", Target: "直连", Enabled: true},
		},
		GroupMode: "all",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.YAML, "name: JP") {
		t.Fatalf("expected JP group kept in all mode:\n%s", res.YAML)
	}
	if !strings.Contains(res.YAML, "DOMAIN-SUFFIX,example.jp,JP") {
		t.Fatalf("expected rule keep target JP:\n%s", res.YAML)
	}
}

func TestGeneratorAutoPrunesEmptyGroup(t *testing.T) {
	g := NewGenerator()
	// auto：仅 US 节点时剪掉 JP；有「节点选择」时规则目标优先回退到它
	res, err := g.Build(BuildInput{
		Proxies: []map[string]interface{}{{"name": "US-a", "type": "ss", "server": "1.1.1.1", "port": 443}},
		Groups: []database.ProxyGroup{
			{Name: "直连", Type: "select", Proxies: `["DIRECT"]`},
			{Name: "节点选择", Type: "select", Proxies: `["ALL"]`},
			{Name: "美国US", Type: "select", Proxies: `["REGION:US"]`},
			{Name: "日本JP", Type: "select", Proxies: `["REGION:JP"]`},
			{Name: "菲律宾PH", Type: "select", Proxies: `["REGION:PH"]`},
		},
		Rules: []database.Rule{
			{Type: "DOMAIN-SUFFIX", Payload: "example.jp", Target: "日本JP", Enabled: true},
			{Type: "MATCH", Target: "直连", Enabled: true},
		},
		GroupMode: "auto",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(res.YAML, "name: 日本JP") || strings.Contains(res.YAML, "name: 菲律宾PH") {
		t.Fatalf("auto should prune empty region groups:\n%s", res.YAML)
	}
	if !strings.Contains(res.YAML, "name: 美国US") {
		t.Fatalf("auto should keep US group:\n%s", res.YAML)
	}
	if !strings.Contains(res.YAML, "name: 节点选择") {
		t.Fatalf("auto should keep 节点选择:\n%s", res.YAML)
	}
	if !strings.Contains(res.YAML, "DOMAIN-SUFFIX,example.jp,节点选择") {
		t.Fatalf("rule targeting pruned group should fallback 节点选择:\n%s", res.YAML)
	}
	found := false
	for _, w := range res.Warnings {
		if strings.Contains(w, "日本JP") && strings.Contains(w, "节点选择") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected fallback warning, got %v", res.Warnings)
	}
}

func TestGeneratorAutoFallbackDirectWithoutSelectGroup(t *testing.T) {
	g := NewGenerator()
	// 无「节点选择」时仍回退 DIRECT
	res, err := g.Build(BuildInput{
		Proxies: []map[string]interface{}{{"name": "US-a", "type": "ss", "server": "1.1.1.1", "port": 443}},
		Groups: []database.ProxyGroup{
			{Name: "直连", Type: "select", Proxies: `["DIRECT"]`},
			{Name: "美国US", Type: "select", Proxies: `["REGION:US"]`},
			{Name: "日本JP", Type: "select", Proxies: `["REGION:JP"]`},
		},
		Rules: []database.Rule{
			{Type: "DOMAIN-SUFFIX", Payload: "example.jp", Target: "日本JP", Enabled: true},
			{Type: "MATCH", Target: "直连", Enabled: true},
		},
		GroupMode: "auto",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.YAML, "DOMAIN-SUFFIX,example.jp,DIRECT") {
		t.Fatalf("without 节点选择 should fallback DIRECT:\n%s", res.YAML)
	}
}

func TestGeneratorCustomWhitelist(t *testing.T) {
	g := NewGenerator()
	res, err := g.Build(BuildInput{
		Proxies: []map[string]interface{}{
			{"name": "US-a", "type": "ss", "server": "1.1.1.1", "port": 443},
			{"name": "PH-b", "type": "ss", "server": "2.2.2.2", "port": 443},
		},
		Groups: []database.ProxyGroup{
			{Name: "直连", Type: "select", Proxies: `["DIRECT"]`},
			{Name: "美国US", Type: "select", Proxies: `["REGION:US"]`},
			{Name: "菲律宾PH", Type: "select", Proxies: `["REGION:PH"]`},
		},
		Rules: []database.Rule{
			{Type: "MATCH", Target: "直连", Enabled: true},
		},
		GroupMode:     "custom",
		AllowedGroups: []string{"直连", "菲律宾PH"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(res.YAML, "name: 美国US") {
		t.Fatalf("custom should not include US:\n%s", res.YAML)
	}
	if !strings.Contains(res.YAML, "name: 菲律宾PH") {
		t.Fatalf("custom should include PH:\n%s", res.YAML)
	}
}

func TestGeneratorSkipsEmptyRegionGroup(t *testing.T) {
	g := NewGenerator()
	res, err := g.Build(BuildInput{
		Proxies: []map[string]interface{}{{"name": "US-a", "type": "ss", "server": "1.1.1.1", "port": 443}},
		Groups: []database.ProxyGroup{
			{Name: "PH", Type: "select", Proxies: `["REGION:PH"]`},
			{Name: "US", Type: "select", Proxies: `["REGION:US", "DIRECT"]`},
		},
		Rules: []database.Rule{{Type: "MATCH", Target: "US", Enabled: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range res.GroupNames {
		if n == "PH" {
			t.Fatalf("empty PH group should be skipped, got groups=%v", res.GroupNames)
		}
	}
	if len(res.GroupNames) != 1 || res.GroupNames[0] != "US" {
		t.Fatalf("unexpected groups: %v", res.GroupNames)
	}
}

func TestGeneratorRejectsAllEmptyGroups(t *testing.T) {
	g := NewGenerator()
	_, err := g.Build(BuildInput{
		Proxies: []map[string]interface{}{{"name": "US-a", "type": "ss", "server": "1.1.1.1", "port": 443}},
		Groups:  []database.ProxyGroup{{Name: "PH", Type: "select", Proxies: `["REGION:PH"]`}},
		Rules:   []database.Rule{{Type: "MATCH", Target: "PH", Enabled: true}},
	})
	if err == nil {
		t.Fatal("expected error when all groups empty")
	}
}

func TestGeneratorRegionOther(t *testing.T) {
	g := NewGenerator()
	res, err := g.Build(BuildInput{
		Proxies: []map[string]interface{}{
			{"name": "US-a", "type": "ss", "server": "1.1.1.1", "port": 443},
			{"name": "DE-b", "type": "ss", "server": "2.2.2.2", "port": 443},
			{"name": "NG-c", "type": "ss", "server": "3.3.3.3", "port": 443},
		},
		Groups: []database.ProxyGroup{
			{Name: "美国US", Type: "select", Proxies: `["REGION:US"]`, Enabled: true},
			{Name: "其他国家", Type: "select", Proxies: `["REGION:OTHER"]`, Enabled: true},
		},
		Rules: []database.Rule{{Type: "MATCH", Target: "其他国家", Enabled: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.YAML, "DE-b") || !strings.Contains(res.YAML, "NG-c") {
		t.Fatalf("OTHER should include non-primary nodes:\n%s", res.YAML)
	}
	// 其他国家组应含 DE/NG，不应只剩空
	if !strings.Contains(res.YAML, "其他国家") {
		t.Fatalf("missing 其他国家 group:\n%s", res.YAML)
	}
}

func TestGeneratorSourceByName(t *testing.T) {
	g := NewGenerator()
	res, err := g.Build(BuildInput{
		Proxies: []map[string]interface{}{
			{"name": "US-a-良心云", "type": "ss", "server": "1.1.1.1", "port": 443, "_source_id": uint(1), "_source_name": "良心云"},
			{"name": "JP-b-良心云", "type": "ss", "server": "2.2.2.2", "port": 443, "_source_id": uint(1), "_source_name": "良心云"},
			{"name": "US-c-机场B", "type": "ss", "server": "3.3.3.3", "port": 443, "_source_id": uint(2), "_source_name": "机场B"},
		},
		Groups: []database.ProxyGroup{
			{Name: "直连", Type: "select", Proxies: `["DIRECT"]`, Enabled: true},
			{Name: "良心云", Type: "select", Proxies: `["SOURCE:良心云"]`, Enabled: true},
			{Name: "机场B", Type: "url-test", Proxies: `["SOURCE:机场B"]`, URL: "https://www.gstatic.com/generate_204", Enabled: true},
		},
		Rules: []database.Rule{{Type: "MATCH", Target: "良心云", Enabled: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	// 源组应含对应节点
	if !strings.Contains(res.YAML, "US-a-良心云") || !strings.Contains(res.YAML, "JP-b-良心云") {
		t.Fatalf("SOURCE:良心云 should expand nodes:\n%s", res.YAML)
	}
	if !strings.Contains(res.YAML, "US-c-机场B") {
		t.Fatalf("SOURCE:机场B should expand nodes:\n%s", res.YAML)
	}
	// 内部字段不得泄漏到 YAML
	if strings.Contains(res.YAML, "_source_id") || strings.Contains(res.YAML, "_source_name") {
		t.Fatalf("internal source meta leaked into yaml:\n%s", res.YAML)
	}
	// 告警应列出可用源
	joined := strings.Join(res.Warnings, "\n")
	if !strings.Contains(joined, "available sources:") {
		t.Fatalf("expected available sources warning, got: %v", res.Warnings)
	}
}

func TestGeneratorSourceByID(t *testing.T) {
	g := NewGenerator()
	res, err := g.Build(BuildInput{
		Proxies: []map[string]interface{}{
			{"name": "US-a-old", "type": "ss", "server": "1.1.1.1", "port": 443, "_source_id": uint(3), "_source_name": "旧名"},
			{"name": "JP-b-other", "type": "ss", "server": "2.2.2.2", "port": 443, "_source_id": uint(9), "_source_name": "其它"},
		},
		Groups: []database.ProxyGroup{
			{Name: "按ID", Type: "select", Proxies: `["SOURCE:id:3"]`, Enabled: true},
			{Name: "直连", Type: "select", Proxies: `["DIRECT"]`, Enabled: true},
		},
		Rules: []database.Rule{{Type: "MATCH", Target: "按ID", Enabled: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.YAML, "US-a-old") {
		t.Fatalf("SOURCE:id:3 should include US-a-old:\n%s", res.YAML)
	}
	// 不应把其它源节点放进「按ID」组：解析 proxy-groups 粗查
	// 组内列表在 YAML 中连续出现；至少保证 JP-b-other 存在于 proxies 但 MATCH 目标组名正确
	if !strings.Contains(res.YAML, "JP-b-other") {
		t.Fatalf("other source node should still be in proxies list:\n%s", res.YAML)
	}
}

func TestGeneratorSourceCaseInsensitiveName(t *testing.T) {
	g := NewGenerator()
	res, err := g.Build(BuildInput{
		Proxies: []map[string]interface{}{
			{"name": "US-a", "type": "ss", "server": "1.1.1.1", "port": 443, "_source_id": 1, "_source_name": "FooBar"},
		},
		Groups: []database.ProxyGroup{
			{Name: "源组", Type: "select", Proxies: `["SOURCE:foobar"]`, Enabled: true},
		},
		Rules: []database.Rule{{Type: "MATCH", Target: "源组", Enabled: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.YAML, "US-a") {
		t.Fatalf("SOURCE name match should be case-insensitive:\n%s", res.YAML)
	}
}

func TestGeneratorSourceEmptyPruned(t *testing.T) {
	g := NewGenerator()
	// 只有源 A 的节点；SOURCE:B 应被 auto 剪掉
	res, err := g.Build(BuildInput{
		Proxies: []map[string]interface{}{
			{"name": "US-a", "type": "ss", "server": "1.1.1.1", "port": 443, "_source_id": uint(1), "_source_name": "A"},
		},
		Groups: []database.ProxyGroup{
			{Name: "A", Type: "select", Proxies: `["SOURCE:A"]`, Enabled: true},
			{Name: "B", Type: "select", Proxies: `["SOURCE:B"]`, Enabled: true},
		},
		Rules: []database.Rule{{Type: "MATCH", Target: "A", Enabled: true}},
		// 显式 auto：允许规则回退；此处 MATCH 指向 A，B 被剪
		GroupMode: "auto",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(res.YAML, "name: B") {
		t.Fatalf("empty SOURCE:B group should be pruned:\n%s", res.YAML)
	}
	if !strings.Contains(res.YAML, "name: A") {
		t.Fatalf("SOURCE:A group should remain:\n%s", res.YAML)
	}
}

// formatRule 只输出 Clash 三元组，业务 category 不得进入 YAML
func TestFormatRuleOmitsCategory(t *testing.T) {
	g := NewGenerator()
	res, err := g.Build(BuildInput{
		Proxies: []map[string]interface{}{
			{"name": "US-a", "type": "ss", "server": "1.1.1.1", "port": 443},
		},
		Groups: []database.ProxyGroup{
			{Name: "直连", Type: "select", Proxies: `["DIRECT"]`, Enabled: true},
			{Name: "美国US", Type: "select", Proxies: `["REGION:US"]`, Enabled: true},
		},
		Rules: []database.Rule{
			{
				Type:     "DOMAIN-SUFFIX",
				Payload:  "openai.com",
				Target:   "美国US",
				Enabled:  true,
				Note:     "仅后台",
				Category: "海外AI",
			},
			{Type: "MATCH", Target: "直连", Enabled: true, Category: "兜底"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.YAML, "DOMAIN-SUFFIX,openai.com,美国US") {
		t.Fatalf("expected TYPE,payload,target without category:\n%s", res.YAML)
	}
	if !strings.Contains(res.YAML, "MATCH,直连") {
		t.Fatalf("expected MATCH,target:\n%s", res.YAML)
	}
	for _, bad := range []string{"海外AI", "兜底", "仅后台", "category"} {
		// rules 段不应出现业务分类/备注；proxies 里也不会有这些中文
		if strings.Contains(res.YAML, bad) {
			t.Fatalf("yaml must not contain %q:\n%s", bad, res.YAML)
		}
	}
	// 确认没有多出第 4 段
	if strings.Contains(res.YAML, "openai.com,美国US,") {
		t.Fatalf("rule line must not have extra fields after target:\n%s", res.YAML)
	}
}
