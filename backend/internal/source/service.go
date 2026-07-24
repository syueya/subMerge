package source

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/config"
	"github.com/submerge/submerge/backend/internal/crypto"
	"github.com/submerge/submerge/backend/internal/database"
	"gorm.io/gorm"
)

// Service 订阅源管理
type Service struct {
	db         *gorm.DB
	box        *crypto.Box
	httpClient *http.Client
	maxBytes   int64
	userAgent  string
	// refreshMu 仅保护 refreshing 集合，避免同一源并发刷新
	refreshMu  sync.Mutex
	refreshing map[uint]struct{}
}

func NewService(db *gorm.DB, box *crypto.Box, timeout time.Duration, maxBytes int64) *Service {
	return NewServiceWithUA(db, box, timeout, maxBytes, "")
}

func NewServiceWithUA(db *gorm.DB, box *crypto.Box, timeout time.Duration, maxBytes int64, userAgent string) *Service {
	ua := strings.TrimSpace(userAgent)
	if ua == "" {
		ua = config.DefaultSourceFetchUA
	}
	return &Service{
		db:         db,
		box:        box,
		httpClient: newFetchHTTPClient(timeout),
		maxBytes:   maxBytes,
		userAgent:  ua,
		refreshing: make(map[uint]struct{}),
	}
}

func (s *Service) List() (common.SourceListResponse, error) {
	var rows []database.Source
	if err := s.db.Order("id asc").Find(&rows).Error; err != nil {
		return common.SourceListResponse{}, err
	}
	items := make([]common.SubscriptionSource, 0, len(rows))
	for _, r := range rows {
		item, err := s.toView(r)
		if err != nil {
			return common.SourceListResponse{}, err
		}
		items = append(items, item)
	}
	return common.SourceListResponse{Items: items}, nil
}

func (s *Service) Create(req common.CreateSourceRequest) (common.SubscriptionSource, error) {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	enc, err := s.box.Encrypt(strings.TrimSpace(req.URL))
	if err != nil {
		return common.SubscriptionSource{}, err
	}

	regionMode := string(common.RegionModeAuto)
	if req.RegionMode != nil && strings.TrimSpace(string(*req.RegionMode)) != "" {
		regionMode = normalizeRegionMode(string(*req.RegionMode))
	}

	// 自动模式未显式指定地区时回退 UNKNOWN，避免默认成 US
	if strings.TrimSpace(string(req.Region)) == "" {
		req.Region = common.Region(fallbackRegionCode())
	}

	defFilter := DefaultFilterOptions()
	excludeName := defFilter.ExcludeNameRegex
	if req.ExcludeNameRegex != nil {
		excludeName = *req.ExcludeNameRegex
	}
	excludeServers := defFilter.ExcludeServers
	if req.ExcludeServers != nil {
		excludeServers = *req.ExcludeServers
	}
	includeName := defFilter.IncludeNameRegex
	if req.IncludeNameRegex != nil {
		includeName = *req.IncludeNameRegex
	}

	// 预先校验正则，避免写入后刷新才失败
	if _, err := CompileFilter(FilterOptions{
		ExcludeNameRegex: excludeName,
		ExcludeServers:   excludeServers,
		IncludeNameRegex: includeName,
	}); err != nil {
		return common.SubscriptionSource{}, err
	}

	row := database.Source{
		Name:             strings.TrimSpace(req.Name),
		Region:           string(req.Region),
		URLEncrypted:     enc,
		Enabled:          enabled,
		RegionMode:       regionMode,
		ExcludeNameRegex: excludeName,
		ExcludeServers:   excludeServers,
		IncludeNameRegex: includeName,
		RefreshStatus:    string(common.RefreshStatusIdle),
	}
	if err := s.db.Create(&row).Error; err != nil {
		return common.SubscriptionSource{}, err
	}
	return s.toView(row)
}

