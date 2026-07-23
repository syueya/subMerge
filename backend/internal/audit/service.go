package audit

import (
	"time"

	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/applog"
	"github.com/submerge/submerge/backend/internal/database"
	"gorm.io/gorm"
)

// Service 审计日志
type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Log(actor, action, resource, detail, ip string) {
	if actor == "" {
		actor = "system"
	}
	// 审计属安全相关数据，写失败不阻断主流程，但必须记录以便排查
	if err := s.db.Create(&database.AuditLog{
		Actor:    actor,
		Action:   action,
		Resource: resource,
		Detail:   detail,
		IP:       ip,
	}).Error; err != nil {
		applog.Error("[audit] write log failed actor=%q action=%q: %v", actor, action, err)
	}
}

func (s *Service) List(limit, offset int) (common.AuditListResponse, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var total int64
	var rows []database.AuditLog
	// Count 与 Find 各用独立查询，避免在同一链式 *gorm.DB 上复用导致子句相互污染
	if err := s.db.Model(&database.AuditLog{}).Count(&total).Error; err != nil {
		return common.AuditListResponse{}, err
	}
	if err := s.db.Model(&database.AuditLog{}).Order("id desc").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return common.AuditListResponse{}, err
	}
	items := make([]common.AuditLog, 0, len(rows))
	for _, r := range rows {
		items = append(items, common.AuditLog{
			ID:        r.ID,
			Actor:     r.Actor,
			Action:    r.Action,
			Resource:  r.Resource,
			Detail:    r.Detail,
			IP:        r.IP,
			CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return common.AuditListResponse{Items: items, Total: total}, nil
}
