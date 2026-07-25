package publish

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/database"
)

// proxyIndex 节点索引：按地区 / 订阅源分桶，供策略组成员展开使用。
type proxyIndex struct {
	byRegion     map[string][]string // 地区码 → 节点名（US/JP/HK…）
	bySourceName map[string][]string // lower(源名) → 节点名
	bySourceID   map[uint][]string   // 源 ID → 节点名
	allNames     []string            // 去重后的全部节点名
	warnings     []string            // 收集的可用地区/源提示
}

// indexProxies 遍历节点，按名称前缀地区与订阅源建立索引（Build 阶段 1）。
// 同名节点只保留首个并计入 dupDropped 警告。
func indexProxies(proxies []map[string]interface{}) proxyIndex {
	idx := proxyIndex{
		byRegion:     map[string][]string{},
		bySourceName: map[string][]string{},
		bySourceID:   map[uint][]string{},
		allNames:     make([]string, 0, len(proxies)),
	}
	sourceNameByID := map[uint]string{}
	seenName := map[string]struct{}{}
	dupDropped := 0
	for _, p := range proxies {
		name, _ := p["name"].(string)
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seenName[name]; ok {
			dupDropped++
			continue
		}
		seenName[name] = struct{}{}
		idx.allNames = append(idx.allNames, name)
		if region, ok := regionFromProxyName(name); ok {
			idx.byRegion[region] = append(idx.byRegion[region], name)
		}
		if sid, ok := proxySourceID(p); ok {
			idx.bySourceID[sid] = append(idx.bySourceID[sid], name)
			if sn := proxySourceName(p); sn != "" {
				sourceNameByID[sid] = sn
				key := strings.ToLower(sn)
				idx.bySourceName[key] = append(idx.bySourceName[key], name)
			}
		} else if sn := proxySourceName(p); sn != "" {
			key := strings.ToLower(sn)
			idx.bySourceName[key] = append(idx.bySourceName[key], name)
		}
	}
	if dupDropped > 0 {
		idx.warnings = append(idx.warnings,
			fmt.Sprintf("dropped %d duplicate proxy name(s); check source name suffixes", dupDropped))
	}
	if len(idx.byRegion) == 0 {
		idx.warnings = append(idx.warnings,
			"no region-prefixed proxies found (expected NAME like US-node / JP-node)")
	} else {
		regions := make([]string, 0, len(idx.byRegion))
		for r := range idx.byRegion {
			regions = append(regions, r)
		}
		sort.Strings(regions)
		idx.warnings = append(idx.warnings, "available regions: "+strings.Join(regions, ", "))
	}
	if len(idx.bySourceName) > 0 || len(idx.bySourceID) > 0 {
		idx.warnings = append(idx.warnings,
			"available sources: "+strings.Join(sourceLabels(idx.bySourceName, sourceNameByID), ", "))
	}
	return idx
}

// sourceLabels 生成「源名(id:N)」可读标签列表（按名称排序保证 hash 稳定）。
func sourceLabels(bySourceName map[string][]string, sourceNameByID map[uint]string) []string {
	labels := make([]string, 0, len(bySourceName)+len(sourceNameByID))
	seenLabel := map[string]struct{}{}
	for id, sn := range sourceNameByID {
		label := fmt.Sprintf("%s(id:%d)", sn, id)
		if _, ok := seenLabel[label]; !ok {
			seenLabel[label] = struct{}{}
			labels = append(labels, label)
		}
	}
	for key := range bySourceName {
		// 无 id 映射时补 lower key
		found := false
		for _, sn := range sourceNameByID {
			if strings.ToLower(sn) == key {
				found = true
				break
			}
		}
		if !found {
			if _, ok := seenLabel[key]; !ok {
				seenLabel[key] = struct{}{}
				labels = append(labels, key)
			}
		}
	}
	sort.Strings(labels)
	return labels
}

// projectedGroups 策略组投影结果（Build 阶段 2）。
type projectedGroups struct {
	groups     []map[string]interface{} // 输出用的策略组
	groupNames []string                 // 组名列表
	groupSet   map[string]struct{}      // 合法目标集合（含 DIRECT/REJECT）
	warnings   []string
}

// projectGroups 按 GroupMode 展开策略组成员并做空组处理（Build 阶段 2）。
//   - custom：仅输出 AllowedGroups 白名单内的组
//   - all：空组保留为仅含 DIRECT
//   - auto/其它：剪掉展开后为空的地区组
func projectGroups(
	groups []database.ProxyGroup,
	mode string,
	allowedGroups []string,
	idx proxyIndex,
) projectedGroups {
	allowSet := map[string]struct{}{}
	if mode == "custom" {
		for _, n := range allowedGroups {
			n = strings.TrimSpace(n)
			if n != "" {
				allowSet[n] = struct{}{}
			}
		}
	}

	out := projectedGroups{
		groups:     make([]map[string]interface{}, 0, len(groups)),
		groupNames: make([]string, 0, len(groups)),
		groupSet: map[string]struct{}{
			common.TargetDirect: {},
			common.TargetReject: {},
		},
	}
	for _, grp := range groups {
		name := strings.TrimSpace(grp.Name)
		if name == "" {
			continue
		}
		if mode == "custom" {
			if _, ok := allowSet[name]; !ok {
				continue
			}
		}
		var refs []string
		_ = json.Unmarshal([]byte(grp.Proxies), &refs)
		expanded := expandRefs(refs, idx.byRegion, idx.bySourceName, idx.bySourceID, idx.allNames)
		// 空地区组：展开后无成员，或本应有节点/地区引用却只剩 DIRECT/REJECT
		// （「直连」「拒绝」这类本身只有 DIRECT/REJECT 的组会保留）
		if isEmptyProjectedGroup(refs, expanded) {
			switch mode {
			case "all":
				out.warnings = append(out.warnings,
					fmt.Sprintf("proxy group %q empty after filter; fallback to DIRECT", name))
				expanded = []string{common.TargetDirect}
			default: // auto / custom：剪掉空组
				out.warnings = append(out.warnings,
					fmt.Sprintf("proxy group %q skipped: empty after expansion", name))
				continue
			}
		}
		item := map[string]interface{}{
			"name":    name,
			"type":    grp.Type,
			"proxies": expanded,
		}
		if grp.URL != "" {
			item["url"] = grp.URL
		}
		if grp.Interval != nil {
			item["interval"] = *grp.Interval
		}
		out.groups = append(out.groups, item)
		out.groupNames = append(out.groupNames, name)
		out.groupSet[name] = struct{}{}
	}
	return out
}
