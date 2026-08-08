package rule

import (
	"strings"

	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/database"
	"gorm.io/gorm"
)

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
		// 仅补齐系统规则的分类，不修改已有 MATCH 的出口
		switch def.Type {
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
