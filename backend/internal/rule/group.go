package rule

import (
	"strings"

	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/database"
	"gorm.io/gorm"
)

func (s *Service) ListGroups() (common.ProxyGroupListResponse, error) {
	var rows []database.ProxyGroup
	if err := s.db.Order("sort_order asc, id asc").Find(&rows).Error; err != nil {
		return common.ProxyGroupListResponse{}, err
	}
	items := make([]common.ProxyGroup, 0, len(rows))
	for _, r := range rows {
		items = append(items, toGroup(r))
	}
	return common.ProxyGroupListResponse{Items: items}, nil
}

func (s *Service) CreateGroup(req common.UpsertProxyGroupRequest) (common.ProxyGroup, error) {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	order := 0
	if req.SortOrder != nil {
		order = *req.SortOrder
	}
	if req.Proxies == nil {
		req.Proxies = []string{}
	}
	row := database.ProxyGroup{
		Name:      strings.TrimSpace(req.Name),
		Type:      req.Type,
		Proxies:   mustJSON(req.Proxies),
		URL:       req.URL,
		Interval:  req.Interval,
		Enabled:   enabled,
		SortOrder: order,
	}
	if err := s.db.Create(&row).Error; err != nil {
		return common.ProxyGroup{}, err
	}
	return toGroup(row), nil
}

func (s *Service) UpdateGroup(id uint, req common.UpsertProxyGroupRequest) (common.ProxyGroup, error) {
	var row database.ProxyGroup
	if err := s.db.First(&row, id).Error; err != nil {
		return common.ProxyGroup{}, err
	}
	row.Name = strings.TrimSpace(req.Name)
	row.Type = req.Type
	if req.Proxies == nil {
		req.Proxies = []string{}
	}
	row.Proxies = mustJSON(req.Proxies)
	row.URL = req.URL
	row.Interval = req.Interval
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if req.SortOrder != nil {
		row.SortOrder = *req.SortOrder
	}
	if err := s.db.Save(&row).Error; err != nil {
		return common.ProxyGroup{}, err
	}
	return toGroup(row), nil
}

// DeleteGroup 删除策略组；cascadeRules 为 true 时一并删除指向该组的规则
func (s *Service) DeleteGroup(id uint, cascadeRules bool) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var row database.ProxyGroup
		if err := tx.First(&row, id).Error; err != nil {
			return err
		}
		if cascadeRules {
			if err := tx.Unscoped().Where("target = ?", row.Name).Delete(&database.Rule{}).Error; err != nil {
				return err
			}
		}
		// 硬删除：Name 带唯一索引，软删除残留行会导致同名重建冲突
		return tx.Unscoped().Delete(&database.ProxyGroup{}, id).Error
	})
}

// EnabledGroups 启用策略组
func (s *Service) EnabledGroups() ([]database.ProxyGroup, error) {
	var rows []database.ProxyGroup
	err := s.db.Where("enabled = ?", true).Order("sort_order asc, id asc").Find(&rows).Error
	return rows, err
}
