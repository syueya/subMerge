package publish

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/submerge/submerge/backend/internal/database"
	"github.com/submerge/submerge/backend/internal/regioncatalog"
	"gopkg.in/yaml.v3"
)

// Generator 生成 Clash Meta 配置
type Generator struct{}

func NewGenerator() *Generator {
	return &Generator{}
}

// BuildInput 生成输入
	type BuildInput struct {
		Proxies []map[string]interface{}
		Groups  []database.ProxyGroup
		Rules   []database.Rule
		// GroupMode 策略组投影：
		//   auto（默认）— 展开后无真实节点的组整组删除；规则目标缺失 → DIRECT
		//   all          — 空组保留为仅含 DIRECT（兼容旧 Lenient）；规则目标缺失 → DIRECT
		//   custom       — 仅输出 AllowedGroups 白名单中的组；空组跳过；规则缺失 → DIRECT
		// 空字符串按 auto 处理。
		GroupMode string
		// AllowedGroups 仅 custom 模式生效；组名白名单（大小写敏感，与面板组名一致）
		AllowedGroups []string
		// Lenient 已废弃：true 时等价于 GroupMode=all（兼容旧测试/调用）
		Lenient bool
	}

	func resolveGroupMode(in BuildInput) string {
		m := strings.ToLower(strings.TrimSpace(in.GroupMode))
		switch m {
		case "all", "custom", "auto":
			return m
		}
		if in.Lenient {
			return "all"
		}
		return "auto"
	}

// BuildResult 生成结果
type BuildResult struct {
	YAML       string
	Hash       string
	ProxyCount int
	RuleCount  int
	GroupNames []string
	Warnings   []string
}

