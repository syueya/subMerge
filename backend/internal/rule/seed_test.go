package rule

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/submerge/submerge/backend/internal/database"
)

func TestLoadSeedDefaults(t *testing.T) {
	groups, rules, err := loadSeedDefaults()
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) < 11 {
		t.Fatalf("expected at least 11 groups, got %d", len(groups))
	}
	if len(rules) < 20 {
		t.Fatalf("expected many rules, got %d", len(rules))
	}

	names := map[string]bool{}
	urlTestCount := 0
	for _, g := range groups {
		if g.Name == "" {
			t.Fatal("empty group name")
		}
		switch g.Name {
		case "AI", "流媒体", "电报", "PROXY", "US", "JP", "HK", "TW", "SG", "KR", "GB", "PH", "TR":
			t.Fatalf("legacy/bare group name %q should not exist", g.Name)
		}
		names[g.Name] = true
		if g.Type == "url-test" {
			urlTestCount++
			if g.URL == "" || g.Interval == nil || *g.Interval < 1 {
				t.Fatalf("url-test group %q missing url/interval", g.Name)
			}
		}
	}
	for _, want := range []string{
		"直连", "拒绝", "节点选择",
		"美国US", "日本JP", "香港HK", "台湾TW", "新加坡SG", "英国GB", "菲律宾PH", "土耳其TR",
		"其他国家",
	} {
		if !names[want] {
			t.Fatalf("missing group %s", want)
		}
	}
	if urlTestCount < 9 {
		t.Fatalf("expected >=9 url-test region groups, got %d", urlTestCount)
	}
	if groups[0].Name != "直连" || groups[1].Name != "拒绝" {
		t.Fatalf("first groups should be 直连, 拒绝; got %s, %s", groups[0].Name, groups[1].Name)
	}
	if last := groups[len(groups)-1]; last.Name != "节点选择" {
		t.Fatalf("last group should be 节点选择, got %s", last.Name)
	}

	assertGroupMembers(t, groups, "节点选择", []string{"ALL"})
	assertGroupMembers(t, groups, "其他国家", []string{"REGION:OTHER"})
	assertGroupMembers(t, groups, "拒绝", []string{"REJECT"})
	assertGroupContains(t, groups, "美国US", "REGION:US")

	hasRejectAd, hasAIToUS, hasMatch, geoOK := false, false, false, false
	hasDocker, hasGithub, hasTelegram := false, false, false
	for _, r := range rules {
		switch r.Target {
		case "DIRECT", "REJECT", "AI", "流媒体", "电报", "PROXY", "US", "JP", "HK":
			t.Fatalf("rule target should use policy group display name, got %q", r.Target)
		}
		if r.Type == "" || r.Target == "" {
			t.Fatalf("invalid rule: %+v", r)
		}
		if r.Type == "GEOSITE" && r.Payload == "category-ads-all" && r.Target == "拒绝" {
			hasRejectAd = true
		}
		// 海外 AI 走美国：GEOSITE category-ai-!cn 或 note 含 AI
		if r.Target == "美国US" && (r.Payload == "category-ai-!cn" || strings.Contains(r.Note, "AI")) {
			hasAIToUS = true
		}
		if r.Type == "MATCH" {
			hasMatch = true
			if r.Target != "美国US" {
				t.Fatalf("MATCH should target 美国US, got %s", r.Target)
			}
			if r.Category != "系统分类" {
				t.Fatalf("MATCH category = %q, want 系统分类", r.Category)
			}
		}
		if r.Type == "GEOIP" && r.Payload == "CN" {
			if r.Target != "直连" {
				t.Fatalf("GEOIP CN should target 直连, got %s", r.Target)
			}
			geoOK = true
		}
		if r.Target == "日本JP" && (r.Payload == "docker" || r.Payload == "hub.dockerhub.com") {
			hasDocker = true
		}
		if r.Target == "日本JP" && (r.Payload == "github" || r.Payload == "github.com") {
			hasGithub = true
		}
		if r.Target == "香港HK" && (r.Payload == "telegram" || r.Payload == "t.me") {
			hasTelegram = true
		}
	}
	if !hasRejectAd {
		t.Fatal("expected GEOSITE category-ads-all → 拒绝")
	}
	if !hasAIToUS {
		t.Fatal("expected overseas AI rules targeting 美国US")
	}
	if !hasDocker {
		t.Fatal("expected docker rule → 日本JP")
	}
	if !hasGithub {
		t.Fatal("expected github rule → 日本JP")
	}
	if !hasTelegram {
		t.Fatal("expected telegram rule → 香港HK")
	}
	if !hasMatch {
		t.Fatal("seed must include MATCH rule")
	}
	if !geoOK {
		t.Fatal("expected GEOIP CN → 直连")
	}

	last := rules[len(rules)-1]
	if last.Type != "MATCH" {
		t.Fatalf("last rule should be MATCH, got %s", last.Type)
	}
	var geoIdx, matchIdx, domainIdx = -1, -1, -1
	for i, r := range rules {
		if r.Type == "GEOSITE" && r.Payload == "category-ads-all" && i != 0 {
			t.Fatalf("ads rule should be first, index=%d", i)
		}
		if (r.Type == "DOMAIN" || r.Type == "DOMAIN-SUFFIX") && domainIdx < 0 {
			domainIdx = i
		}
		if r.Type == "GEOIP" && r.Payload == "CN" {
			geoIdx = i
		}
		if r.Type == "MATCH" {
			matchIdx = i
		}
	}
	if domainIdx < 0 || geoIdx < 0 || matchIdx < 0 {
		t.Fatal("missing DOMAIN/DOMAIN-SUFFIX / GEOIP / MATCH in seed")
	}
	if !(domainIdx < geoIdx && geoIdx < matchIdx) {
		t.Fatalf("order should be domain < GEOIP < MATCH, got %d %d %d", domainIdx, geoIdx, matchIdx)
	}
	for i, r := range rules {
		if r.SortOrder != i+1 {
			t.Fatalf("rule[%d] sortOrder want %d got %d (%s)", i, i+1, r.SortOrder, r.Type)
		}
	}
	for i, g := range groups {
		if g.SortOrder != i {
			t.Fatalf("group[%d] %q sortOrder want %d got %d", i, g.Name, i, g.SortOrder)
		}
	}
}

func assertGroupMembers(t *testing.T, groups []database.ProxyGroup, name string, want []string) {
	t.Helper()
	var got []string
	for _, g := range groups {
		if g.Name != name {
			continue
		}
		_ = json.Unmarshal([]byte(g.Proxies), &got)
		break
	}
	if len(got) != len(want) {
		t.Fatalf("%s members=%v want %v", name, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s members=%v want %v", name, got, want)
		}
	}
}

func assertGroupContains(t *testing.T, groups []database.ProxyGroup, name, member string) {
	t.Helper()
	for _, g := range groups {
		if g.Name != name {
			continue
		}
		var mem []string
		_ = json.Unmarshal([]byte(g.Proxies), &mem)
		for _, m := range mem {
			if strings.EqualFold(m, member) {
				return
			}
		}
		t.Fatalf("%s should include %s, got %v", name, member, mem)
	}
	t.Fatalf("group %s not found", name)
}
