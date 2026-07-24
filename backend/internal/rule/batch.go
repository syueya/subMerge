package rule

import (
	"fmt"
	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/database"
	"gorm.io/gorm"
	"strings"
)

// BatchImportRules 批量导入规则（一行一条），插在 GEOIP CN / MATCH 之前。
// 解析错误不阻断整批：合法行照常写入，错误汇总返回。
func (s *Service) BatchImportRules(req common.BatchImportRulesRequest) (common.BatchImportRulesResponse, error) {
	parsed, parseErrs := parseBatchImportText(req.Text, req.DefaultType, req.DefaultTarget, req.DefaultNote, req.DefaultCategory)
	res := common.BatchImportRulesResponse{
		Errors: parseErrs,
		Items:  []common.Rule{},
	}
	if len(parsed) == 0 {
		res.Skipped = len(parseErrs)
		return res, nil
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 已有规则：跳过 type+payload+target 完全相同的
		var existing []database.Rule
		if err := tx.Find(&existing).Error; err != nil {
			return err
		}
		seen := make(map[string]struct{}, len(existing))
		for _, r := range existing {
			key := strings.ToUpper(r.Type) + "\x00" + strings.ToLower(r.Payload) + "\x00" + r.Target
			seen[key] = struct{}{}
		}

		anchor, hasAnchor := resolveInsertAnchor(existing)

		// 给新规则腾位置：锚点及之后整体后移
		shift := len(parsed) * 10
		if hasAnchor && shift > 0 {
			if err := tx.Model(&database.Rule{}).
				Where("sort_order >= ?", anchor).
				Update("sort_order", gorm.Expr("sort_order + ?", shift)).Error; err != nil {
				return err
			}
		}

		created := make([]common.Rule, 0, len(parsed))
		skipped := 0
		for i, p := range parsed {
			if isSystemSeedRule(p.Type, p.Payload) {
				skipped++
				res.Errors = append(res.Errors, fmt.Sprintf("第%d行：系统规则由程序托管，已跳过", p.LineNo))
				continue
			}
			key := strings.ToUpper(p.Type) + "\x00" + strings.ToLower(p.Payload) + "\x00" + p.Target
			if _, dup := seen[key]; dup {
				skipped++
				res.Errors = append(res.Errors, fmt.Sprintf("第%d行：已存在，已跳过", p.LineNo))
				continue
			}
			row := database.Rule{
				Type:      p.Type,
				Payload:   p.Payload,
				Target:    p.Target,
				Enabled:   enabled,
				SortOrder: anchor + i*10,
				Note:      p.Note,
				Category:  p.Category,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			seen[key] = struct{}{}
			created = append(created, toRule(row))
		}
		res.Created = len(created)
		res.Skipped = skipped
		res.Items = created
		// 导入后钉死：广告最先、国内 GEOIP 倒数第二、MATCH 最后
		return ensureSystemRuleOrder(tx)
	})
	if err != nil {
		return common.BatchImportRulesResponse{}, err
	}
	return res, nil
}

// uniqRuleIDs 去重并去掉 0
func uniqRuleIDs(ids []uint) []uint {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[uint]struct{}, len(ids))
	uniq := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	return uniq
}

// BatchUpdateRulesTarget 批量修改规则目标出口（仅改 target，类型/匹配/顺序不变）
func (s *Service) BatchUpdateRulesTarget(ids []uint, target string) (int, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return 0, fmt.Errorf("rule target required")
	}
	uniq := uniqRuleIDs(ids)
	if len(uniq) == 0 {
		return 0, nil
	}
	res := s.db.Model(&database.Rule{}).Where("id IN ?", uniq).Update("target", target)
	if res.Error != nil {
		return 0, res.Error
	}
	return int(res.RowsAffected), nil
}

// BatchUpdateRulesEnabled 批量启用/禁用
func (s *Service) BatchUpdateRulesEnabled(ids []uint, enabled bool) (int, error) {
	uniq := uniqRuleIDs(ids)
	if len(uniq) == 0 {
		return 0, nil
	}
	res := s.db.Model(&database.Rule{}).Where("id IN ?", uniq).Update("enabled", enabled)
	if res.Error != nil {
		return 0, res.Error
	}
	return int(res.RowsAffected), nil
}

// BatchUpdateRulesCategory 批量修改业务分类（允许空字符串表示未分类；跳过系统规则）
func (s *Service) BatchUpdateRulesCategory(ids []uint, category string) (int, error) {
	category = strings.TrimSpace(category)
	uniq := uniqRuleIDs(ids)
	if len(uniq) == 0 {
		return 0, nil
	}
	// 不改系统规则分类
	res := s.db.Model(&database.Rule{}).
		Where("id IN ?", uniq).
		Where("NOT (type = ? OR (type = ? AND UPPER(payload) = ?) OR (type = ? AND payload = ?))",
			"MATCH", "GEOIP", "CN", "GEOSITE", "category-ads-all").
		Update("category", category)
	if res.Error != nil {
		return 0, res.Error
	}
	return int(res.RowsAffected), nil
}

// BatchDeleteRules 批量删除规则（系统规则跳过不删）
func (s *Service) BatchDeleteRules(ids []uint) (int, error) {
	uniq := uniqRuleIDs(ids)
	if len(uniq) == 0 {
		return 0, nil
	}
	res := s.db.
		Where("id IN ?", uniq).
		Where("NOT (type = ? OR (type = ? AND UPPER(payload) = ?) OR (type = ? AND payload = ?))",
			"MATCH", "GEOIP", "CN", "GEOSITE", "category-ads-all").
		Delete(&database.Rule{})
	if res.Error != nil {
		return 0, res.Error
	}
	return int(res.RowsAffected), nil
}
