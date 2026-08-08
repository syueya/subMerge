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
	if err := s.db.Model(&database.Source{}).
		Where("id = ?", id).
		Update("refresh_status", string(common.RefreshStatusRunning)).Error; err != nil {
		return common.RefreshSourceResponse{}, fmt.Errorf("mark source refresh running: %w", err)
	}
	var running database.Source
	if err := s.db.First(&running, id).Error; err != nil {
		return common.RefreshSourceResponse{}, err
	}
	row = running

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
	_, _, userAgent := s.RuntimeOptions()
	applog.Info("[refresh] 拉取并解析完成 source id=%d name=%q mode=%s defaultRegion=%s upstreamBytes=%d ua=%q",
		row.ID, row.Name, mode, row.Region, len(body), userAgent)

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
	changeStats, err := s.persistRefresh(&row, stats.kept, body, fetched.UserInfo)
	if err != nil {
		return s.failRefresh(row, err)
	}

	viewRow := row
	if err := s.db.First(&viewRow, row.ID).Error; err != nil {
		return common.RefreshSourceResponse{}, err
	}
	view, err := s.toView(viewRow)
	if err != nil {
		return common.RefreshSourceResponse{}, err
	}
	applog.Info("[refresh] 刷新成功 source id=%d name=%q previous=%d kept=%d added=%d removed=%d modified=%d skipped=%d upstream=%d parsed=%d filterDropped=%v regionConflicts=%d",
		row.ID, row.Name, changeStats.previous, changeStats.kept, changeStats.added,
		changeStats.removed, changeStats.modified, stats.skipped, parseStats.Total, parseStats.Valid,
		stats.filterDropped, stats.regionConflictTotal)
	if len(stats.filteredNames) > 0 {
		applog.Debug("[refresh] 已过滤节点 source id=%d name=%q total=%d samples=%d omitted=%d reasons=%v",
			row.ID, row.Name, stats.filteredTotal, len(stats.filteredNames),
			stats.filteredTotal-len(stats.filteredNames), stats.filterDropped)
	}
	if stats.regionConflictTotal > 0 {
		applog.Debug("[region-detect] 地区识别冲突 source id=%d total=%d samples=%d",
			row.ID, stats.regionConflictTotal, len(stats.regionConflicts))
		for i, conflict := range stats.regionConflicts {
			applog.Debug("[region-detect] 冲突样例=%d name=%q flag=%s keyword=%s resolved=%s",
				i+1, conflict.Name, conflict.FlagRegion, conflict.KeywordRegion, conflict.ResolvedRegion)
		}
	}

	return common.RefreshSourceResponse{
		Source:                view,
		UpstreamTotal:         parseStats.Total,
		Parsed:                parseStats.Valid,
		Previous:              changeStats.previous,
		Kept:                  changeStats.kept,
		Added:                 changeStats.added,
		Removed:               changeStats.removed,
		Modified:              changeStats.modified,
		Skipped:               stats.skipped,
		ParseDropped:          parseStats.Dropped,
		FilterDropped:         stats.filterDropped,
		FilteredNames:         stats.filteredNames,
		FilteredNamesOmitted:  stats.filteredTotal - len(stats.filteredNames),
		RegionCounts:          stats.regionCounts,
		RegionConflictTotal:   stats.regionConflictTotal,
		RegionConflicts:       stats.regionConflicts,
		RegionConflictOmitted: stats.regionConflictTotal - len(stats.regionConflicts),
	}, nil
}

// proxyChangeStats 新旧快照差分。指纹标识同一节点；enabled 不计入 modified。
type proxyChangeStats struct {
	previous int
	kept     int
	added    int
	removed  int
	modified int
}

// persistRefresh 在单事务里整批重写某源的节点并更新源状态/流量。
// 顺带基于旧快照计算 previous/kept/added/removed/modified，避免重复查库。
func (s *Service) persistRefresh(
	row *database.Source,
	kept []preparedProxy,
	body []byte,
	userInfo SubscriptionUserInfo,
) (proxyChangeStats, error) {
	var changeStats proxyChangeStats
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var current database.Source
		if err := tx.First(&current, row.ID).Error; err != nil {
			return err
		}
		var oldProxies []database.Proxy
		if err := tx.Where("source_id = ?", row.ID).Find(&oldProxies).Error; err != nil {
			return fmt.Errorf("load existing proxies: %w", err)
		}
		changeStats = diffProxySnapshots(oldProxies, kept)

		oldEnabled := map[string]bool{}
		for _, op := range oldProxies {
			if op.Fingerprint != "" {
				oldEnabled[op.Fingerprint] = op.Enabled
			}
		}
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
		updates := map[string]interface{}{
			"refresh_status":  string(common.RefreshStatusSuccess),
			"last_refresh_at": &now,
			"last_error":      "",
			"snapshot_yaml":   string(body),
		}
		if userInfo.HasAny() {
			updates["traffic_upload"] = userInfo.Upload
			updates["traffic_download"] = userInfo.Download
			updates["traffic_total"] = userInfo.Total
			updates["traffic_expire"] = userInfo.Expire
		} else {
			updates["traffic_upload"] = 0
			updates["traffic_download"] = 0
			updates["traffic_total"] = 0
			updates["traffic_expire"] = 0
		}
		return tx.Model(&database.Source{}).Where("id = ?", current.ID).Updates(updates).Error
	})
	if err != nil {
		return proxyChangeStats{}, err
	}
	return changeStats, nil
}

