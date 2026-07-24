package publish

import (
	"strings"
	"testing"

	"github.com/submerge/submerge/backend/internal/database"
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
		out := sanitizeProxiesForMeta([]map[string]interface{}{
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
	out := sanitizeProxiesForMeta([]map[string]interface{}{
		{"name": "a", "type": "vmess", "server": "1.1.1.1", "port": "443"},
		{"name": "b", "type": "ss", "server": "2.2.2.2", "port": 8443.0},
		{"name": "bad", "type": "vmess", "server": "3.3.3.3"}, // no port
	})
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
