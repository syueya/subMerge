package rule

import (
	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/database"
	"gorm.io/gorm"
)

// SeedDefaults 初始化默认策略组与规则（仅空库）。
// 内容来自 backend/defaults/groups.yaml + rules.yaml（go:embed 打进二进制）。
func (s *Service) SeedDefaults() error {
	groups, rules, err := loadSeedDefaults()
	if err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var gCount, rCount int64
		if err := tx.Model(&database.ProxyGroup{}).Count(&gCount).Error; err != nil {
			return err
		}
		if err := tx.Model(&database.Rule{}).Count(&rCount).Error; err != nil {
			return err
		}
		if gCount == 0 && rCount == 0 {
			for i := range groups {
				if err := tx.Create(&groups[i]).Error; err != nil {
					return err
				}
			}
			for i := range rules {
				if err := tx.Create(&rules[i]).Error; err != nil {
					return err
				}
			}
			return nil
		}
		// 已有库轻量迁移：直连/拒绝/节点选择 + 国家组改名 + 规则挂组名 + 广告规则
		if err := ensureNamedGroup(tx, common.GroupNameDirect, "select", []string{common.TargetDirect}, 0); err != nil {
			return err
		}
		if err := ensureNamedGroup(tx, common.GroupNameReject, "select", []string{common.TargetReject}, 1); err != nil {
			return err
		}
		// 总选组（排在列表末尾）：订阅投影时规则目标组被剪掉后优先回退到此
		if err := ensureNamedGroup(tx, common.GroupNameSelectAll, "select", []string{common.MemberTokenAll}, 100); err != nil {
			return err
		}
		// 已有库若先前插在靠前位置，启动时挪到末尾
		if err := tx.Model(&database.ProxyGroup{}).
			Where("name = ?", common.GroupNameSelectAll).
			Update("sort_order", 100).Error; err != nil {
			return err
		}
		// 规则 target：引擎关键字 → 策略组名
		if err := tx.Model(&database.Rule{}).Where("target = ?", common.TargetDirect).
			Update("target", common.GroupNameDirect).Error; err != nil {
			return err
		}
		if err := tx.Model(&database.Rule{}).Where("target = ?", common.TargetReject).
			Update("target", common.GroupNameReject).Error; err != nil {
			return err
		}
		// 旧短码国家组 → 「中文+码」（美国US…）；规则 target 同步
		if err := migrateCountryGroupNames(tx); err != nil {
			return err
		}
		// 旧业务策略组名 → 美国US
		for _, old := range []string{"AI", "流媒体", "电报", "PROXY", "OpenAI"} {
			if err := tx.Model(&database.Rule{}).Where("target = ?", old).
				Update("target", common.GroupNameDefaultUS).Error; err != nil {
				return err
			}
		}
		// 补「其他国家」组（非常用地区节点）
		if err := ensureNamedGroup(tx, common.GroupNameOther, "url-test", []string{common.RegionTokenOther}, 50); err != nil {
			return err
		}
		// 历史库：url-test/fallback 可能缺 url/interval（前端会显示 ?s）
		if err := repairURLTestGroups(tx); err != nil {
			return err
		}
			// 系统规则（广告/国内/MATCH）由代码托管，缺则补；已有 MATCH 出口不改
		if err := ensureSystemRules(tx); err != nil {
			return err
		}
		// 从 defaults/rules.yaml 补缺业务规则（不覆盖用户已改过的 type+payload）
		if err := syncMissingDefaultRulesFromSeed(tx); err != nil {
			return err
		}
		// 已有库回填业务分类（空 category 才写，不覆盖用户已设）
		if err := backfillRuleCategories(tx); err != nil {
			return err
		}
		// 旧「广告/国内/兜底」→ 统一系统分类
		if err := migrateSystemCategories(tx); err != nil {
			return err
		}
		// 系统规则固定顺序：广告最先、国内 GEOIP 倒数第二、MATCH 最后
		return ensureSystemRuleOrder(tx)
	})
}

// syncMissingDefaultRulesFromSeed 业务默认源：backend/defaults/rules.yaml。
// 老库启动时：YAML 里有、库中没有的业务规则补上；已有 type+payload 不改 target（用户改过的出口保留）。
// 系统规则不在此处理（见 ensureSystemRules）。
func syncMissingDefaultRulesFromSeed(tx *gorm.DB) error {
	seed, err := loadSeedRules()
	if err != nil {
		return err
	}
	var existing []database.Rule
	if err := tx.Find(&existing).Error; err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(existing))
	for _, r := range existing {
		seen[ruleIdentity(r.Type, r.Payload)] = struct{}{}
	}

	anchor, hasAnchor := resolveInsertAnchor(existing)

	toInsert := make([]database.Rule, 0)
	for i := range seed {
		s := seed[i]
		key := ruleIdentity(s.Type, s.Payload)
		if _, ok := seen[key]; ok {
			continue
		}
		toInsert = append(toInsert, s)
		seen[key] = struct{}{}
	}

	if len(toInsert) == 0 {
		return nil
	}
	shift := len(toInsert) * 10
	if hasAnchor && shift > 0 {
		if err := tx.Model(&database.Rule{}).
			Where("sort_order >= ?", anchor).
			Update("sort_order", gorm.Expr("sort_order + ?", shift)).Error; err != nil {
			return err
		}
	}
	for i, s := range toInsert {
		row := database.Rule{
			Type:      s.Type,
			Payload:   s.Payload,
			Target:    s.Target,
			Enabled:   s.Enabled,
			SortOrder: anchor + i*10,
			Note:      s.Note,
			Category:  s.Category,
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}