func diffProxySnapshots(old []database.Proxy, next []preparedProxy) proxyChangeStats {
	stats := proxyChangeStats{previous: len(old)}
	oldByFingerprint := make(map[string]database.Proxy, len(old))
	for _, proxy := range old {
		oldByFingerprint[proxy.Fingerprint] = proxy
	}
	seen := make(map[string]struct{}, len(next))
	for _, proxy := range next {
		seen[proxy.fingerprint] = struct{}{}
		previous, ok := oldByFingerprint[proxy.fingerprint]
		if !ok {
			stats.added++
			continue
		}
		if proxyContentChanged(previous, proxy) {
			stats.modified++
		} else {
			stats.kept++
		}
	}
	for _, proxy := range old {
		if _, ok := seen[proxy.Fingerprint]; !ok {
			stats.removed++
		}
	}
	return stats
}

func proxyContentChanged(old database.Proxy, next preparedProxy) bool {
	return old.Name != next.name ||
		old.Region != next.region ||
		old.Type != next.typ ||
		old.Server != next.server ||
		old.Port != next.port ||
		old.RawJSON != next.rawJSON
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
		applog.Warn("[startup] 启动时重置 %d 个卡住的刷新任务为失败", res.RowsAffected)
	}
	return nil
}

// RefreshAll 刷新所有启用源（串行；单源失败不影响后续）
func (s *Service) RefreshAll() common.RefreshAllResponse {
	var rows []database.Source
	if err := s.db.Where("enabled = ?", true).Order("id asc").Find(&rows).Error; err != nil {
		applog.Error("[refresh-all] 获取启用订阅源失败: %v", err)
		return common.RefreshAllResponse{}
	}
	applog.Info("[refresh-all] 开始批量刷新 total=%d", len(rows))
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
		} else {
			item.OK = true
			item.Previous = res.Previous
			item.Kept = res.Kept
			item.Added = res.Added
			item.Removed = res.Removed
			item.Modified = res.Modified
			item.Skipped = res.Skipped
			item.RegionConflictTotal = res.RegionConflictTotal
			out.OK++
		}

		out.Results = append(out.Results, item)
	}
	applog.Info("[refresh-all] 批量刷新完成 ok=%d failed=%d total=%d", out.OK, out.Failed, out.Total)
	return out
}

func (s *Service) failRefresh(row database.Source, cause error) (common.RefreshSourceResponse, error) {
	// 入库 / 对外展示的错误必须脱敏；日志里 cause 本身若经 fetch 也已 masked
	errText := cause.Error()
	applog.Error("[refresh] 刷新失败 source id=%d name=%q: %s", row.ID, row.Name, errText)
	if err := s.db.Model(&database.Source{}).Where("id = ?", row.ID).Updates(map[string]interface{}{
		"refresh_status": string(common.RefreshStatusFailed),
		"last_error":     errText,
	}).Error; err != nil {
		return common.RefreshSourceResponse{}, fmt.Errorf("refresh failed (%s); save failure state: %w", errText, err)
	}
	var current database.Source
	if err := s.db.First(&current, row.ID).Error; err != nil {
		return common.RefreshSourceResponse{}, err
	}
	view, err := s.toView(current)
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
	applog.Info("[region-detect] 地区识别汇总 source id=%d name=%q mode=%s default=%s kept=%d regions=%s methods=%s",
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
			applog.Debug("[region-detect] 地区样本 id=%d (%d-%d/%d): %s",
				sourceID, i+1, end, len(samples), strings.Join(samples[i:end], " | "))
		}
	}
	if n := methodCounts["fallback"]; n > 0 {
		msg := fmt.Sprintf("[region-detect] 地区识别回退 id=%d count=%d default=%s", sourceID, n, defaultRegion)
		if len(fallbackSamples) > 0 {
			msg += " names=[" + strings.Join(fallbackSamples, ", ") + "]"
			if n > len(fallbackSamples) {
				msg += fmt.Sprintf(" …(+%d)", n-len(fallbackSamples))
			}
		}
		applog.Debug("%s", msg)
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
