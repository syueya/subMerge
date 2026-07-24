package rule

import (
	"encoding/json"
	"fmt"
	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/database"
	"strings"
)

// ruleIdentity 用于判断「是否已有同一条默认规则」：type + payload（忽略大小写）
func ruleIdentity(typ, payload string) string {
	return strings.ToUpper(strings.TrimSpace(typ)) + "\x00" + strings.ToLower(strings.TrimSpace(payload))
}

// insertAnchorBeforeSystem 返回国内 GEOIP / MATCH 中较小的 sort_order（新规则应插在此之前）。
// has=false 时调用方应接在末尾（max+10）。
func insertAnchorBeforeSystem(existing []database.Rule) (anchor int, has bool) {
	for _, r := range existing {
		if r.Type == "MATCH" || (r.Type == "GEOIP" && strings.EqualFold(r.Payload, "CN")) {
			if !has || r.SortOrder < anchor {
				anchor = r.SortOrder
				has = true
			}
		}
	}
	return anchor, has
}

// nextOrderAfterMax 在无系统锚点时，返回现有规则 sort_order 最大值 + 10（空列表则为 10）。
func nextOrderAfterMax(existing []database.Rule) int {
	maxOrder := 0
	for _, r := range existing {
		if r.SortOrder > maxOrder {
			maxOrder = r.SortOrder
		}
	}
	return maxOrder + 10
}

// resolveInsertAnchor 统一「插在系统规则前」的锚点：有 GEOIP CN/MATCH 用其较小 sort_order；否则接末尾。
// hasAnchor 表示是否应对锚点及之后做整体后移。
func resolveInsertAnchor(existing []database.Rule) (anchor int, hasAnchor bool) {
	anchor, hasAnchor = insertAnchorBeforeSystem(existing)
	if !hasAnchor {
		return nextOrderAfterMax(existing), false
	}
	return anchor, true
}

func validateRule(typ, payload, target string) error {
	if strings.TrimSpace(typ) == "" {
		return fmt.Errorf("rule type required")
	}
	if strings.TrimSpace(target) == "" {
		return fmt.Errorf("rule target required")
	}
	if typ != string(common.RuleTypeMatch) && strings.TrimSpace(payload) == "" {
		return fmt.Errorf("payload required for type %s", typ)
	}
	return nil
}

func toRule(r database.Rule) common.Rule {
	return common.Rule{
		ID:        r.ID,
		Type:      common.RuleType(r.Type),
		Payload:   r.Payload,
		Target:    r.Target,
		Enabled:   r.Enabled,
		SortOrder: r.SortOrder,
		Note:      r.Note,
		Category:  r.Category,
	}
}

func toGroup(r database.ProxyGroup) common.ProxyGroup {
	var proxies []string
	_ = json.Unmarshal([]byte(r.Proxies), &proxies)
	if proxies == nil {
		proxies = []string{}
	}
	return common.ProxyGroup{
		ID:        r.ID,
		Name:      r.Name,
		Type:      common.ProxyGroupType(r.Type),
		Proxies:   proxies,
		URL:       r.URL,
		Interval:  r.Interval,
		Enabled:   r.Enabled,
		SortOrder: r.SortOrder,
	}
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
