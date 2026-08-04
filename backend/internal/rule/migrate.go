package rule

import (
	"strings"

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
		// 旧 MATCH 指向短码/PROXY → 美国US（默认兜底）
		for _, bad := range []string{"US", "PROXY"} {
			if err := tx.Model(&database.Rule{}).
				Where("type = ? AND target = ?", "MATCH", bad).
				Updates(map[string]interface{}{
					"target": common.GroupNameDefaultUS,
					"note":   "默认走代理",
				}).Error; err != nil {
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
		// 系统规则（广告/国内/MATCH）由代码托管，缺则补、旧默认出口可对齐
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

// backfillRuleCategories 给空 category 的规则回填业务分类。
// 优先按 defaults 的 type+payload 对齐；否则用 note/target 启发式；不覆盖已有 category。
func backfillRuleCategories(tx *gorm.DB) error {
	seed, err := loadSeedRules()
	if err != nil {
		return err
	}
	seedCat := make(map[string]string, len(seed))
	for _, s := range seed {
		cat := strings.TrimSpace(s.Category)
		if cat == "" {
			continue
		}
		seedCat[ruleIdentity(s.Type, s.Payload)] = cat
	}

	var rows []database.Rule
	if err := tx.Where("category = ? OR category IS NULL", "").Find(&rows).Error; err != nil {
		return err
	}
	for _, r := range rows {
		cat := seedCat[ruleIdentity(r.Type, r.Payload)]
		if cat == "" {
			cat = inferRuleCategory(r.Type, r.Payload, r.Target, r.Note)
		}
		if cat == "" {
			continue
		}
		if err := tx.Model(&database.Rule{}).Where("id = ? AND (category = ? OR category IS NULL)", r.ID, "").
			Update("category", cat).Error; err != nil {
			return err
		}
	}
	return nil
}

// inferRuleCategory 启发式推断业务分类。
// 系统规则（广告 GEOSITE / 国内 GEOIP CN / MATCH）靠 type+payload 识别。
func inferRuleCategory(typ, payload, target, note string) string {
	typ = strings.TrimSpace(typ)
	note = strings.TrimSpace(note)
	target = strings.TrimSpace(target)
	payload = strings.TrimSpace(payload)

	// 系统规则：广告 / 国内 / 兜底 → 统一「系统分类」
	if isSystemSeedRule(typ, payload) {
		return systemRuleCategory
	}
	if note == "广告" || strings.HasPrefix(note, "广告") {
		return systemRuleCategory
	}
	if note == "AI" || strings.HasPrefix(note, "AI-") || strings.HasPrefix(note, "AI") {
		if target == common.GroupNameDirect || target == common.TargetDirect {
			return "国内AI"
		}
		return "海外AI"
	}
	switch note {
	case "流媒体":
		return "流媒体"
	case "电报":
		return "电报"
	case "社交":
		return "社交"
	case "Google":
		return "Google"
	case "GitHub", "Docker":
		return "开发"
	case "TMDB", "TMDB图片CDN", "Fanart", "TVDB", "IMDb", "IMDb图片", "OMDb", "MusicBrainz", "CoverArt":
		return "影视元数据"
	case "国内直连", "默认走代理", "默认走直连":
		return systemRuleCategory
	case "搜索", "基础设施", "游戏", "NAS", "GG卡", "Z站", "阡陌居":
		return "其它"
	}
	return ""
}

// migrateSystemCategories 把历史「广告/国内/兜底」统一改成「系统分类」。
func migrateSystemCategories(tx *gorm.DB) error {
	return tx.Model(&database.Rule{}).
		Where("category IN ?", []string{"广告", "国内", "兜底"}).
		Update("category", systemRuleCategory).Error
}

// ensureSystemRuleOrder 固定系统规则匹配顺序：
//  1. 广告 GEOSITE category-ads-all 最先
//  2. 中间：其余业务规则
//  3. 国内 GEOIP CN 倒数第二
//  4. MATCH 兜底最后
//
// 历史库常见：GEOIP 被迁到 900、MATCH 停在 300+、广告不在最前；补域名后订阅会报 invalid rules。
func ensureSystemRuleOrder(tx *gorm.DB) error {
	var rows []database.Rule
	if err := tx.Order("sort_order asc, id asc").Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	isAd := func(r database.Rule) bool {
		return r.Type == "GEOSITE" && r.Payload == "category-ads-all"
	}
	isCN := func(r database.Rule) bool {
		return r.Type == "GEOIP" && strings.EqualFold(r.Payload, "CN")
	}
	isMatch := func(r database.Rule) bool {
		return r.Type == "MATCH"
	}

	var ads, middles, cns, matches []database.Rule
	for _, r := range rows {
		switch {
		case isAd(r):
			ads = append(ads, r)
		case isMatch(r):
			matches = append(matches, r)
		case isCN(r):
			cns = append(cns, r)
		default:
			middles = append(middles, r)
		}
	}

	ordered := make([]database.Rule, 0, len(rows))
	ordered = append(ordered, ads...)
	ordered = append(ordered, middles...)
	ordered = append(ordered, cns...)
	ordered = append(ordered, matches...)

	for i, r := range ordered {
		want := (i + 1) * 10
		if r.SortOrder == want {
			continue
		}
		if err := tx.Model(&database.Rule{}).Where("id = ?", r.ID).
			Update("sort_order", want).Error; err != nil {
			return err
		}
	}
	return nil
}

const defaultTestURL = "https://www.gstatic.com/generate_204"
const defaultTestInterval = 300

// repairURLTestGroups 为 url-test / fallback 补齐测速 URL 与间隔
func repairURLTestGroups(tx *gorm.DB) error {
	var rows []database.ProxyGroup
	if err := tx.Where("type IN ?", []string{"url-test", "fallback"}).Find(&rows).Error; err != nil {
		return err
	}
	interval := defaultTestInterval
	for _, g := range rows {
		updates := map[string]interface{}{}
		if strings.TrimSpace(g.URL) == "" {
			updates["url"] = defaultTestURL
		}
		if g.Interval == nil || *g.Interval < 1 {
			updates["interval"] = interval
		}
		if len(updates) == 0 {
			continue
		}
		if err := tx.Model(&database.ProxyGroup{}).Where("id = ?", g.ID).Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

// migrateCountryGroupNames 将裸码策略组（US）重命名为 美国US，并同步规则 target
func migrateCountryGroupNames(tx *gorm.DB) error {
	// code → 展示名（与 defaults/groups.yaml 一致）
	renames := []struct{ old, neu string }{
		{"US", common.GroupNameDefaultUS},
		{"JP", "日本JP"},
		{"HK", "香港HK"},
		{"TW", "台湾TW"},
		{"SG", "新加坡SG"},
		{"KR", "韩国KR"},
		{"GB", "英国GB"},
		{"PH", "菲律宾PH"},
		{"TR", "土耳其TR"},
	}
	for _, r := range renames {
		var oldCount, newCount int64
		if err := tx.Model(&database.ProxyGroup{}).Where("name = ?", r.old).Count(&oldCount).Error; err != nil {
			return err
		}
		if err := tx.Model(&database.ProxyGroup{}).Where("name = ?", r.neu).Count(&newCount).Error; err != nil {
			return err
		}
		if oldCount > 0 && newCount == 0 {
			if err := tx.Model(&database.ProxyGroup{}).Where("name = ?", r.old).
				Update("name", r.neu).Error; err != nil {
				return err
			}
		}
		// 规则 / 其它组里引用旧名 → 新名
		if err := tx.Model(&database.Rule{}).Where("target = ?", r.old).
			Update("target", r.neu).Error; err != nil {
			return err
		}
	}
	return nil
}

// ensureSystemRules 确保三条系统规则存在（不进 rules.yaml，由代码托管）。
// 缺则创建；仍是旧默认出口/文案时对齐到 systemRuleDefs（不覆盖用户刻意改过的）。
// 同时清理旧版默认「一长串域名广告规则」。
func ensureSystemRules(tx *gorm.DB) error {
	defs := systemRuleDefs()
	for _, def := range defs {
		var count int64
		q := tx.Model(&database.Rule{}).Where("type = ?", def.Type)
		if def.Type == "MATCH" {
			// MATCH payload 恒为空
		} else {
			q = q.Where("payload = ?", def.Payload)
		}
		if err := q.Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := tx.Create(&database.Rule{
				Type:      def.Type,
				Payload:   def.Payload,
				Target:    def.Target,
				Enabled:   true,
				SortOrder: 1, // 顺序由 ensureSystemRuleOrder 收尾
				Note:      def.Note,
				Category:  def.Category,
			}).Error; err != nil {
				return err
			}
			continue
		}
		// 旧默认出口对齐：不覆盖用户明显改过的
		switch def.Type {
		case "MATCH":
			// 仍是「默认走直连/默认走代理」且目标为 直连/日本JP/PROXY/US → 对齐美国US
			if err := tx.Model(&database.Rule{}).
				Where("type = ?", "MATCH").
				Where("target IN ?", []string{common.GroupNameDirect, common.TargetDirect, "日本JP", "PROXY", "US"}).
				Where("note IN ?", []string{"默认走直连", "默认走代理", ""}).
				Updates(map[string]interface{}{
					"target":   def.Target,
					"note":     def.Note,
					"category": def.Category,
				}).Error; err != nil {
				return err
			}
		case "GEOIP":
			if err := tx.Model(&database.Rule{}).
				Where("type = ? AND UPPER(payload) = ?", "GEOIP", "CN").
				Where("category IS NULL OR category = ? OR category IN ?", "", []string{"国内", "广告", "兜底"}).
				Update("category", def.Category).Error; err != nil {
				return err
			}
		case "GEOSITE":
			if err := tx.Model(&database.Rule{}).
				Where("type = ? AND payload = ?", "GEOSITE", "category-ads-all").
				Where("category IS NULL OR category = ? OR category IN ?", "", []string{"广告", "国内", "兜底"}).
				Update("category", def.Category).Error; err != nil {
				return err
			}
		}
	}

	// 清理旧版默认「一长串域名广告规则」，避免列表臃肿（仅匹配我们曾写入的 note）
	return tx.Where(
		"type IN ? AND note IN ? AND payload <> ?",
		[]string{"DOMAIN-SUFFIX", "DOMAIN-KEYWORD"},
		[]string{
			"广告", "广告追踪", "统计", "广告统计", "广告关键词",
			"友盟统计", "CNZZ 统计", "百度联盟", "阿里妈妈", "阿里统计",
			"头条广告", "腾讯广告", "广点通", "小米追踪", "Meta 追踪",
		},
		"category-ads-all",
	).Delete(&database.Rule{}).Error
}

// ensureNamedGroup 若不存在指定名策略组则创建
func ensureNamedGroup(tx *gorm.DB, name, typ string, proxies []string, sortOrder int) error {
	var count int64
	if err := tx.Model(&database.ProxyGroup{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	g := database.ProxyGroup{
		Name:      name,
		Type:      typ,
		Proxies:   mustJSON(proxies),
		Enabled:   true,
		SortOrder: sortOrder,
	}
	// url-test / fallback 默认带测速参数，避免前端显示 ?s、发布配置缺 interval
	if typ == "url-test" || typ == "fallback" {
		interval := defaultTestInterval
		g.URL = defaultTestURL
		g.Interval = &interval
	}
	return tx.Create(&g).Error
}