// Build 合并节点、策略组与规则，输出 YAML
func (g *Generator) Build(in BuildInput) (*BuildResult, error) {
	warnings := []string{}
	if len(in.Proxies) == 0 {
		return nil, fmt.Errorf("no proxies available; refresh sources first")
	}

	// 按节点名前缀收集各地区节点，例如 US-xxx / JP-xxx / HK-xxx
	byRegion := map[string][]string{}
	allNames := make([]string, 0, len(in.Proxies))
	seenName := map[string]struct{}{}
	dupDropped := 0
	for _, p := range in.Proxies {
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
		allNames = append(allNames, name)
		if region, ok := regionFromProxyName(name); ok {
			byRegion[region] = append(byRegion[region], name)
		}
	}
	if dupDropped > 0 {
		warnings = append(warnings, fmt.Sprintf("dropped %d duplicate proxy name(s); check source name suffixes", dupDropped))
	}

	if len(byRegion) == 0 {
		warnings = append(warnings, "no region-prefixed proxies found (expected NAME like US-node / JP-node)")
	} else {
		regions := make([]string, 0, len(byRegion))
		for r := range byRegion {
			regions = append(regions, r)
		}
		sort.Strings(regions)
		warnings = append(warnings, "available regions: "+strings.Join(regions, ", "))
	}

	// 策略组投影：按 GroupMode 决定空组/白名单行为
		mode := resolveGroupMode(in)
		allowSet := map[string]struct{}{}
		if mode == "custom" {
			for _, n := range in.AllowedGroups {
				n = strings.TrimSpace(n)
				if n != "" {
					allowSet[n] = struct{}{}
				}
			}
		}
			// 全量发布（auto 且无源过滤语义）时，规则目标缺失应硬失败；
			// 投影场景（auto 剪组 / all 占位 / custom）规则目标缺失：
			//   优先回退「节点选择」，否则 DIRECT。
			// 约定：Publish 全量 build 用 GroupMode="" → auto，且期望严格规则校验。
			// 订阅投影显式传 auto/all/custom。为区分「全量严格」与「投影 auto」，
			// 用 Projected 标志… 简化：仅当 GroupMode 为空且 !Lenient 时严格；
			// 显式 auto/all/custom 均允许规则回退。
		strictRules := strings.TrimSpace(in.GroupMode) == "" && !in.Lenient

		groupNames := make([]string, 0, len(in.Groups))
		groupSet := map[string]struct{}{
			"DIRECT": {},
			"REJECT": {},
		}
		proxyGroups := make([]map[string]interface{}, 0, len(in.Groups))
		for _, grp := range in.Groups {
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
			expanded := expandRefs(refs, byRegion, allNames)
			// 空地区组：展开后无成员，或本应有节点/地区引用却只剩 DIRECT/REJECT
			// （「直连」「拒绝」这类本身只有 DIRECT/REJECT 的组会保留）
			if isEmptyProjectedGroup(refs, expanded) {
				switch mode {
				case "all":
					warnings = append(warnings,
						fmt.Sprintf("proxy group %q empty after filter; fallback to DIRECT", name))
					expanded = []string{"DIRECT"}
				default: // auto / custom：剪掉空组
					warnings = append(warnings,
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
			proxyGroups = append(proxyGroups, item)
			groupNames = append(groupNames, name)
			groupSet[name] = struct{}{}
		}
		if len(proxyGroups) == 0 {
			return nil, fmt.Errorf("no usable proxy groups after expansion; add nodes or fix REGION:xx refs")
		}

			// 规则
			ruleLines := make([]string, 0, len(in.Rules))
			hasMatch := false
			// 投影场景下缺失目标的优先回退：节点选择 → DIRECT
			const fallbackProxyGroup = "节点选择"
			for _, r := range in.Rules {
				target := strings.TrimSpace(r.Target)
				if _, ok := groupSet[target]; !ok {
					if strictRules {
						return nil, fmt.Errorf("rule target %q not found in proxy-groups/DIRECT/REJECT", target)
					}
					fallback := "DIRECT"
					if _, ok := groupSet[fallbackProxyGroup]; ok {
						fallback = fallbackProxyGroup
					}
					warnings = append(warnings,
						fmt.Sprintf("rule target %q missing after filter; fallback to %s", target, fallback))
					r.Target = fallback
				}
				line := formatRule(r)
				if line == "" {
					continue
				}
				if r.Type == "MATCH" {
					hasMatch = true
				}
				ruleLines = append(ruleLines, line)
			}
	if len(ruleLines) == 0 {
		return nil, fmt.Errorf("no enabled rules")
	}
	if !hasMatch {
		return nil, fmt.Errorf("MATCH rule is required and must be last")
	}
	// MATCH 必须在末尾
	last := ruleLines[len(ruleLines)-1]
	if !strings.HasPrefix(last, "MATCH,") {
		return nil, fmt.Errorf("MATCH rule must be the last rule")
	}

		// 规范化节点字段，避免 yaml 数字变成 float、缺 type 等导致 mihomo 解析失败
		proxies := sanitizeProxiesForMeta(in.Proxies)

		// 订阅向配置：只下发节点 / 策略组 / 规则与少量无冲突的全局项。
		// 刻意不写 mixed-port / external-controller / allow-lan 等客户端本机设置，
		// 避免 Clash Verge 导入订阅时覆盖本机端口导致测速 error/timeout。
		doc := map[string]interface{}{
			"mode":           "rule",
			"ipv6":           true,
			"unified-delay":  true,
			"tcp-concurrent": true,
			"proxies":        proxies,
			"proxy-groups":   proxyGroups,
			"rules":          ruleLines,
		}

		out, err := yaml.Marshal(doc)
		if err != nil {
			return nil, err
		}
		// 文件头注释：方便识别与 mihomo 导入
		header := "# SubMerge generated subscription for Clash Meta / mihomo / Clash Verge\n" +
			"# proxies + proxy-groups + rules only; client keeps local port/DNS settings\n"
		yamlText := header + string(out)
	sum := sha256.Sum256([]byte(yamlText))
	return &BuildResult{
		YAML:       yamlText,
		Hash:       hex.EncodeToString(sum[:]),
		ProxyCount: len(proxies),
		RuleCount:  len(ruleLines),
		GroupNames: groupNames,
		Warnings:   warnings,
	}, nil
}

// sanitizeProxiesForMeta 清洗节点 map，保证 mihomo 可解析
func sanitizeProxiesForMeta(in []map[string]interface{}) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(in))
	for _, raw := range in {
		if raw == nil {
			continue
		}
		m := make(map[string]interface{}, len(raw)+4)
		for k, v := range raw {
			m[k] = v
		}
		name := strings.TrimSpace(fmt.Sprint(m["name"]))
		typ := strings.TrimSpace(strings.ToLower(fmt.Sprint(m["type"])))
		server := strings.TrimSpace(fmt.Sprint(m["server"]))
		if name == "" || typ == "" || typ == "<nil>" || server == "" || server == "<nil>" {
			continue
		}
		m["name"] = name
		m["type"] = typ
		m["server"] = server
		if port, ok := coerceYAMLPort(m["port"]); ok {
			m["port"] = port
		} else {
			continue
		}
		// 常见布尔字段规范化
		for _, bk := range []string{"udp", "tls", "skip-cert-verify", "tfo", "mptcp"} {
			if v, exists := m[bk]; exists {
				m[bk] = coerceYAMLBool(v)
			}
		}
		// 旧式 ws-path / ws-headers 与 ws-opts 并存时，Meta 以 ws-opts 为准；去掉重复字段避免解析歧义
		normalizeTransportOpts(m)
		out = append(out, m)
	}
	return out
}

// normalizeTransportOpts 统一传输层字段：保留 *-opts，去掉并行的旧字段
func normalizeTransportOpts(m map[string]interface{}) {
	if _, has := m["ws-opts"]; has {
		delete(m, "ws-path")
		delete(m, "ws-headers")
	} else if path, ok := m["ws-path"]; ok || m["ws-headers"] != nil {
		// 仅有旧字段时提升为 ws-opts
		opts := map[string]interface{}{}
		if ok && path != nil && fmt.Sprint(path) != "" && fmt.Sprint(path) != "<nil>" {
			opts["path"] = path
		}
		if h, ok := m["ws-headers"]; ok && h != nil {
			opts["headers"] = h
		}
		if len(opts) > 0 {
			m["ws-opts"] = opts
		}
		delete(m, "ws-path")
		delete(m, "ws-headers")
	}
	if _, has := m["http-opts"]; has {
		delete(m, "http-path")
		delete(m, "http-headers")
	}
	if _, has := m["h2-opts"]; has {
		delete(m, "h2-path")
		delete(m, "h2-headers")
	}
	// hysteria2：ports / mport 并存时保留 ports（Meta 主字段），去掉重复 mport
	if strings.EqualFold(fmt.Sprint(m["type"]), "hysteria2") {
		if _, hasPorts := m["ports"]; hasPorts {
			delete(m, "mport")
		} else if mp, ok := m["mport"]; ok && mp != nil {
			m["ports"] = mp
			delete(m, "mport")
		}
	}
}

func coerceYAMLPort(v interface{}) (int, bool) {
	switch t := v.(type) {
	case int:
		if t >= 1 && t <= 65535 {
			return t, true
		}
	case int64:
		if t >= 1 && t <= 65535 {
			return int(t), true
		}
	case float64:
		if t == float64(int(t)) {
			p := int(t)
			if p >= 1 && p <= 65535 {
				return p, true
			}
		}
	case string:
		s := strings.TrimSpace(t)
		n := 0
		for _, c := range s {
			if c < '0' || c > '9' {
				return 0, false
			}
			n = n*10 + int(c-'0')
			if n > 65535 {
				return 0, false
			}
		}
		if n >= 1 {
			return n, true
		}
	}
	return 0, false
}

func coerceYAMLBool(v interface{}) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		return s == "true" || s == "1" || s == "yes" || s == "on"
	case int:
		return t != 0
	case float64:
		return t != 0
	default:
		return false
	}
}

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

