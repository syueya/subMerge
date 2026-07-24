package rule

import (
	"fmt"
	"strings"

	"github.com/submerge/submerge/backend/defaults"
	"github.com/submerge/submerge/backend/internal/database"
	"gopkg.in/yaml.v3"
)

// seedGroupsFile 与 backend/defaults/groups.yaml 结构一致
type seedGroupsFile struct {
	Groups []seedGroup `yaml:"groups"`
}

// seedRulesFile 与 backend/defaults/rules.yaml 结构一致
type seedRulesFile struct {
	Rules []seedRule `yaml:"rules"`
}

type seedGroup struct {
		Name     string   `yaml:"name"`
		Type     string   `yaml:"type"`
		Proxies  []string `yaml:"proxies"`
		Enabled  *bool    `yaml:"enabled"`
		// SortOrder 可选；省略则按 YAML 列表顺序自动编号
		SortOrder *int   `yaml:"sortOrder"`
		URL       string `yaml:"url"`
		Interval  *int   `yaml:"interval"`
	}

type seedRule struct {
			Type    string `yaml:"type"`
			Payload string `yaml:"payload"`
			Target  string `yaml:"target"`
			Enabled *bool  `yaml:"enabled"`
			// SortOrder 可选；省略则按 YAML 列表顺序自动编号（Clash 匹配顺序）
			SortOrder *int   `yaml:"sortOrder"`
			Note      string `yaml:"note"`
			// Category 业务分类（面板分组，不进 Clash）
			Category  string `yaml:"category"`
		}

func loadSeedDefaults() (groups []database.ProxyGroup, rules []database.Rule, err error) {
	groups, err = loadSeedGroups()
	if err != nil {
		return nil, nil, err
	}
	business, err := loadSeedRules()
	if err != nil {
		return nil, nil, err
	}
	if len(groups) == 0 && len(business) == 0 {
		return nil, nil, fmt.Errorf("defaults groups.yaml and rules.yaml are both empty")
	}
	// 空库写入：系统规则 + 业务规则（顺序：广告 → 业务 → 国内 → MATCH）
	return groups, composeSeedRules(business), nil
}

// systemRuleCategory 系统内置规则统一分类（广告 / 国内 GEOIP / MATCH 兜底）
const systemRuleCategory = "系统分类"

// systemRuleDefs 固定系统规则（不进 rules.yaml，由代码生成）。
// 顺序约定：广告最先、国内 GEOIP 倒数第二、MATCH 最后。
func systemRuleDefs() []database.Rule {
	return []database.Rule{
		{
			Type:     "GEOSITE",
			Payload:  "category-ads-all",
			Target:   "拒绝",
			Enabled:  true,
			Note:     "广告",
			Category: systemRuleCategory,
		},
		{
			Type:     "GEOIP",
			Payload:  "CN",
			Target:   "直连",
			Enabled:  true,
			Note:     "国内直连",
			Category: systemRuleCategory,
		},
		{
			Type:     "MATCH",
			Payload:  "",
			Target:   "美国US",
			Enabled:  true,
			Note:     "默认走代理",
			Category: systemRuleCategory,
		},
	}
}

// isSystemSeedRule 判断是否为代码托管的系统规则（YAML 不应再写）。
func isSystemSeedRule(typ, payload string) bool {
	typ = strings.TrimSpace(typ)
	payload = strings.TrimSpace(payload)
	if typ == "MATCH" {
		return true
	}
	if typ == "GEOIP" && strings.EqualFold(payload, "CN") {
		return true
	}
	if typ == "GEOSITE" && payload == "category-ads-all" {
		return true
	}
	return false
}

// composeSeedRules 组装空库完整规则列表：广告 + 业务 + 国内 + MATCH，并连续编号。
func composeSeedRules(business []database.Rule) []database.Rule {
	sys := systemRuleDefs()
	ad, cn, match := sys[0], sys[1], sys[2]
	out := make([]database.Rule, 0, len(business)+3)
	out = append(out, ad)
	out = append(out, business...)
	out = append(out, cn, match)
	for i := range out {
		out[i].SortOrder = i + 1
	}
	return out
}

func loadSeedGroups() ([]database.ProxyGroup, error) {
	raw := defaults.GroupsYAML
	if len(raw) == 0 {
		return nil, fmt.Errorf("embedded defaults/groups.yaml is empty")
	}
	var file seedGroupsFile
	if err := yaml.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("parse defaults/groups.yaml: %w", err)
	}
	if len(file.Groups) == 0 {
		return nil, fmt.Errorf("defaults/groups.yaml has no groups")
	}

	out := make([]database.ProxyGroup, 0, len(file.Groups))
	for i, g := range file.Groups {
		name := strings.TrimSpace(g.Name)
		if name == "" {
			return nil, fmt.Errorf("seed group[%d]: name required", i)
		}
		typ := strings.TrimSpace(g.Type)
		if typ == "" {
			typ = "select"
		}
		enabled := true
		if g.Enabled != nil {
			enabled = *g.Enabled
		}
		proxies := g.Proxies
		if proxies == nil {
			proxies = []string{}
		}
		// 列表顺序即面板排列；显式 sortOrder 可覆盖
			order := i
			if g.SortOrder != nil {
				order = *g.SortOrder
			}
			out = append(out, database.ProxyGroup{
				Name:      name,
				Type:      typ,
				Proxies:   mustJSON(proxies),
				URL:       g.URL,
				Interval:  g.Interval,
				Enabled:   enabled,
				SortOrder: order,
			})
		}
		return out, nil
	}

// loadSeedRules 只读业务默认（backend/defaults/rules.yaml）。
// 系统规则（广告 / 国内 GEOIP / MATCH）不在此文件，由 systemRuleDefs + ensureSystemRules 托管。
func loadSeedRules() ([]database.Rule, error) {
	raw := defaults.RulesYAML
	if len(raw) == 0 {
		return nil, fmt.Errorf("embedded defaults/rules.yaml is empty")
	}
	var file seedRulesFile
	if err := yaml.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("parse defaults/rules.yaml: %w", err)
	}
	if len(file.Rules) == 0 {
		return nil, fmt.Errorf("defaults/rules.yaml has no rules")
	}

	out := make([]database.Rule, 0, len(file.Rules))
	for i, r := range file.Rules {
		typ := strings.TrimSpace(r.Type)
		target := strings.TrimSpace(r.Target)
		if typ == "" {
			return nil, fmt.Errorf("seed rule[%d]: type required", i)
		}
		if target == "" {
			return nil, fmt.Errorf("seed rule[%d]: target required", i)
		}
		payload := strings.TrimSpace(r.Payload)
		// YAML 误写系统规则时跳过（真正来源是 systemRuleDefs）
		if isSystemSeedRule(typ, payload) {
			continue
		}
		enabled := true
		if r.Enabled != nil {
			enabled = *r.Enabled
		}
		// 列表顺序即业务段内匹配顺序；显式 sortOrder 可覆盖
		order := i + 1
		if r.SortOrder != nil {
			order = *r.SortOrder
		}
		note := strings.TrimSpace(r.Note)
		cat := strings.TrimSpace(r.Category)
		if cat == "" {
			cat = inferRuleCategory(typ, payload, target, note)
		}
		out = append(out, database.Rule{
			Type:      typ,
			Payload:   payload,
			Target:    target,
			Enabled:   enabled,
			SortOrder: order,
			Note:      note,
			Category:  cat,
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("defaults/rules.yaml has no business rules")
	}
	return out, nil
}
