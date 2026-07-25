package rule

import (
	"fmt"
	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/database"
	"gorm.io/gorm"
	"strings"
)

// Service 规则与策略组草稿
type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) ListRules() (common.RuleListResponse, error) {
	var rows []database.Rule
	if err := s.db.Order("sort_order asc, id asc").Find(&rows).Error; err != nil {
		return common.RuleListResponse{}, err
	}
	items := make([]common.Rule, 0, len(rows))
	for _, r := range rows {
		items = append(items, toRule(r))
	}
	return common.RuleListResponse{Items: items}, nil
}

func (s *Service) validateRuleTarget(target string) error {
	target = strings.TrimSpace(target)
	if target == "" || target == common.TargetDirect || target == common.TargetReject {
		if target == "" {
			return fmt.Errorf("rule target required")
		}
		return nil
	}
	var count int64
	if err := s.db.Model(&database.ProxyGroup{}).Where("name = ?", target).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		var groups int64
		if err := s.db.Model(&database.ProxyGroup{}).Count(&groups).Error; err != nil {
			return err
		}
		if groups == 0 {
			return nil
		}
		return fmt.Errorf("rule target %q does not reference an existing proxy group", target)
	}
	return nil
}

func (s *Service) validateRuleRequest(req common.UpsertRuleRequest) (string, string, string, error) {
	typ := strings.ToUpper(strings.TrimSpace(req.Type))
	payload := strings.TrimSpace(req.Payload)
	target := strings.TrimSpace(req.Target)
	if err := validateRuleFields(typ, payload, target, req.Note, req.Category); err != nil {
		return "", "", "", err
	}
	if err := validateRule(typ, payload, target); err != nil {
		return "", "", "", err
	}
	if err := s.validateRuleTarget(target); err != nil {
		return "", "", "", err
	}
	return typ, payload, target, nil
}

func (s *Service) CreateRule(req common.UpsertRuleRequest) (common.Rule, error) {
	typ, payload, target, err := s.validateRuleRequest(req)
	if err != nil {
		return common.Rule{}, err
	}
	// 不允许再建第二条系统规则
	if isSystemSeedRule(typ, payload) {
		var n int64
		q := s.db.Model(&database.Rule{}).Where("type = ?", typ)
		if typ != "MATCH" {
			q = q.Where("payload = ?", payload)
		}
		if err := q.Count(&n).Error; err != nil {
			return common.Rule{}, err
		}
		if n > 0 {
			return common.Rule{}, fmt.Errorf("系统规则已存在，不可重复添加")
		}
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	cat := strings.TrimSpace(req.Category)
	if isSystemSeedRule(typ, payload) {
		cat = systemRuleCategory
	}

	var created database.Rule
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 插在国内 GEOIP / MATCH 之前（与批量导入一致）
		order := 100
		if req.SortOrder != nil {
			order = *req.SortOrder
		} else {
			var existing []database.Rule
			if err := tx.Find(&existing).Error; err != nil {
				return err
			}
			anchor, hasAnchor := resolveInsertAnchor(existing)
			if hasAnchor {
				if err := tx.Model(&database.Rule{}).
					Where("sort_order >= ?", anchor).
					Update("sort_order", gorm.Expr("sort_order + ?", 10)).Error; err != nil {
					return err
				}
			}
			order = anchor
		}
		row := database.Rule{
			Type:      typ,
			Payload:   payload,
			Target:    target,
			Enabled:   enabled,
			SortOrder: order,
			Note:      strings.TrimSpace(req.Note),
			Category:  cat,
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		created = row
		return ensureSystemRuleOrder(tx)
	})
	if err != nil {
		return common.Rule{}, err
	}
	// 重排后读回最终 sort_order
	_ = s.db.First(&created, created.ID)
	return toRule(created), nil
}

