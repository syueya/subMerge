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
	}

func loadSeedDefaults() (groups []database.ProxyGroup, rules []database.Rule, err error) {
	groups, err = loadSeedGroups()
	if err != nil {
		return nil, nil, err
	}
	rules, err = loadSeedRules()
	if err != nil {
		return nil, nil, err
	}
	if len(groups) == 0 && len(rules) == 0 {
		return nil, nil, fmt.Errorf("defaults groups.yaml and rules.yaml are both empty")
	}
	return groups, rules, nil
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
		enabled := true
		if r.Enabled != nil {
			enabled = *r.Enabled
		}
		// 列表顺序即 Clash 匹配顺序；显式 sortOrder 可覆盖
			order := i + 1
			if r.SortOrder != nil {
				order = *r.SortOrder
			}
			out = append(out, database.Rule{
				Type:      typ,
				Payload:   strings.TrimSpace(r.Payload),
				Target:    target,
				Enabled:   enabled,
				SortOrder: order,
				Note:      r.Note,
			})
		}
		return out, nil
	}
