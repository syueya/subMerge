package source

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/submerge/submerge/backend/internal/applog"

	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/database"
	"gorm.io/gorm"
)

// Refresh 拉取上游订阅并更新节点；失败时保留旧快照。
// 网络拉取不持全局锁，仅用 per-source 防重入，避免阻塞其它源/节点操作。
func (s *Service) Refresh(id uint) (common.RefreshSourceResponse, error) {
	s.refreshMu.Lock()
	if _, ok := s.refreshing[id]; ok {
		s.refreshMu.Unlock()
		return common.RefreshSourceResponse{}, fmt.Errorf("source refresh already in progress")
	}
	s.refreshing[id] = struct{}{}
	s.refreshMu.Unlock()
	defer func() {
		s.refreshMu.Lock()
		delete(s.refreshing, id)
		s.refreshMu.Unlock()
	}()

	var row database.Source
	if err := s.db.First(&row, id).Error; err != nil {
		return common.RefreshSourceResponse{}, err
	}
	row.RefreshStatus = string(common.RefreshStatusRunning)
	if err := s.db.Save(&row).Error; err != nil {
		return common.RefreshSourceResponse{}, fmt.Errorf("mark source refresh running: %w", err)
	}

	rawURL, err := s.box.Decrypt(row.URLEncrypted)
	if err != nil {
		return s.failRefresh(row, fmt.Errorf("decrypt url: %w", err))
	}

	// 网络 I/O 在锁外（对齐 Clash Verge：GET + UA + 解析 Subscription-Userinfo）
	fetched, err := s.fetch(rawURL)
	if err != nil {
		return s.failRefresh(row, err)
	}
	body := fetched.Body

	proxies, parseStats, err := ParseClashProxiesDetailed(body)
	if err != nil {
		return s.failRefresh(row, err)
	}

	filter, err := CompileFilter(FilterOptions{
		ExcludeNameRegex: row.ExcludeNameRegex,
		ExcludeServers:   row.ExcludeServers,
		IncludeNameRegex: row.IncludeNameRegex,
	})
	if err != nil {
		return s.failRefresh(row, err)
	}

	mode := normalizeRegionMode(row.RegionMode)
	applog.Info("[refresh] source id=%d name=%q mode=%s defaultRegion=%s upstreamBytes=%d ua=%q",
		row.ID, row.Name, mode, row.Region, len(body), s.userAgent)

	// 过滤 + 地区识别 + 改名去重（无 DB 副作用）
	stats, err := prepareProxies(row.ID, row.Name, mode, row.Region, proxies, filter)
	if err != nil {
		return s.failRefresh(row, err)
	}
	if len(stats.kept) == 0 {
		return s.failRefresh(row, fmt.Errorf(
			"no proxies left after filtering (upstream=%d parsed=%d skipped=%d parseDropped=%v filter=%v)",
			parseStats.Total, parseStats.Valid, stats.skipped, parseStats.Dropped, stats.filterDropped,
		))
	}

	// 事务重写节点 + 更新源状态/流量；失败保留旧快照
	oldCount, err := s.persistRefresh(&row, stats.kept, body, fetched.UserInfo)
	if err != nil {
		return s.failRefresh(row, err)
	}

	view, err := s.toView(row)
	if err != nil {
		return common.RefreshSourceResponse{}, err
	}
	applog.Info("[refresh] ok source id=%d name=%q kept=%d removed=%d skipped=%d upstream=%d parsed=%d filterDropped=%v",
		row.ID, row.Name, len(stats.kept), oldCount, stats.skipped, parseStats.Total, parseStats.Valid, stats.filterDropped)
	if len(stats.filteredNames) > 0 {
		applog.Info("[refresh] filtered source id=%d name=%q total=%d samples=%d omitted=%d reasons=%v",
			row.ID, row.Name, stats.filteredTotal, len(stats.filteredNames),
			stats.filteredTotal-len(stats.filteredNames), stats.filterDropped)
	}

	return common.RefreshSourceResponse{
		Source:               view,
		UpstreamTotal:        parseStats.Total,
		Parsed:               parseStats.Valid,
		Added:                len(stats.kept),
		Removed:              oldCount,
		Skipped:              stats.skipped,
		ParseDropped:         parseStats.Dropped,
		FilterDropped:        stats.filterDropped,
		FilteredNames:        stats.filteredNames,
		FilteredNamesOmitted: stats.filteredTotal - len(stats.filteredNames),
		RegionCounts:         stats.regionCounts,
	}, nil
}