func (s *Service) UpdateRule(id uint, req common.UpsertRuleRequest) (common.Rule, error) {
	var row database.Rule
	if err := s.db.First(&row, id).Error; err != nil {
		return common.Rule{}, err
	}
	sys := isSystemSeedRule(row.Type, row.Payload)
	if sys {
		// 系统规则：类型/匹配/分类锁死，只允许改出口、备注、启用
		if err := validateRule(row.Type, row.Payload, req.Target); err != nil {
			return common.Rule{}, err
		}
		if err := validateRuleFields(row.Type, row.Payload, req.Target, req.Note, systemRuleCategory); err != nil {
			return common.Rule{}, err
		}
		if err := s.validateRuleTarget(req.Target); err != nil {
			return common.Rule{}, err
		}
		row.Target = strings.TrimSpace(req.Target)
		row.Note = strings.TrimSpace(req.Note)
		row.Category = systemRuleCategory
		if req.Enabled != nil {
			row.Enabled = *req.Enabled
		}
		// 忽略 sortOrder / type / payload 修改
		if err := s.db.Save(&row).Error; err != nil {
			return common.Rule{}, err
		}
		return toRule(row), nil
	}

	typ, payload, target, err := s.validateRuleRequest(req)
	if err != nil {
		return common.Rule{}, err
	}
	// 业务规则不可改成「第二条」系统规则身份
	newType := typ
	newPayload := payload
	if isSystemSeedRule(newType, newPayload) {
		return common.Rule{}, fmt.Errorf("不可将业务规则改为系统规则（广告/国内/兜底由系统托管）")
	}
	row.Type = newType
	row.Payload = newPayload
	row.Target = target
	row.Note = strings.TrimSpace(req.Note)
	row.Category = strings.TrimSpace(req.Category)
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if req.SortOrder != nil {
		row.SortOrder = *req.SortOrder
	}
	if err := s.db.Save(&row).Error; err != nil {
		return common.Rule{}, err
	}
	// 若用户改了 sortOrder，仍钉死系统三段位置
	_ = s.db.Transaction(func(tx *gorm.DB) error {
		return ensureSystemRuleOrder(tx)
	})
	_ = s.db.First(&row, id)
	return toRule(row), nil
}

func (s *Service) DeleteRule(id uint) error {
	var row database.Rule
	if err := s.db.First(&row, id).Error; err != nil {
		return err
	}
	if isSystemSeedRule(row.Type, row.Payload) {
		return fmt.Errorf("系统规则不可删除（广告/国内 GEOIP/MATCH 兜底）")
	}
	return s.db.Delete(&database.Rule{}, id).Error
}

func (s *Service) ReorderRules(ids []uint) error {
	if len(ids) > maxReorderIDs {
		return fmt.Errorf("too many orderedIds; maximum is %d", maxReorderIDs)
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var rows []database.Rule
		if err := tx.Order("sort_order asc, id asc").Find(&rows).Error; err != nil {
			return err
		}
		if len(ids) != len(rows) {
			return fmt.Errorf("orderedIds must include every rule exactly once")
		}
		expected := make(map[uint]struct{}, len(rows))
		for _, row := range rows {
			expected[row.ID] = struct{}{}
		}
		seen := make(map[uint]struct{}, len(ids))
		for _, id := range ids {
			if _, ok := expected[id]; !ok {
				return fmt.Errorf("orderedIds contains unknown rule %d", id)
			}
			if _, ok := seen[id]; ok {
				return fmt.Errorf("orderedIds contains duplicate rule %d", id)
			}
			seen[id] = struct{}{}
		}
		for i, id := range ids {
			if err := tx.Model(&database.Rule{}).Where("id = ?", id).Update("sort_order", (i+1)*10).Error; err != nil {
				return err
			}
		}
		// 拖拽后仍钉死：广告最先、国内 GEOIP 倒数第二、MATCH 最后
		return ensureSystemRuleOrder(tx)
	})
}

// EnabledRules 启用规则按顺序
func (s *Service) EnabledRules() ([]database.Rule, error) {
	var rows []database.Rule
	err := s.db.Where("enabled = ?", true).Order("sort_order asc, id asc").Find(&rows).Error
	return rows, err
}
