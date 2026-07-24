package rule

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLoadSeedDefaults(t *testing.T) {
	groups, rules, err := loadSeedDefaults()
	if err != nil {
		t.Fatal(err)
	}
	// 直连 + 拒绝 + 常用国家 + 其他国家
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
		// 不应有业务用途策略组
		switch g.Name {
		case "AI", "流媒体", "电报", "PROXY":
			t.Fatalf("business group %q should not exist; use country targets in rules", g.Name)
		}
		// 国家组应为「中文+码」或「其他国家」，不再用裸 US/JP
		switch g.Name {
		case "US", "JP", "HK", "TW", "SG", "KR", "GB", "PH", "TR":
			t.Fatalf("country group should use DisplayName form, got bare %q", g.Name)
		}
		names[g.Name] = true
		if g.Type == "url-test" {
			urlTestCount++
			if g.URL == "" {
				t.Fatalf("url-test group %q missing url", g.Name)
			}
			if g.Interval == nil || *g.Interval < 1 {
				t.Fatalf("url-test group %q missing interval", g.Name)
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
	// 常用国家 url-test + 「其他国家」；当前默认约 9 组（无韩国独立组）
			if urlTestCount < 9 {
				t.Fatalf("expected >=9 url-test region groups, got %d", urlTestCount)
			}
		if groups[0].Name != "直连" || groups[1].Name != "拒绝" {
			t.Fatalf("first groups should be 直连, 拒绝; got %s, %s", groups[0].Name, groups[1].Name)
		}
		lastGroup := groups[len(groups)-1]
		if lastGroup.Name != "节点选择" {
			t.Fatalf("last group should be 节点选择, got %s", lastGroup.Name)
		}
		var selectMembers []string
		_ = json.Unmarshal([]byte(lastGroup.Proxies), &selectMembers)
		if len(selectMembers) != 1 || selectMembers[0] != "ALL" {
			t.Fatalf("节点选择 should be [ALL], got %v", selectMembers)
		}

	// 其他国家成员应为 REGION:OTHER
	var otherMembers []string
	for _, g := range groups {
		if g.Name == "其他国家" {
			_ = json.Unmarshal([]byte(g.Proxies), &otherMembers)
			break
		}
	}
	if len(otherMembers) != 1 || otherMembers[0] != "REGION:OTHER" {
		t.Fatalf("其他国家 should be [REGION:OTHER], got %v", otherMembers)
	}

	var rejectMembers []string
	for _, g := range groups {
		if g.Name == "拒绝" {
			_ = json.Unmarshal([]byte(g.Proxies), &rejectMembers)
			break
		}
	}
	if len(rejectMembers) != 1 || rejectMembers[0] != "REJECT" {
		t.Fatalf("拒绝 group should be [REJECT], got %v", rejectMembers)
	}

	hasRejectAd := false
	hasAIToUS := false
	hasMatch := false
	geoOK := false
	for _, r := range rules {
		if r.Target == "DIRECT" || r.Target == "REJECT" {
			t.Fatalf("rule target should be policy group name, got %q", r.Target)
		}
		if r.Target == "AI" || r.Target == "流媒体" || r.Target == "电报" || r.Target == "PROXY" {
			t.Fatalf("rule should target country/直连/拒绝, got business group %q", r.Target)
		}
		// 旧短码目标不应再出现
		if r.Target == "US" || r.Target == "JP" || r.Target == "HK" {
			t.Fatalf("rule target should use DisplayName form, got bare %q", r.Target)
		}
		if r.Type == "GEOSITE" && r.Payload == "category-ads-all" && r.Target == "拒绝" {
			hasRejectAd = true
		}
// note 形如 AI / AI-OpenAI / AI-Claude
			if r.Target == "美国US" && (r.Note == "AI" || strings.HasPrefix(r.Note, "AI-")) {
				hasAIToUS = true
			}
if r.Type == "MATCH" {
					hasMatch = true
					// 系统兜底：未命中规则默认走美国US
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
			if r.Type == "" || r.Target == "" {
				t.Fatalf("invalid rule: %+v", r)
			}
		}
		if !hasRejectAd {
			t.Fatal("expected GEOSITE category-ads-all → 拒绝")
		}
		if !hasAIToUS {
			t.Fatal("expected AI rules targeting 美国US")
		}
		hasDocker := false
		hasGithubJP := false
		hasTGHK := false
		for _, r := range rules {
			if r.Payload == "docker.io" && r.Target == "日本JP" {
				hasDocker = true
			}
			if r.Payload == "github.com" && r.Target == "日本JP" {
				hasGithubJP = true
			}
			if r.Payload == "t.me" && r.Target == "香港HK" {
				hasTGHK = true
			}
		}
		if !hasDocker {
			t.Fatal("expected docker.io → 日本JP")
		}
		if !hasGithubJP {
			t.Fatal("expected github.com → 日本JP")
		}
		if !hasTGHK {
			t.Fatal("expected t.me → 香港HK")
		}
		hasGrok := false
		hasZhipu := false
		for _, r := range rules {
			if r.Payload == "x.ai" && r.Target == "美国US" {
				hasGrok = true
			}
			if r.Payload == "bigmodel.cn" && r.Target == "直连" {
				hasZhipu = true
			}
		}
		if !hasGrok {
			t.Fatal("expected x.ai → 美国US")
		}
		if !hasZhipu {
			t.Fatal("expected bigmodel.cn → 直连")
		}
		if !hasMatch {
			t.Fatal("seed must include MATCH rule")
		}
		if !geoOK {
			t.Fatal("expected GEOIP CN → 直连")
		}
			// 列表顺序：可有个人优先规则在前；须含广告 GEOSITE；GEOIP 在 MATCH 前；MATCH 最后
			hasAds := false
			for _, r := range rules {
				if r.Type == "GEOSITE" && r.Payload == "category-ads-all" {
					hasAds = true
					break
				}
			}
			if !hasAds {
				t.Fatal("expected GEOSITE category-ads-all in seed")
			}
			last := rules[len(rules)-1]
			if last.Type != "MATCH" {
				t.Fatalf("last rule should be MATCH, got %s", last.Type)
			}
		var geoIdx, matchIdx, domainIdx = -1, -1, -1
		for i, r := range rules {
			if r.Type == "GEOIP" && r.Payload == "CN" {
				geoIdx = i
			}
			if r.Type == "MATCH" {
				matchIdx = i
			}
			if r.Type == "DOMAIN-SUFFIX" && domainIdx < 0 {
				domainIdx = i
			}
		}
		if domainIdx < 0 || geoIdx < 0 || matchIdx < 0 {
			t.Fatal("missing DOMAIN-SUFFIX / GEOIP / MATCH in seed")
		}
		if !(domainIdx < geoIdx && geoIdx < matchIdx) {
			t.Fatalf("order should be domain < GEOIP < MATCH, got %d %d %d", domainIdx, geoIdx, matchIdx)
		}
		// 自动编号：连续递增
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

	// 美国US 组成员含 REGION:US
	for _, g := range groups {
		if g.Name != "美国US" {
			continue
		}
		var mem []string
		_ = json.Unmarshal([]byte(g.Proxies), &mem)
		ok := false
		for _, m := range mem {
			if strings.EqualFold(m, "REGION:US") {
				ok = true
			}
		}
		if !ok {
			t.Fatalf("美国US should include REGION:US, got %v", mem)
		}
	}
}
