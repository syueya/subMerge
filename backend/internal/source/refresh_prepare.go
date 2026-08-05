package source

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/applog"
)

// preparedProxy 过滤 + 地区识别 + 改名后的待入库节点
type preparedProxy struct {
	name        string
	region      string
	typ         string
	server      string
	port        int
	fingerprint string
	rawJSON     string
}

// refreshStats 刷新过程中的过滤/识别统计（用于响应与日志）
type refreshStats struct {
	kept                []preparedProxy
	skipped             int
	filterDropped       map[string]int
	filteredNames       []string
	filteredTotal       int
	regionCounts        map[string]int
	regionConflictTotal int
	regionConflicts     []common.RegionConflict
}

// maxDetectSamples 地区识别样本日志上限（节点过多时避免刷屏）
const maxDetectSamples = 40

// maxFallbackSamples 回退地区样本日志上限
const maxFallbackSamples = 20

// maxRegionConflictSamples 地区冲突响应样本上限
const maxRegionConflictSamples = 20

// maxFilteredNameSamples 过滤节点名称响应样本上限
const maxFilteredNameSamples = 1000

// maxFilteredNameLen 单个过滤节点名称的最大 rune 数
const maxFilteredNameLen = 255

// prepareProxies 对解析出的节点做「信息节点丢弃 → 过滤规则 → 地区识别 → 改名去重 → 序列化」，
// 返回待入库节点与统计。无副作用（除日志），DB 写入见 persistRefresh。
func prepareProxies(
	sourceID uint,
	sourceName, mode, defaultRegion string,
	proxies []ParsedProxy,
	filter *CompiledFilter,
) (refreshStats, error) {
	stats := refreshStats{
		kept:            make([]preparedProxy, 0, len(proxies)),
		filterDropped:   map[string]int{},
		filteredNames:   make([]string, 0, 16),
		regionCounts:    map[string]int{},
		regionConflicts: make([]common.RegionConflict, 0, maxRegionConflictSamples),
	}
	detectMethodCounts := map[string]int{}
	detectSamples := make([]string, 0, maxDetectSamples)
	fallbackSamples := make([]string, 0, maxFallbackSamples)
	usedNames := map[string]struct{}{}

	for _, p := range proxies {
		// 始终丢弃明显信息节点（即使该源过滤规则是旧的）
		if IsInfoNodeName(p.Name) {
			stats.dropFiltered(sourceID, p, "info_node", "信息节点")
			continue
		}
		if ok, reason := filter.ShouldKeep(p); !ok {
			if reason == "" {
				reason = "filtered"
			}
			label := reason
			if reason == "name excluded" {
				label = ""
			}
			stats.dropFiltered(sourceID, p, reason, label)
			continue
		}

		resolved := ResolveRegionDetailed(p.Name, mode, defaultRegion)
		region := resolved.Region
		if region == "" {
			region = strings.ToUpper(strings.TrimSpace(defaultRegion))
			resolved.UsedFallback = true
		}

		// 统计识别方式
		methodKey := resolved.Detect.Method
		if resolved.UsedFallback {
			methodKey = "fallback"
		}
		detectMethodCounts[methodKey]++
		stats.regionCounts[region]++
		if resolved.Detect.Conflict {
			stats.regionConflictTotal++
			if len(stats.regionConflicts) < maxRegionConflictSamples {
				stats.regionConflicts = append(stats.regionConflicts, common.RegionConflict{
					Name:           cleanFilteredName(p.Name),
					FlagRegion:     resolved.Detect.FlagRegion,
					FlagMatched:    resolved.Detect.FlagMatched,
					KeywordRegion:  resolved.Detect.KeywordRegion,
					KeywordMatched: resolved.Detect.KeywordMatched,
					ResolvedRegion: region,
				})
			}
		}

		// 逐节点样本：原名 → 地区 (方式:命中) → 最终名
		if len(detectSamples) < maxDetectSamples {
			matchInfo := resolved.Detect.Method
			if resolved.Detect.Matched != "" {
				matchInfo = resolved.Detect.Method + ":" + resolved.Detect.Matched
			}
			if resolved.UsedFallback {
				matchInfo = "fallback→" + region
			}
			namePreview := uniqueProxyName(FormatProxyName(p.Name, region, sourceName), map[string]struct{}{})
			detectSamples = append(detectSamples, fmt.Sprintf("%q → %s (%s) → %q",
				p.Name, region, matchInfo, namePreview))
		}
		if resolved.UsedFallback && len(fallbackSamples) < maxFallbackSamples {
			fallbackSamples = append(fallbackSamples, p.Name)
		}

		name := uniqueProxyName(FormatProxyName(p.Name, region, sourceName), usedNames)
		usedNames[name] = struct{}{}

		raw := map[string]interface{}{}
		for k, v := range p.Raw {
			raw[k] = v
		}
		raw["name"] = name
		rawJSON, err := json.Marshal(raw)
		if err != nil {
			return refreshStats{}, err
		}
		// 指纹基于身份字段，不含展示名，避免改名后丢失 enabled
		stats.kept = append(stats.kept, preparedProxy{
			name:        name,
			region:      region,
			typ:         p.Type,
			server:      p.Server,
			port:        p.Port,
			fingerprint: ProxyFingerprint(p),
			rawJSON:     string(rawJSON),
		})
	}

	// 地区自动识别汇总日志
	logRegionDetectSummary(sourceID, sourceName, mode, defaultRegion,
		len(stats.kept), stats.regionCounts, detectMethodCounts, detectSamples, fallbackSamples)

	return stats, nil
}

// dropFiltered 记录一个被过滤掉的节点（统计 + 有限明细 + debug 日志）。
// extraTag 非空时追加到明细标签（如 "信息节点"、具体过滤原因）。
func (st *refreshStats) dropFiltered(sourceID uint, p ParsedProxy, reason, extraTag string) {
	st.skipped++
	st.filteredTotal++
	st.filterDropped[reason]++
	label := cleanFilteredName(p.Name)
	if label == "" {
		label = "(无名称)"
	}
	if extraTag != "" {
		label = label + " [" + cleanFilteredName(extraTag) + "]"
	}
	if len(st.filteredNames) < maxFilteredNameSamples {
		st.filteredNames = append(st.filteredNames, label)
	}
	applog.Debug("[refresh] filter drop source id=%d reason=%q type=%s port=%d",
		sourceID, reason, p.Type, p.Port)
}

func cleanFilteredName(raw string) string {
	raw = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, strings.TrimSpace(raw))
	if len([]rune(raw)) > maxFilteredNameLen {
		return string([]rune(raw)[:maxFilteredNameLen]) + "..."
	}
	return raw
}
