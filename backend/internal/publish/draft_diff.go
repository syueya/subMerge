package publish

import (
	"fmt"
	"strings"

	common "github.com/submerge/submerge/backend/common"
	"gopkg.in/yaml.v3"
)

// configSnapshot 从已发布/草稿 YAML 抽出可比较的实体快照
type configSnapshot struct {
	proxies []string            // 节点名（有序）
	groups  map[string][]string // 策略组名 → 成员（有序）
	order   []string            // 策略组名有序列表
	rules   []string            // 规则原始行（有序）
}

// parseConfigSnapshot 解析 YAML 为实体快照，供 diff 使用
func parseConfigSnapshot(yamlText string) configSnapshot {
	snap := configSnapshot{groups: map[string][]string{}}
	if strings.TrimSpace(yamlText) == "" {
		return snap
	}
	var doc map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlText), &doc); err != nil {
		return snap
	}
	if raw, ok := doc["proxies"].([]interface{}); ok {
		for _, p := range raw {
			m, ok := p.(map[string]interface{})
			if !ok {
				continue
			}
			if name := strings.TrimSpace(fmt.Sprint(m["name"])); name != "" && name != "<nil>" {
				snap.proxies = append(snap.proxies, name)
			}
		}
	}
	if raw, ok := doc["proxy-groups"].([]interface{}); ok {
		for _, g := range raw {
			m, ok := g.(map[string]interface{})
			if !ok {
				continue
			}
			name := strings.TrimSpace(fmt.Sprint(m["name"]))
			if name == "" || name == "<nil>" {
				continue
			}
			members := []string{}
			if mem, ok := m["proxies"].([]interface{}); ok {
				for _, x := range mem {
					members = append(members, fmt.Sprint(x))
				}
			}
			snap.groups[name] = members
			snap.order = append(snap.order, name)
		}
	}
	if raw, ok := doc["rules"].([]interface{}); ok {
		for _, r := range raw {
			line := strings.TrimSpace(fmt.Sprint(r))
			if line != "" && line != "<nil>" {
				snap.rules = append(snap.rules, line)
			}
		}
	}
	return snap
}

// diffConfigs 对比已发布与草稿 YAML，产出实体级变更列表（节点数量、策略组增删改、规则增删）。
func diffConfigs(publishedYAML, draftYAML string) []common.DraftChange {
	oldSnap := parseConfigSnapshot(publishedYAML)
	newSnap := parseConfigSnapshot(draftYAML)
	changes := []common.DraftChange{}

	// 节点：逐个列出新增 / 删除的节点名
	addedProxies, removedProxies := diffStringSlices(oldSnap.proxies, newSnap.proxies)
	for _, name := range addedProxies {
		changes = append(changes, common.DraftChange{
			Kind: "proxy", Action: "added", Name: name,
		})
	}
for _, name := range removedProxies {
			changes = append(changes, common.DraftChange{
				Kind: "proxy", Action: "removed", Name: name,
			})
		}

		// 策略组：增 / 删 / 成员修改（成员细节只记数量，节点名已在 proxy 变更中列出）
		for _, name := range newSnap.order {
			newMembers := newSnap.groups[name]
			oldMembers, existed := oldSnap.groups[name]
			if !existed {
				changes = append(changes, common.DraftChange{
					Kind: "group", Action: "added", Name: name,
					Detail: memberSummary(newMembers),
				})
				continue
			}
			if strings.Join(oldMembers, ",") != strings.Join(newMembers, ",") {
				addedM, removedM := diffStringSlices(oldMembers, newMembers)
				changes = append(changes, common.DraftChange{
					Kind: "group", Action: "modified", Name: name,
					Detail: memberDiffSummary(addedM, removedM),
				})
			}
		}
		for _, name := range oldSnap.order {
			if _, ok := newSnap.groups[name]; !ok {
				changes = append(changes, common.DraftChange{
					Kind: "group", Action: "removed", Name: name,
				})
			}
		}

	// 规则：按内容集合比较增删（保留顺序展示）
	oldRuleSet := make(map[string]struct{}, len(oldSnap.rules))
	for _, r := range oldSnap.rules {
		oldRuleSet[r] = struct{}{}
	}
	newRuleSet := make(map[string]struct{}, len(newSnap.rules))
	for _, r := range newSnap.rules {
		newRuleSet[r] = struct{}{}
	}
	for _, r := range newSnap.rules {
		if _, ok := oldRuleSet[r]; !ok {
			changes = append(changes, common.DraftChange{
				Kind: "rule", Action: "added", Name: r,
			})
		}
	}
	for _, r := range oldSnap.rules {
		if _, ok := newRuleSet[r]; !ok {
			changes = append(changes, common.DraftChange{
				Kind: "rule", Action: "removed", Name: r,
			})
		}
	}

	return changes
}

// diffStringSlices 返回 newer 相对 older 的新增项与删除项（按 newer/older 原顺序）。
func diffStringSlices(older, newer []string) (added, removed []string) {
	oldSet := make(map[string]struct{}, len(older))
	for _, v := range older {
		oldSet[v] = struct{}{}
	}
	newSet := make(map[string]struct{}, len(newer))
	for _, v := range newer {
		newSet[v] = struct{}{}
	}
	for _, v := range newer {
		if _, ok := oldSet[v]; !ok {
			added = append(added, v)
		}
	}
	for _, v := range older {
		if _, ok := newSet[v]; !ok {
			removed = append(removed, v)
		}
	}
	return added, removed
}

// memberSummary 新增策略组时只记成员数量（节点级变更已单独列出，不必再展开名单）。
func memberSummary(members []string) string {
	if len(members) == 0 {
		return "无成员"
	}
	return fmt.Sprintf("%d 个成员", len(members))
}

// memberDiffSummary 策略组成员修改时只记数量摘要，避免与节点增删重复罗列长名。
func memberDiffSummary(added, removed []string) string {
	parts := []string{}
	if len(added) > 0 {
		parts = append(parts, fmt.Sprintf("+%d", len(added)))
	}
	if len(removed) > 0 {
		parts = append(parts, fmt.Sprintf("-%d", len(removed)))
	}
	if len(parts) == 0 {
		return "成员顺序调整"
	}
	return "成员 " + strings.Join(parts, "/")
}