func (s *Service) Update(id uint, req common.UpdateSourceRequest) (common.SubscriptionSource, error) {
	var row database.Source
	if err := s.db.First(&row, id).Error; err != nil {
		return common.SubscriptionSource{}, err
	}
	oldRegion := row.Region
	oldMode := row.RegionMode
	oldName := row.Name

	if req.Name != nil {
		row.Name = strings.TrimSpace(*req.Name)
	}
	if req.Region != nil {
		row.Region = string(*req.Region)
	}
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if req.RegionMode != nil {
		row.RegionMode = normalizeRegionMode(string(*req.RegionMode))
	}
	if req.ExcludeNameRegex != nil {
		row.ExcludeNameRegex = *req.ExcludeNameRegex
	}
	if req.ExcludeServers != nil {
		row.ExcludeServers = *req.ExcludeServers
	}
	if req.IncludeNameRegex != nil {
		row.IncludeNameRegex = *req.IncludeNameRegex
	}
	if req.URL != nil && strings.TrimSpace(*req.URL) != "" {
		enc, err := s.box.Encrypt(strings.TrimSpace(*req.URL))
		if err != nil {
			return common.SubscriptionSource{}, err
		}
		row.URLEncrypted = enc
	}

	if _, err := CompileFilter(FilterOptions{
		ExcludeNameRegex: row.ExcludeNameRegex,
		ExcludeServers:   row.ExcludeServers,
		IncludeNameRegex: row.IncludeNameRegex,
	}); err != nil {
		return common.SubscriptionSource{}, err
	}

	// fixed 改地区 / 切到 fixed / 改源名称：重写子节点名
	rewriteChildren := false
	if normalizeRegionMode(row.RegionMode) == string(common.RegionModeFixed) {
		if oldRegion != row.Region || normalizeRegionMode(oldMode) != string(common.RegionModeFixed) {
			rewriteChildren = true
		}
	}
	if oldName != row.Name {
		rewriteChildren = true
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&row).Error; err != nil {
			return err
		}
		if !rewriteChildren {
			return nil
		}
		var proxies []database.Proxy
		if err := tx.Where("source_id = ?", row.ID).Find(&proxies).Error; err != nil {
			return err
		}
		used := make(map[string]struct{}, len(proxies))
		for _, proxy := range proxies {
			core := stripSourceSuffix(proxy.Name, oldName)
			core = stripRegionPrefix(core)
			region := row.Region
			if normalizeRegionMode(row.RegionMode) != string(common.RegionModeFixed) && proxy.Region != "" {
				region = proxy.Region
			}
			name := uniqueProxyName(FormatProxyName(core, region, row.Name), used)
			used[name] = struct{}{}
			raw := map[string]interface{}{}
			if err := json.Unmarshal([]byte(proxy.RawJSON), &raw); err != nil {
				return err
			}
			raw["name"] = name
			rawJSON, err := json.Marshal(raw)
			if err != nil {
				return err
			}
			updates := map[string]interface{}{"name": name, "raw_json": string(rawJSON)}
			if normalizeRegionMode(row.RegionMode) == string(common.RegionModeFixed) {
				updates["region"] = row.Region
			}
			if err := tx.Model(&database.Proxy{}).Where("id = ?", proxy.ID).Updates(updates).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return common.SubscriptionSource{}, err
	}
	return s.toView(row)
}

func (s *Service) Delete(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("source_id = ?", id).Delete(&database.Proxy{}).Error; err != nil {
			return err
		}
		return tx.Delete(&database.Source{}, id).Error
	})
}

// DeleteMany 批量删除订阅源及其节点
func (s *Service) DeleteMany(ids []uint) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	// 去重
	seen := map[uint]struct{}{}
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
	if len(uniq) == 0 {
		return 0, nil
	}

	var deleted int64
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("source_id IN ?", uniq).Delete(&database.Proxy{}).Error; err != nil {
			return err
		}
		res := tx.Where("id IN ?", uniq).Delete(&database.Source{})
		if res.Error != nil {
			return res.Error
		}
		deleted = res.RowsAffected
		return nil
	})
	return int(deleted), err
}

func (s *Service) ListProxies(sourceID *uint) (common.ProxyListResponse, error) {
	q := s.db.Model(&database.Proxy{}).Order("region asc, name asc")
	if sourceID != nil {
		q = q.Where("source_id = ?", *sourceID)
	}
	var rows []database.Proxy
	if err := q.Find(&rows).Error; err != nil {
		return common.ProxyListResponse{}, err
	}
	items := make([]common.ProxyNode, 0, len(rows))
	for _, r := range rows {
		items = append(items, toProxyNode(r.ID, r.SourceID, r.Name, r.Region, r.Type, r.Server, r.Port, r.Enabled))
	}
	return common.ProxyListResponse{Items: items}, nil
}

// UpdateProxy 更新单个节点启用状态
func (s *Service) UpdateProxy(id uint, enabled bool) (common.ProxyNode, error) {
	var row database.Proxy
	if err := s.db.First(&row, id).Error; err != nil {
		return common.ProxyNode{}, err
	}
	row.Enabled = enabled
	if err := s.db.Save(&row).Error; err != nil {
		return common.ProxyNode{}, err
	}
	return toProxyNode(row.ID, row.SourceID, row.Name, row.Region, row.Type, row.Server, row.Port, row.Enabled), nil
}

// BatchUpdateProxies 批量更新启用状态
func (s *Service) BatchUpdateProxies(ids []uint, enabled bool) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	res := s.db.Model(&database.Proxy{}).Where("id IN ?", ids).Update("enabled", enabled)
	if res.Error != nil {
		return 0, res.Error
	}
	return int(res.RowsAffected), nil
}