// persistRefresh 在单事务里整批重写某源的节点并更新源状态/流量。
// 返回被替换掉的旧节点数量（用于统计 removed）。保留旧节点的 enabled 状态。
func (s *Service) persistRefresh(
	row *database.Source,
	kept []preparedProxy,
	body []byte,
	userInfo SubscriptionUserInfo,
) (int, error) {
	// 旧节点 fingerprint → enabled，用于保留用户禁用状态
	var oldProxies []database.Proxy
	if err := s.db.Where("source_id = ?", row.ID).Find(&oldProxies).Error; err != nil {
		return 0, fmt.Errorf("load existing proxies: %w", err)
	}
	oldEnabled := map[string]bool{}
	for _, op := range oldProxies {
		if op.Fingerprint != "" {
			oldEnabled[op.Fingerprint] = op.Enabled
		}
	}
	oldCount := len(oldProxies)

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("source_id = ?", row.ID).Delete(&database.Proxy{}).Error; err != nil {
			return err
		}
		for _, p := range kept {
			enabled := true
			if prev, ok := oldEnabled[p.fingerprint]; ok {
				enabled = prev
			}
			if err := tx.Create(&database.Proxy{
				SourceID:    row.ID,
				Name:        p.name,
				Region:      p.region,
				Type:        p.typ,
				Server:      p.server,
				Port:        p.port,
				Enabled:     enabled,
				Fingerprint: p.fingerprint,
				RawJSON:     p.rawJSON,
			}).Error; err != nil {
				return err
			}
		}
		now := time.Now()
		row.RefreshStatus = string(common.RefreshStatusSuccess)
		row.LastRefreshAt = &now
		row.LastError = ""
		row.SnapshotYAML = string(body)
		// 有 userinfo 头则覆盖；无头则清零，避免展示过期残留
		if userInfo.HasAny() {
			row.TrafficUpload = userInfo.Upload
			row.TrafficDownload = userInfo.Download
			row.TrafficTotal = userInfo.Total
			row.TrafficExpire = userInfo.Expire
		} else {
			row.TrafficUpload = 0
			row.TrafficDownload = 0
			row.TrafficTotal = 0
			row.TrafficExpire = 0
		}
		return tx.Save(row).Error
	})
	if err != nil {
		return 0, err
	}
	return oldCount, nil
}

// ResetStuckRefresh 启动时把残留的 running 状态复位为 failed。
// 进程在刷新中途崩溃时，per-source 内存锁随进程消失，但 DB 里的 running
// 无人复位，会导致该源永久显示「刷新中」。启动时调用一次即可清理。
func (s *Service) ResetStuckRefresh() error {
	res := s.db.Model(&database.Source{}).
		Where("refresh_status = ?", string(common.RefreshStatusRunning)).
		Updates(map[string]interface{}{
			"refresh_status": string(common.RefreshStatusFailed),
			"last_error":     "refresh interrupted by restart",
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected > 0 {
		applog.Warn("[startup] reset %d stuck refreshing source(s) to failed", res.RowsAffected)
	}
	return nil
}

// RefreshAll 刷新所有启用源（串行；单源失败不影响后续）
func (s *Service) RefreshAll() common.RefreshAllResponse {
	var rows []database.Source
	if err := s.db.Where("enabled = ?", true).Order("id asc").Find(&rows).Error; err != nil {
		applog.Error("[refresh-all] list enabled sources failed: %v", err)
		return common.RefreshAllResponse{}
	}
	applog.Info("[refresh-all] start total=%d", len(rows))
	out := common.RefreshAllResponse{
		Total:   len(rows),
		Results: make([]common.RefreshAllItem, 0, len(rows)),
	}
	for _, r := range rows {
		item := common.RefreshAllItem{SourceID: r.ID, Name: r.Name}
		res, err := s.Refresh(r.ID)
		if err != nil {
			item.OK = false
			item.Error = err.Error()
			out.Failed++
			applog.Warn("[refresh-all] fail id=%d name=%q: %v", r.ID, r.Name, err)
		} else {
			item.OK = true
			item.Added = res.Added
			item.Skipped = res.Skipped
			out.OK++
		}
		out.Results = append(out.Results, item)
	}
	applog.Info("[refresh-all] done ok=%d failed=%d total=%d", out.OK, out.Failed, out.Total)
	return out
}

func (s *Service) failRefresh(row database.Source, cause error) (common.RefreshSourceResponse, error) {
	// 入库 / 对外展示的错误必须脱敏；日志里 cause 本身若经 fetch 也已 masked
	errText := cause.Error()
	applog.Error("[refresh] fail source id=%d name=%q: %s", row.ID, row.Name, errText)
	row.RefreshStatus = string(common.RefreshStatusFailed)
	row.LastError = errText
	if err := s.db.Save(&row).Error; err != nil {
		return common.RefreshSourceResponse{}, fmt.Errorf("refresh failed (%s); save failure state: %w", errText, err)
	}
	view, err := s.toView(row)
	if err != nil {
		return common.RefreshSourceResponse{}, err
	}
	return common.RefreshSourceResponse{Source: view}, cause
}

// logRegionDetectSummary 输出地区自动识别汇总与样本
func logRegionDetectSummary(
	sourceID uint,
	sourceName, mode, defaultRegion string,
	kept int,
	regionCounts, methodCounts map[string]int,
	samples, fallbackSamples []string,
) {
	applog.Info("[region-detect] source id=%d name=%q mode=%s default=%s kept=%d regions=%s methods=%s",
		sourceID, sourceName, mode, defaultRegion, kept,
		formatCountMap(regionCounts), formatCountMap(methodCounts))

	if len(samples) > 0 {
		// 分批打印，单行过长不易读
		const batch = 8
		for i := 0; i < len(samples); i += batch {
			end := i + batch
			if end > len(samples) {
				end = len(samples)
			}
			applog.Debug("[region-detect] samples id=%d (%d-%d/%d): %s",
				sourceID, i+1, end, len(samples), strings.Join(samples[i:end], " | "))
		}
	}
	if n := methodCounts["fallback"]; n > 0 {
		msg := fmt.Sprintf("[region-detect] fallback id=%d count=%d default=%s", sourceID, n, defaultRegion)
		if len(fallbackSamples) > 0 {
			msg += " names=[" + strings.Join(fallbackSamples, ", ") + "]"
			if n > len(fallbackSamples) {
				msg += fmt.Sprintf(" …(+%d)", n-len(fallbackSamples))
			}
		}
		applog.Warn("%s", msg)
	}
}

func formatCountMap(m map[string]int) string {
	if len(m) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, m[k]))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}
