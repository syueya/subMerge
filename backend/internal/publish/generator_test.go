package publish

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/submerge/submerge/backend/internal/database"
	"gopkg.in/yaml.v3"
)

func ssProxy(name, server string, port int, extra ...map[string]interface{}) map[string]interface{} {
	p := map[string]interface{}{"name": name, "type": "ss", "server": server, "port": port}
	for _, m := range extra {
		for k, v := range m {
			p[k] = v
		}
	}
	return p
}

func group(name, typ, proxies string, extra ...func(*database.ProxyGroup)) database.ProxyGroup {
	g := database.ProxyGroup{Name: name, Type: typ, Proxies: proxies, Enabled: true}
	for _, fn := range extra {
		fn(&g)
	}
	return g
}

func withURL(url string) func(*database.ProxyGroup) {
	return func(g *database.ProxyGroup) { g.URL = url }
}

func makeRule(typ, payload, target string) database.Rule {
	return database.Rule{Type: typ, Payload: payload, Target: target, Enabled: true}
}

func mustBuild(t *testing.T, in BuildInput) *BuildResult {
	t.Helper()
	res, err := NewGenerator().Build(in)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestGeneratorBuild(t *testing.T) {
	res := mustBuild(t, BuildInput{
		Proxies: []map[string]interface{}{
			ssProxy("US-a", "1.1.1.1", 443),
			ssProxy("JP-b", "2.2.2.2", 443),
		},
		Groups: []database.ProxyGroup{
			group("直连", "select", `["DIRECT"]`),
			group("拒绝", "select", `["REJECT"]`),
			group("US", "select", `["REGION:US"]`),
			group("JP", "select", `["REGION:JP"]`),
		},
		Rules: []database.Rule{
			makeRule("DOMAIN-SUFFIX", "openai.com", "US"),
			makeRule("GEOIP", "CN", "直连"),
			makeRule("MATCH", "", "直连"),
		},
	})
	if res.ProxyCount != 2 || res.RuleCount != 3 {
		t.Fatalf("counts: %+v", res)
	}
	for _, want := range []string{"MATCH,直连", "JP-b", "mode: rule", "proxies:", "proxy-groups:", "rules:"} {
		if !strings.Contains(res.YAML, want) {
			t.Fatalf("missing %q in yaml:\n%s", want, res.YAML)
		}
	}
	for _, bad := range []string{"mixed-port:", "external-controller:", "dns:"} {
		if strings.Contains(res.YAML, bad) {
			t.Fatalf("subscription yaml should not contain %q:\n%s", bad, res.YAML)
		}
	}
}

func TestBuildHashStableAcrossRuns(t *testing.T) {
	in := BuildInput{
		Proxies: []map[string]interface{}{
			ssProxy("US-a", "1.1.1.1", 443),
			ssProxy("JP-b", "2.2.2.2", 443),
			ssProxy("DE-c", "3.3.3.3", 443),
			ssProxy("FR-d", "4.4.4.4", 443),
			ssProxy("CA-e", "5.5.5.5", 443),
		},
		Groups: []database.ProxyGroup{
			group("直连", "select", `["DIRECT"]`),
			group("美国US", "url-test", `["REGION:US"]`, withURL("https://www.gstatic.com/generate_204")),
			group("其他国家", "url-test", `["REGION:OTHER"]`, withURL("https://www.gstatic.com/generate_204")),
		},
		Rules: []database.Rule{makeRule("MATCH", "", "直连")},
	}
	first := mustBuild(t, in)
	for i := 0; i < 10; i++ {
		res := mustBuild(t, in)
		if res.Hash != first.Hash || res.YAML != first.YAML {
			t.Fatalf("hash/yaml unstable at iter %d", i)
		}
	}
}

func TestSanitizeProxies(t *testing.T) {
	out, dropped := sanitizeProxiesForMeta([]map[string]interface{}{
		{
			"name": "a", "type": "vmess", "server": "1.1.1.1", "port": 443,
			"ws-path": "/", "ws-headers": map[string]interface{}{"Host": "example.com"},
			"ws-opts": map[string]interface{}{"path": "/", "headers": map[string]interface{}{"Host": "example.com"}},
		},
		{"name": "b", "type": "ss", "server": "2.2.2.2", "port": "8443"},
		{"name": "bad", "type": "vmess", "server": "3.3.3.3"},
	})
	if dropped != 0 || len(out) != 2 {
		t.Fatalf("dropped=%d len=%d", dropped, len(out))
	}
	if _, ok := out[0]["ws-path"]; ok {
		t.Fatal("ws-path should be removed when ws-opts present")
	}
	if _, ok := out[0]["ws-headers"]; ok {
		t.Fatal("ws-headers should be removed when ws-opts present")
	}
	if out[0]["port"] != 443 || out[1]["port"] != 8443 {
		t.Fatalf("ports=%v %v", out[0]["port"], out[1]["port"])
	}
}

func TestRealityShortIDHandling(t *testing.T) {
	res := mustBuild(t, BuildInput{
		Proxies: []map[string]interface{}{
			{
				"name": "JP-reality", "type": "vless", "server": "1.2.3.4", "port": 443,
				"uuid": "98bb17d9-9815-4923-a1d7-3d017ffd3f08", "tls": true, "client-fingerprint": "chrome",
				"reality-opts": map[string]interface{}{
					"public-key": "VOFSjjWT0wIH3Q0ntyEZd8WwksrIAb5gPt_3PBnEASg",
					"short-id":   "6314e825",
				},
			},
			ssProxy("US-a", "1.1.1.1", 443),
		},
		Groups: []database.ProxyGroup{
			group("直连", "select", `["DIRECT"]`),
			group("节点选择", "select", `["ALL"]`),
		},
		Rules:     []database.Rule{makeRule("MATCH", "", "直连")},
		GroupMode: "auto",
	})
	if !strings.Contains(res.YAML, `short-id: "6314e825"`) {
		t.Fatalf("expected quoted short-id:\n%s", res.YAML)
	}
	if strings.Contains(res.YAML, ".inf") || strings.Contains(res.YAML, ".Inf") {
		t.Fatalf("yaml must not contain .inf:\n%s", res.YAML)
	}
	jp := strings.Index(res.YAML, "name: JP-reality")
	tp := strings.Index(res.YAML, "type: vless")
	if jp < 0 || tp < 0 || jp > tp {
		t.Fatalf("expected name before type:\n%s", res.YAML)
	}
	if !strings.Contains(res.YAML, "# 1") {
		t.Fatalf("expected proxy index comments:\n%s", res.YAML)
	}
	var doc map[string]interface{}
	if err := yaml.Unmarshal([]byte(res.YAML), &doc); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range doc["proxies"].([]interface{}) {
		m := item.(map[string]interface{})
		if fmt.Sprint(m["name"]) != "JP-reality" {
			continue
		}
		found = true
		ro := m["reality-opts"].(map[string]interface{})
		if fmt.Sprint(ro["short-id"]) != "6314e825" {
			t.Fatalf("round-trip short-id=%q", ro["short-id"])
		}
	}
	if !found {
		t.Fatal("JP-reality missing after unmarshal")
	}

	out, dropped := sanitizeProxiesForMeta([]map[string]interface{}{
		{"name": "bad-inf", "type": "vless", "server": "1.1.1.1", "port": 443,
			"reality-opts": map[string]interface{}{"public-key": "k", "short-id": ".inf"}},
		{"name": "bad-float", "type": "vless", "server": "1.1.1.1", "port": 443,
			"reality-opts": map[string]interface{}{"public-key": "k", "short-id": math.Inf(1)}},
		{"name": "ok", "type": "vless", "server": "1.1.1.1", "port": 443,
			"reality-opts": map[string]interface{}{"public-key": "k", "short-id": "9c5b8c53"}},
	})
	if dropped != 2 || len(out) != 1 || fmt.Sprint(out[0]["name"]) != "ok" {
		t.Fatalf("dropped=%d out=%v", dropped, out)
	}
}

func TestGeneratorGroupModes(t *testing.T) {
	baseProxies := []map[string]interface{}{ssProxy("US-a", "1.1.1.1", 443)}
	baseGroups := []database.ProxyGroup{
		group("直连", "select", `["DIRECT"]`),
		group("节点选择", "select", `["ALL"]`),
		group("美国US", "select", `["REGION:US"]`),
		group("日本JP", "select", `["REGION:JP"]`),
		group("菲律宾PH", "select", `["REGION:PH"]`),
	}
	baseRules := []database.Rule{
		makeRule("DOMAIN-SUFFIX", "example.jp", "日本JP"),
		makeRule("MATCH", "", "直连"),
	}

	if _, err := NewGenerator().Build(BuildInput{
		Proxies: baseProxies,
		Groups:  []database.ProxyGroup{group("US", "select", `["REGION:US"]`)},
		Rules:   []database.Rule{makeRule("MATCH", "", "MISSING")},
	}); err == nil {
		t.Fatal("expected unknown rule target to fail")
	}

	all := mustBuild(t, BuildInput{Proxies: baseProxies, Groups: baseGroups, Rules: baseRules, GroupMode: "all"})
	if !strings.Contains(all.YAML, "name: 日本JP") || !strings.Contains(all.YAML, "DOMAIN-SUFFIX,example.jp,日本JP") {
		t.Fatalf("all mode should keep empty JP:\n%s", all.YAML)
	}

	auto := mustBuild(t, BuildInput{Proxies: baseProxies, Groups: baseGroups, Rules: baseRules, GroupMode: "auto"})
	if strings.Contains(auto.YAML, "name: 日本JP") || strings.Contains(auto.YAML, "name: 菲律宾PH") {
		t.Fatalf("auto should prune empty region groups:\n%s", auto.YAML)
	}
	if !strings.Contains(auto.YAML, "name: 美国US") || !strings.Contains(auto.YAML, "name: 节点选择") {
		t.Fatalf("auto should keep US and 节点选择:\n%s", auto.YAML)
	}
	if !strings.Contains(auto.YAML, "DOMAIN-SUFFIX,example.jp,节点选择") {
		t.Fatalf("pruned group should fallback 节点选择:\n%s", auto.YAML)
	}

	noSelect := mustBuild(t, BuildInput{
		Proxies: baseProxies,
		Groups: []database.ProxyGroup{
			group("直连", "select", `["DIRECT"]`),
			group("美国US", "select", `["REGION:US"]`),
			group("日本JP", "select", `["REGION:JP"]`),
		},
		Rules:     baseRules,
		GroupMode: "auto",
	})
	if !strings.Contains(noSelect.YAML, "DOMAIN-SUFFIX,example.jp,DIRECT") {
		t.Fatalf("without 节点选择 should fallback DIRECT:\n%s", noSelect.YAML)
	}

	custom := mustBuild(t, BuildInput{
		Proxies: []map[string]interface{}{
			ssProxy("US-a", "1.1.1.1", 443),
			ssProxy("PH-b", "2.2.2.2", 443),
		},
		Groups: []database.ProxyGroup{
			group("直连", "select", `["DIRECT"]`),
			group("美国US", "select", `["REGION:US"]`),
			group("菲律宾PH", "select", `["REGION:PH"]`),
		},
		Rules:         []database.Rule{makeRule("MATCH", "", "直连")},
		GroupMode:     "custom",
		AllowedGroups: []string{"直连", "菲律宾PH"},
	})
	if strings.Contains(custom.YAML, "name: 美国US") || !strings.Contains(custom.YAML, "name: 菲律宾PH") {
		t.Fatalf("custom whitelist unexpected:\n%s", custom.YAML)
	}
}

func TestGeneratorEmptyAndOtherRegions(t *testing.T) {
	res := mustBuild(t, BuildInput{
		Proxies: []map[string]interface{}{ssProxy("US-a", "1.1.1.1", 443)},
		Groups: []database.ProxyGroup{
			group("PH", "select", `["REGION:PH"]`),
			group("US", "select", `["REGION:US", "DIRECT"]`),
		},
		Rules: []database.Rule{makeRule("MATCH", "", "US")},
	})
	if len(res.GroupNames) != 1 || res.GroupNames[0] != "US" {
		t.Fatalf("unexpected groups: %v", res.GroupNames)
	}
	if _, err := NewGenerator().Build(BuildInput{
		Proxies: []map[string]interface{}{ssProxy("US-a", "1.1.1.1", 443)},
		Groups:  []database.ProxyGroup{group("PH", "select", `["REGION:PH"]`)},
		Rules:   []database.Rule{makeRule("MATCH", "", "PH")},
	}); err == nil {
		t.Fatal("expected error when all groups empty")
	}

	other := mustBuild(t, BuildInput{
		Proxies: []map[string]interface{}{
			ssProxy("US-a", "1.1.1.1", 443),
			ssProxy("DE-b", "2.2.2.2", 443),
			ssProxy("NG-c", "3.3.3.3", 443),
		},
		Groups: []database.ProxyGroup{
			group("美国US", "select", `["REGION:US"]`),
			group("其他国家", "select", `["REGION:OTHER"]`),
		},
		Rules: []database.Rule{makeRule("MATCH", "", "其他国家")},
	})
	for _, want := range []string{"DE-b", "NG-c", "其他国家"} {
		if !strings.Contains(other.YAML, want) {
			t.Fatalf("missing %q:\n%s", want, other.YAML)
		}
	}
}

func TestGeneratorSourceMembers(t *testing.T) {
	res := mustBuild(t, BuildInput{
		Proxies: []map[string]interface{}{
			ssProxy("US-a-良心云", "1.1.1.1", 443, map[string]interface{}{"_source_id": uint(1), "_source_name": "良心云"}),
			ssProxy("JP-b-良心云", "2.2.2.2", 443, map[string]interface{}{"_source_id": uint(1), "_source_name": "良心云"}),
			ssProxy("US-c-机场B", "3.3.3.3", 443, map[string]interface{}{"_source_id": uint(2), "_source_name": "机场B"}),
		},
		Groups: []database.ProxyGroup{
			group("直连", "select", `["DIRECT"]`),
			group("良心云", "select", `["SOURCE:良心云"]`),
			group("机场B", "url-test", `["SOURCE:机场B"]`, withURL("https://www.gstatic.com/generate_204")),
		},
		Rules: []database.Rule{makeRule("MATCH", "", "良心云")},
	})
	for _, want := range []string{"US-a-良心云", "JP-b-良心云", "US-c-机场B"} {
		if !strings.Contains(res.YAML, want) {
			t.Fatalf("missing %q:\n%s", want, res.YAML)
		}
	}
	if strings.Contains(res.YAML, "_source_id") || strings.Contains(res.YAML, "_source_name") {
		t.Fatalf("internal source meta leaked:\n%s", res.YAML)
	}

	byID := mustBuild(t, BuildInput{
		Proxies: []map[string]interface{}{
			ssProxy("US-a-old", "1.1.1.1", 443, map[string]interface{}{"_source_id": uint(3), "_source_name": "旧名"}),
			ssProxy("JP-b-other", "2.2.2.2", 443, map[string]interface{}{"_source_id": uint(9), "_source_name": "其它"}),
		},
		Groups: []database.ProxyGroup{
			group("按ID", "select", `["SOURCE:id:3"]`),
			group("直连", "select", `["DIRECT"]`),
		},
		Rules: []database.Rule{makeRule("MATCH", "", "按ID")},
	})
	if !strings.Contains(byID.YAML, "US-a-old") || !strings.Contains(byID.YAML, "JP-b-other") {
		t.Fatalf("SOURCE:id expansion unexpected:\n%s", byID.YAML)
	}

	ci := mustBuild(t, BuildInput{
		Proxies: []map[string]interface{}{
			ssProxy("US-a", "1.1.1.1", 443, map[string]interface{}{"_source_id": 1, "_source_name": "FooBar"}),
		},
		Groups: []database.ProxyGroup{group("源组", "select", `["SOURCE:foobar"]`)},
		Rules:  []database.Rule{makeRule("MATCH", "", "源组")},
	})
	if !strings.Contains(ci.YAML, "US-a") {
		t.Fatalf("SOURCE name match should be case-insensitive:\n%s", ci.YAML)
	}

	pruned := mustBuild(t, BuildInput{
		Proxies: []map[string]interface{}{
			ssProxy("US-a", "1.1.1.1", 443, map[string]interface{}{"_source_id": uint(1), "_source_name": "A"}),
		},
		Groups: []database.ProxyGroup{
			group("A", "select", `["SOURCE:A"]`),
			group("B", "select", `["SOURCE:B"]`),
		},
		Rules:     []database.Rule{makeRule("MATCH", "", "A")},
		GroupMode: "auto",
	})
	if strings.Contains(pruned.YAML, "name: B") || !strings.Contains(pruned.YAML, "name: A") {
		t.Fatalf("empty SOURCE group prune unexpected:\n%s", pruned.YAML)
	}
}

func TestFormatRuleOmitsCategory(t *testing.T) {
	res := mustBuild(t, BuildInput{
		Proxies: []map[string]interface{}{ssProxy("US-a", "1.1.1.1", 443)},
		Groups: []database.ProxyGroup{
			group("直连", "select", `["DIRECT"]`),
			group("美国US", "select", `["REGION:US"]`),
		},
		Rules: []database.Rule{
			{Type: "DOMAIN-SUFFIX", Payload: "openai.com", Target: "美国US", Enabled: true, Note: "仅后台", Category: "海外AI"},
			{Type: "MATCH", Target: "直连", Enabled: true, Category: "兜底"},
		},
	})
	if !strings.Contains(res.YAML, "DOMAIN-SUFFIX,openai.com,美国US") || !strings.Contains(res.YAML, "MATCH,直连") {
		t.Fatalf("unexpected rules:\n%s", res.YAML)
	}
	for _, bad := range []string{"海外AI", "兜底", "仅后台", "openai.com,美国US,"} {
		if strings.Contains(res.YAML, bad) {
			t.Fatalf("yaml must not contain %q:\n%s", bad, res.YAML)
		}
	}
}
