package publish

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/regioncatalog"
)

// regionFromProxyName 从 "US-foo" / "jp-bar" 解析地区码
func regionFromProxyName(name string) (string, bool) {
	i := strings.Index(name, "-")
	if i <= 0 || i > 16 {
		return "", false
	}
	region := strings.ToUpper(strings.TrimSpace(name[:i]))
	if region == "" {
		return "", false
	}
	for _, c := range region {
		if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return "", false
		}
	}
	return region, true
}

// proxySourceID 读取节点内部字段 _source_id（发布前剥离）
func proxySourceID(p map[string]interface{}) (uint, bool) {
	if p == nil {
		return 0, false
	}
	v, ok := p["_source_id"]
	if !ok || v == nil {
		return 0, false
	}
	switch t := v.(type) {
	case uint:
		return t, t > 0
	case uint64:
		return uint(t), t > 0
	case int:
		if t > 0 {
			return uint(t), true
		}
	case int64:
		if t > 0 {
			return uint(t), true
		}
	case float64:
		if t > 0 {
			return uint(t), true
		}
	case json.Number:
		i, err := t.Int64()
		if err == nil && i > 0 {
			return uint(i), true
		}
	case string:
		i, err := strconv.ParseUint(strings.TrimSpace(t), 10, 64)
		if err == nil && i > 0 {
			return uint(i), true
		}
	}
	return 0, false
}

// proxySourceName 读取节点内部字段 _source_name（发布前剥离）
func proxySourceName(p map[string]interface{}) string {
	if p == nil {
		return ""
	}
	v, ok := p["_source_name"]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

// refsWantProxies 原始成员是否期望真实节点/地区/其它组（而非纯 DIRECT/REJECT）
func refsWantProxies(refs []string) bool {
	for _, r := range refs {
		u := strings.ToUpper(strings.TrimSpace(r))
		if u == "" || u == common.TargetDirect || u == common.TargetReject {
			continue
		}
		return true
	}
	return false
}

// isEngineOnlyMembers 展开结果是否仅含 DIRECT/REJECT
func isEngineOnlyMembers(expanded []string) bool {
	if len(expanded) == 0 {
		return true
	}
	for _, m := range expanded {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		if !strings.EqualFold(m, common.TargetDirect) && !strings.EqualFold(m, common.TargetReject) {
			return false
		}
	}
	return true
}

// isEmptyProjectedGroup 投影后是否应视为空组（auto/custom 下跳过）
func isEmptyProjectedGroup(refs, expanded []string) bool {
	if len(expanded) == 0 {
		return true
	}
	// 模板只要 DIRECT/REJECT → 保留（直连/拒绝）
	if !refsWantProxies(refs) {
		return false
	}
	// 期望有节点但展开后只剩引擎关键字 → 空地区组
	return isEngineOnlyMembers(expanded)
}

// expandRefs 展开策略组成员
// 支持：
//   - REGION:US / region:jp → 该地区前缀节点
//   - REGION:OTHER          → 非常用国家节点（regions.yaml 中 primary=false）
//   - REGION:* / ALL / *    → 全部节点
//   - SOURCE:名称           → 该订阅源节点（名称大小写不敏感）
//   - SOURCE:id:N           → 该订阅源 ID 的节点
//   - 其它字符串原样保留（组名、DIRECT、具体节点名）
func expandRefs(
	refs []string,
	byRegion map[string][]string,
	bySourceName map[string][]string,
	bySourceID map[uint][]string,
	allNames []string,
) []string {
	out := make([]string, 0, len(refs)+len(allNames))
	seen := map[string]struct{}{}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}

	for _, r := range refs {
		raw := strings.TrimSpace(r)
		if raw == "" {
			continue
		}
		upper := strings.ToUpper(raw)
		switch {
		case upper == common.MemberTokenAll || upper == "*" || upper == "REGION:*" || upper == common.RegionTokenAll:
			for _, n := range allNames {
				add(n)
			}
		case strings.HasPrefix(upper, "SOURCE:"):
			spec := strings.TrimSpace(raw[len("SOURCE:"):])
			if spec == "" {
				continue
			}
			// SOURCE:id:3 / SOURCE:ID:3
			if strings.HasPrefix(strings.ToLower(spec), "id:") {
				idStr := strings.TrimSpace(spec[3:])
				id, err := strconv.ParseUint(idStr, 10, 64)
				if err == nil {
					for _, n := range bySourceID[uint(id)] {
						add(n)
					}
				}
				continue
			}
			for _, n := range bySourceName[strings.ToLower(spec)] {
				add(n)
			}
		case strings.HasPrefix(upper, common.RegionTokenPrefix):
			code := strings.TrimSpace(upper[len(common.RegionTokenPrefix):])
			if code == "" || code == "*" || code == "ALL" {
				for _, n := range allNames {
					add(n)
				}
				continue
			}
			if code == "OTHER" || code == "OTHERS" || code == "REST" {
				// map 遍历顺序不稳定；按地区码排序，保证发布 hash 可复现
				regions := make([]string, 0, len(byRegion))
				for region := range byRegion {
					if regioncatalog.IsPrimary(region) || region == "UNKNOWN" {
						continue
					}
					regions = append(regions, region)
				}
				sort.Strings(regions)
				for _, region := range regions {
					for _, n := range byRegion[region] {
						add(n)
					}
				}
				continue
			}
			for _, n := range byRegion[code] {
				add(n)
			}
		default:
			add(raw)
		}
	}
	return out
}