// EnabledProxies 返回所有启用源的启用节点 raw map
func (s *Service) EnabledProxies() ([]map[string]interface{}, error) {
	return s.EnabledProxiesBySourceIDs(nil)
}

// EnabledProxiesBySourceIDs 返回指定启用源的启用节点 raw map。
// sourceIDs 为空时等同全部启用源；非空时仅包含所列且仍启用的源。
//
// 会注入内部字段（发布生成器使用，输出 YAML 前会剥离）：
//   - _source_id   uint   订阅源 ID
//   - _source_name string 订阅源名称（用于 SOURCE:名称 展开）
func (s *Service) EnabledProxiesBySourceIDs(sourceIDs []uint) ([]map[string]interface{}, error) {
	// 必须 Model(Proxy)：自定义结果结构体若直接 Find，GORM 会当成表名 proxy_with_sources
	type proxyWithSource struct {
		database.Proxy
		SourceName string `gorm:"column:source_name"`
	}
	q := s.db.Model(&database.Proxy{}).
		Select("proxies.*, sources.name AS source_name").
		Joins("JOIN sources ON sources.id = proxies.source_id AND sources.deleted_at IS NULL AND sources.enabled = ?", true).
		Where("proxies.enabled = ?", true)
	if len(sourceIDs) > 0 {
		q = q.Where("proxies.source_id IN ?", sourceIDs)
	}
	var rows []proxyWithSource
	// Scan 绑定自定义列；Find 在部分 GORM 版本会对非模型结构体推错表名
	err := q.Order("proxies.region asc, proxies.name asc").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(r.RawJSON), &m); err != nil {
			continue
		}
		// 内部元数据：生成器 SOURCE: 展开用；不会写入 Clash YAML
		m["_source_id"] = r.SourceID
		m["_source_name"] = r.SourceName
		out = append(out, m)
	}
	return out, nil
}

func uniqueProxyName(base string, used map[string]struct{}) string {
	if _, exists := used[base]; !exists {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if _, exists := used[candidate]; !exists {
			return candidate
		}
	}
}

// stripSourceSuffix 若 name 以 -{源后缀} 结尾则剥掉（用于改名后重写）
func stripSourceSuffix(name, sourceName string) string {
	suffix := SanitizeSourceSuffix(sourceName)
	if suffix == "" || name == "" {
		return name
	}
	// 精确后缀（中文等大小写不变）
	if strings.HasSuffix(name, "-"+suffix) {
		return strings.TrimRight(strings.TrimSuffix(name, "-"+suffix), "-")
	}
	// ASCII 大小写不敏感
	upper := strings.ToUpper(name)
	tail := "-" + strings.ToUpper(suffix)
	if strings.HasSuffix(upper, tail) {
		return strings.TrimRight(name[:len(name)-len(tail)], "-")
	}
	return name
}

func normalizeRegionMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(common.RegionModeFixed):
		return string(common.RegionModeFixed)
	default:
		return string(common.RegionModeAuto)
	}
}

func (s *Service) toView(r database.Source) (common.SubscriptionSource, error) {
	urlPlain := ""
	if r.URLEncrypted != "" {
		if plain, err := s.box.Decrypt(r.URLEncrypted); err == nil {
			urlPlain = plain
		}
	}
	var count int64
	_ = s.db.Model(&database.Proxy{}).Where("source_id = ?", r.ID).Count(&count).Error
	mode := common.RegionMode(normalizeRegionMode(r.RegionMode))
	v := common.SubscriptionSource{
		ID:               r.ID,
		Name:             r.Name,
		Region:           common.Region(r.Region),
		URLMasked:        crypto.MaskURL(urlPlain),
		Enabled:          r.Enabled,
		RegionMode:       mode,
		ExcludeNameRegex: r.ExcludeNameRegex,
		ExcludeServers:   r.ExcludeServers,
		IncludeNameRegex: r.IncludeNameRegex,
		RefreshStatus:    common.RefreshStatus(r.RefreshStatus),
		ProxyCount:       int(count),
		TrafficUpload:    r.TrafficUpload,
		TrafficDownload:  r.TrafficDownload,
		TrafficTotal:     r.TrafficTotal,
		TrafficExpire:    r.TrafficExpire,
		CreatedAt:        r.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:        r.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if r.LastRefreshAt != nil {
		s := r.LastRefreshAt.UTC().Format(time.RFC3339)
		v.LastRefreshAt = &s
	}
	if r.LastError != "" {
		e := r.LastError
		v.LastError = &e
	}
	return v, nil
}
