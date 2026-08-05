package rule

import (
	"strings"

	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/database"
	"gorm.io/gorm"
)

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
