package rule

import (
	"encoding/json"
	"fmt"
	"strings"

	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/database"
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

const (
	maxRuleTypeLen     = 32
	maxRulePayloadLen  = 512
	maxRuleTargetLen   = 128
	maxRuleNoteLen     = 255
	maxRuleCategoryLen = 64
)

func validateRuleFields(typ, payload, target, note, category string) error {
	typ = strings.ToUpper(strings.TrimSpace(typ))
	if !isKnownRuleType(typ) {
		return fmt.Errorf("unknown rule type %q", strings.TrimSpace(typ))
	}
	if len([]rune(typ)) > maxRuleTypeLen {
		return fmt.Errorf("rule type too long")
	}
	if len([]rune(strings.TrimSpace(payload))) > maxRulePayloadLen {
		return fmt.Errorf("rule payload too long")
	}
	if len([]rune(strings.TrimSpace(target))) > maxRuleTargetLen {
		return fmt.Errorf("rule target too long")
	}
	if len([]rune(strings.TrimSpace(note))) > maxRuleNoteLen {
		return fmt.Errorf("rule note too long")
	}
	if len([]rune(strings.TrimSpace(category))) > maxRuleCategoryLen {
		return fmt.Errorf("rule category too long")
	}
	return nil
}

func isKnownRuleType(s string) bool {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "DOMAIN", "DOMAIN-SUFFIX", "DOMAIN-KEYWORD", "GEOSITE", "GEOIP",
		"IP-CIDR", "IP-CIDR6", "SRC-IP-CIDR", "SRC-PORT", "DST-PORT",
		"PROCESS-NAME", "PROCESS-PATH", "RULE-SET", "MATCH":
		return true
	default:
		return false
	}
}

func validateRule(typ, payload, target string) error {
	typ = strings.ToUpper(strings.TrimSpace(typ))
	payload = strings.TrimSpace(payload)
	target = strings.TrimSpace(target)
	if err := validateRuleFields(typ, payload, target, "", ""); err != nil {
		return err
	}
	if target == "" {
		return fmt.Errorf("rule target required")
	}
	if typ != string(common.RuleTypeMatch) && payload == "" {
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
