package source

import (
	"fmt"
	"strings"
	"time"

	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/database"
	"gorm.io/gorm"
)

// CreateManual 解析并保存一批手工节点分享链接。
func (s *Service) CreateManual(req common.ManualSourceRequest) (common.ManualSourceImportResponse, error) {
	if s.box == nil {
		return common.ManualSourceImportResponse{}, fmt.Errorf("manual source encryption unavailable")
	}
	name := strings.TrimSpace(req.Name)
	content := req.Content
	if name == "" || strings.TrimSpace(content) == "" {
		return common.ManualSourceImportResponse{}, fmt.Errorf("name and content are required")
	}
	parsed, parseStats, err := parseManualContent(content)
	if err != nil {
		return common.ManualSourceImportResponse{}, err
	}
	regionMode := string(common.RegionModeAuto)
	if req.RegionMode != nil && strings.TrimSpace(string(*req.RegionMode)) != "" {
		regionMode = normalizeRegionMode(string(*req.RegionMode))
	}
	region := strings.ToUpper(strings.TrimSpace(string(req.Region)))
	if region == "" {
		region = fallbackRegionCode()
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	enc, err := s.box.Encrypt(content)
	if err != nil {
		return common.ManualSourceImportResponse{}, err
	}
	row := database.Source{
		Name: name, Region: region, Kind: string(common.SourceKindManual), ManualContentEncrypted: enc,
		Enabled: enabled, RegionMode: regionMode, RefreshStatus: string(common.RefreshStatusSuccess),
		LastRefreshAt: func() *time.Time { now := time.Now(); return &now }(),
	}
	stats, err := prepareManualProxies(0, row.Name, row.RegionMode, row.Region, parsed)
	if err != nil {
		return common.ManualSourceImportResponse{}, err
	}
	if len(stats.kept) == 0 {
		return common.ManualSourceImportResponse{}, fmt.Errorf("no valid nodes after parsing")
	}
	var change proxyChangeStats
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		// 节点名和指纹包含源 ID，仅在源创建后再准备一次。
		stats, err = prepareManualProxies(row.ID, row.Name, row.RegionMode, row.Region, parsed)
		if err != nil {
			return err
		}
		change, err = replaceSourceProxies(tx, row.ID, stats.kept)
		return err
	})
	if err != nil {
		return common.ManualSourceImportResponse{}, err
	}
	view, err := s.toView(row)
	if err != nil {
		return common.ManualSourceImportResponse{}, err
	}
	return manualImportResponse(view, parseStats, stats, change), nil
}

// UpdateManual 原子更新手工节点源。解析失败或没有有效节点时保留原内容和节点。
func (s *Service) UpdateManual(id uint, req common.ManualSourceRequest) (common.ManualSourceImportResponse, error) {
	if s.box == nil {
		return common.ManualSourceImportResponse{}, fmt.Errorf("manual source encryption unavailable")
	}
	name := strings.TrimSpace(req.Name)
	content := req.Content
	if name == "" || strings.TrimSpace(content) == "" {
		return common.ManualSourceImportResponse{}, fmt.Errorf("name and content are required")
	}
	parsed, parseStats, err := parseManualContent(content)
	if err != nil {
		return common.ManualSourceImportResponse{}, err
	}
	enc, err := s.box.Encrypt(content)
	if err != nil {
		return common.ManualSourceImportResponse{}, err
	}
	var row database.Source
	var stats refreshStats
	var change proxyChangeStats
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&row, id).Error; err != nil {
			return err
		}
		if normalizeSourceKind(row.Kind) != common.SourceKindManual {
			return fmt.Errorf("source is not a manual node source")
		}
		row.Name = name
		row.ManualContentEncrypted = enc
		if req.RegionMode != nil {
			row.RegionMode = normalizeRegionMode(string(*req.RegionMode))
		}
		if req.Region != "" {
			row.Region = strings.ToUpper(strings.TrimSpace(string(req.Region)))
		}
		if req.Enabled != nil {
			row.Enabled = *req.Enabled
		}
		stats, err = prepareManualProxies(row.ID, row.Name, row.RegionMode, row.Region, parsed)
		if err != nil {
			return err
		}
		if len(stats.kept) == 0 {
			return fmt.Errorf("no valid nodes after parsing")
		}
		change, err = replaceSourceProxies(tx, row.ID, stats.kept)
		if err != nil {
			return err
		}
		now := time.Now()
		return tx.Model(&database.Source{}).Where("id = ?", row.ID).Updates(map[string]interface{}{
			"name": row.Name, "region": row.Region, "region_mode": row.RegionMode, "enabled": row.Enabled,
			"manual_content_encrypted": row.ManualContentEncrypted, "refresh_status": string(common.RefreshStatusSuccess),
			"last_refresh_at": &now, "last_error": "", "traffic_upload": 0, "traffic_download": 0,
			"traffic_total": 0, "traffic_expire": 0,
		}).Error
	})
	if err != nil {
		return common.ManualSourceImportResponse{}, err
	}
	if err := s.db.First(&row, id).Error; err != nil {
		return common.ManualSourceImportResponse{}, err
	}
	view, err := s.toView(row)
	if err != nil {
		return common.ManualSourceImportResponse{}, err
	}
	return manualImportResponse(view, parseStats, stats, change), nil
}

func parseManualContent(content string) ([]ParsedProxy, ParseStats, error) {
	// 手工源仅接受分享链接。不能走订阅解析器，否则 Clash YAML / Base64
	// 会被当成手工输入悄悄保存，违背表单的协议约束。
	parsed, parseStats, err := parseURIListDetailed([]byte(content))
	if err != nil {
		return nil, parseStats, err
	}
	return parsed, parseStats, nil
}

func manualImportResponse(view common.SubscriptionSource, parseStats ParseStats, stats refreshStats, change proxyChangeStats) common.ManualSourceImportResponse {
	return common.ManualSourceImportResponse{
		Source: view, InputTotal: parseStats.Total, Parsed: parseStats.Valid, Previous: change.previous,
		Kept: change.kept, Added: change.added, Removed: change.removed, Modified: change.modified,
		ParseDropped: parseStats.Dropped, RegionCounts: stats.regionCounts,
		RegionConflictTotal: stats.regionConflictTotal, RegionConflicts: stats.regionConflicts,
		RegionConflictOmitted: stats.regionConflictTotal - len(stats.regionConflicts),
	}
}
