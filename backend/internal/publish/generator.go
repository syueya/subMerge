package publish

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/submerge/submerge/backend/internal/database"
	"gopkg.in/yaml.v3"
	"sort"
	"strings"
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
	// 按订阅源收集：SOURCE:名称 / SOURCE:id:N 展开用
	bySourceName := map[string][]string{} // key = lower(name)
	bySourceID := map[uint][]string{}
	sourceNameByID := map[uint]string{}
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
		if sid, ok := proxySourceID(p); ok {
			bySourceID[sid] = append(bySourceID[sid], name)
			if sn := proxySourceName(p); sn != "" {
				sourceNameByID[sid] = sn
				key := strings.ToLower(sn)
				bySourceName[key] = append(bySourceName[key], name)
			}
		} else if sn := proxySourceName(p); sn != "" {
			key := strings.ToLower(sn)
			bySourceName[key] = append(bySourceName[key], name)
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
	if len(bySourceName) > 0 || len(bySourceID) > 0 {
		// 展示名优先用原始大小写；按名称排序保证 hash 稳定
		labels := make([]string, 0, len(bySourceName)+len(bySourceID))
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
		warnings = append(warnings, "available sources: "+strings.Join(labels, ", "))
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
		expanded := expandRefs(refs, byRegion, bySourceName, bySourceID, allNames)
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
	proxies, droppedReality := sanitizeProxiesForMeta(in.Proxies)
	if droppedReality > 0 {
		warnings = append(warnings,
			fmt.Sprintf("dropped %d proxy(ies) with invalid REALITY short-id", droppedReality))
	}
	// sanitize 可能丢节点，策略组成员需再滤一次（仅删不存在的节点名；DIRECT/REJECT/组名保留）
	if droppedReality > 0 {
		alive := map[string]struct{}{}
		for _, p := range proxies {
			if n, _ := p["name"].(string); n != "" {
				alive[n] = struct{}{}
			}
		}
		for _, g := range groupNames {
			alive[g] = struct{}{}
		}
		alive["DIRECT"] = struct{}{}
		alive["REJECT"] = struct{}{}
		for _, g := range proxyGroups {
			refs, _ := g["proxies"].([]string)
			if refs == nil {
				// yaml 前可能是 []interface{} 路径未走到；兼容 []string
				if raw, ok := g["proxies"].([]interface{}); ok {
					refs = make([]string, 0, len(raw))
					for _, x := range raw {
						refs = append(refs, fmt.Sprint(x))
					}
				}
			}
			filtered := make([]string, 0, len(refs))
			for _, r := range refs {
				if _, ok := alive[r]; ok {
					filtered = append(filtered, r)
				}
			}
			if len(filtered) == 0 {
				filtered = []string{"DIRECT"}
			}
			g["proxies"] = filtered
		}
	}

	// 有序输出：name 置顶；块状列表（不用 {} 行内），可读性更好，也方便对照 proxy N
	orderedProxies := make([]yamlMap, 0, len(proxies))
	for _, p := range proxies {
		orderedProxies = append(orderedProxies, orderedProxy(p))
	}
	orderedGroups := make([]yamlMap, 0, len(proxyGroups))
	for _, g := range proxyGroups {
		orderedGroups = append(orderedGroups, orderedGroup(g))
	}

	// 订阅向配置：只下发节点 / 策略组 / 规则与少量无冲突的全局项。
	// 刻意不写 mixed-port / external-controller / allow-lan 等客户端本机设置，
	// 避免 Clash Verge 导入订阅时覆盖本机端口导致测速 error/timeout。
	// 顶层也用有序 map，保证 proxies → proxy-groups → rules 顺序稳定。
	doc := yamlMap{
		{Key: "mode", Value: "rule"},
		{Key: "ipv6", Value: true},
		{Key: "unified-delay", Value: true},
		{Key: "tcp-concurrent", Value: true},
		{Key: "proxies", Value: orderedProxies},
		{Key: "proxy-groups", Value: orderedGroups},
		{Key: "rules", Value: ruleLines},
	}

	out, err := yaml.Marshal(doc)
	if err != nil {
		return nil, err
	}
	// 在每个 proxy 块前插入「# 1」「# 2」序号注释（不改 name，避免组引用对不上）
	body := annotateProxyIndexes(string(out))
	// 文件头注释：方便识别与 mihomo 导入
	header := "# SubMerge generated subscription for Clash Meta / mihomo / Clash Verge\n" +
		"# proxies + proxy-groups + rules only; client keeps local port/DNS settings\n" +
		"# proxy field order: name, type, server, port, ...; index comments are 1-based\n"
	yamlText := header + body
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