// refsWantProxies 原始成员是否期望真实节点/地区/其它组（而非纯 DIRECT/REJECT）
func refsWantProxies(refs []string) bool {
	for _, r := range refs {
		u := strings.ToUpper(strings.TrimSpace(r))
		if u == "" || u == "DIRECT" || u == "REJECT" {
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
		if !strings.EqualFold(m, "DIRECT") && !strings.EqualFold(m, "REJECT") {
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
	//   - 其它字符串原样保留（组名、DIRECT、具体节点名）
	func expandRefs(refs []string, byRegion map[string][]string, allNames []string) []string {
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
		case upper == "ALL" || upper == "*" || upper == "REGION:*" || upper == "REGION:ALL":
			for _, n := range allNames {
				add(n)
			}
		case strings.HasPrefix(upper, "REGION:"):
			code := strings.TrimSpace(upper[len("REGION:"):])
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

func formatRule(r database.Rule) string {
	typ := strings.TrimSpace(r.Type)
	payload := strings.TrimSpace(r.Payload)
	target := strings.TrimSpace(r.Target)
	if typ == "" || target == "" {
		return ""
	}
	if typ == "MATCH" {
		return fmt.Sprintf("MATCH,%s", target)
	}
	if payload == "" {
		return ""
	}
	return fmt.Sprintf("%s,%s,%s", typ, payload, target)
}
