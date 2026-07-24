package publish

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/submerge/submerge/backend/internal/database"
	"github.com/submerge/submerge/backend/internal/regioncatalog"
	"gopkg.in/yaml.v3"
)

// yamlQuotedString 强制双引号输出，避免 short-id: 6314e825 被 YAML 1.1 解析成 .inf
type yamlQuotedString string

func (s yamlQuotedString) MarshalYAML() (interface{}, error) {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Style: yaml.DoubleQuotedStyle,
		Tag:   "!!str",
		Value: string(s),
	}, nil
}

// yamlMap 有序 map，保证 name/type 等关键字段排在前面（map 默认乱序不好读）
type yamlMap []yamlKV

type yamlKV struct {
	Key   string
	Value interface{}
}

func (m yamlMap) MarshalYAML() (interface{}, error) {
	node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, kv := range m {
		var valNode yaml.Node
		if err := valNode.Encode(kv.Value); err != nil {
			return nil, err
		}
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: kv.Key},
			&valNode,
		)
	}
	return node, nil
}

// proxyFieldOrder 节点字段展示顺序：name 最前，便于对照 proxy N 与策略组成员
var proxyFieldOrder = []string{
	"name", "type", "server", "port", "uuid", "password", "cipher", "alterId",
	"network", "tls", "udp", "flow", "servername", "client-fingerprint",
	"skip-cert-verify", "sni", "fingerprint", "ports",
	"reality-opts", "ws-opts", "grpc-opts", "h2-opts", "http-opts",
	"smux", "tfo", "mptcp",
}

// groupFieldOrder 策略组字段顺序
var groupFieldOrder = []string{"name", "type", "proxies", "url", "interval", "lazy", "tolerance"}

// realityFieldOrder REALITY 子字段顺序
var realityFieldOrder = []string{"public-key", "short-id", "support-x25519mlkem768"}

func orderedMap(m map[string]interface{}, prefer []string) yamlMap {
	if m == nil {
		return nil
	}
	out := make(yamlMap, 0, len(m))
	seen := map[string]struct{}{}
	for _, k := range prefer {
		if v, ok := m[k]; ok {
			out = append(out, yamlKV{Key: k, Value: v})
			seen[k] = struct{}{}
		}
	}
	rest := make([]string, 0, len(m))
	for k := range m {
		if _, ok := seen[k]; !ok {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	for _, k := range rest {
		out = append(out, yamlKV{Key: k, Value: m[k]})
	}
	return out
}

func orderedProxy(m map[string]interface{}) yamlMap {
	// 嵌套 opts 也固定顺序
	if ro, ok := m["reality-opts"].(map[string]interface{}); ok {
		m["reality-opts"] = orderedMap(ro, realityFieldOrder)
	}
	return orderedMap(m, proxyFieldOrder)
}

func orderedGroup(m map[string]interface{}) yamlMap {
	return orderedMap(m, groupFieldOrder)
}

// realityShortIDRe mihomo：hex，解码后 ≤8 字节（0~16 个 hex 字符，可为空）
var realityShortIDRe = regexp.MustCompile(`(?i)^[0-9a-f]{0,16}$`)

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

// sanitizeProxiesForMeta 清洗节点 map，保证 mihomo 可解析。
// 第二返回值：因非法 REALITY short-id 丢弃的节点数。
func sanitizeProxiesForMeta(in []map[string]interface{}) ([]map[string]interface{}, int) {
	out := make([]map[string]interface{}, 0, len(in))
	droppedReality := 0
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
		// REALITY：short-id 必须当字符串输出（防 6314e825 → .inf），并校验 hex
		if !normalizeRealityOpts(m) {
			droppedReality++
			continue
		}
		out = append(out, m)
	}
	return out, droppedReality
}

// normalizeRealityOpts 规范化 reality-opts；非法 short-id 返回 false（调用方应丢弃节点）
func normalizeRealityOpts(m map[string]interface{}) bool {
	raw, ok := m["reality-opts"]
	if !ok || raw == nil {
		return true
	}
	opts, ok := raw.(map[string]interface{})
	if !ok {
		// 兼容 json 反序列化后偶发的 map[string]string
		if ms, ok := raw.(map[string]string); ok {
			opts = make(map[string]interface{}, len(ms))
			for k, v := range ms {
				opts[k] = v
			}
		} else {
			delete(m, "reality-opts")
			return true
		}
	}
	out := make(map[string]interface{}, len(opts)+2)
	for k, v := range opts {
		out[k] = v
	}

	if v, exists := out["public-key"]; exists && v != nil {
		pk := strings.TrimSpace(fmt.Sprint(v))
		if pk == "" || pk == "<nil>" {
			delete(out, "public-key")
		} else {
			// 强制引号，避免 base64 中的特殊字符/歧义
			out["public-key"] = yamlQuotedString(pk)
		}
	}

	if v, exists := out["short-id"]; exists && v != nil {
		sid := normalizeRealityShortID(v)
		if sid == "" && fmt.Sprint(v) != "" && fmt.Sprint(v) != "<nil>" {
			// 有值但非法（含 .inf / 非 hex / 过长）
			return false
		}
		// 空 short-id 合法（部分节点允许）；仍强制字符串形态
		out["short-id"] = yamlQuotedString(sid)
	}

	if len(out) == 0 {
		delete(m, "reality-opts")
		return true
	}
	m["reality-opts"] = out
	return true
}

// annotateProxyIndexes 在 proxies 列表每项前加「# N」注释（1-based，对齐 mihomo proxy N 日志）
func annotateProxyIndexes(yamlText string) string {
	lines := strings.Split(yamlText, "\n")
	out := make([]string, 0, len(lines)+32)
	inProxies := false
	idx := 0
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		// 进入 proxies:
		if !inProxies {
			if trimmed == "proxies:" {
				inProxies = true
				out = append(out, line)
				continue
			}
			out = append(out, line)
			continue
		}
		// 离开 proxies：下一顶层 key（无缩进且非空、非列表项）
		if trimmed != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && !strings.HasPrefix(trimmed, "#") {
			inProxies = false
			out = append(out, line)
			continue
		}
		// 列表项起始：两个空格 + "- "（yaml.v3 默认缩进）
		if strings.HasPrefix(line, "  - ") || strings.HasPrefix(line, "    - ") {
			idx++
			// 注释缩进与列表项对齐
			indent := "  "
			if strings.HasPrefix(line, "    - ") {
				indent = "    "
			}
			out = append(out, fmt.Sprintf("%s# %d", indent, idx))
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// normalizeRealityShortID 将 short-id 规范为 hex 字符串；非法返回 "" 且调用方应区分「空」与「非法」
func normalizeRealityShortID(v interface{}) string {
	// 若已被 YAML 误解析成 float/inf，无法还原原始 hex，只能判非法
	switch t := v.(type) {
	case float64, float32:
		return ""
	case yamlQuotedString:
		return normalizeRealityShortID(string(t))
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return ""
		}
		// 客户端已炸成 .inf / +Inf 等
		low := strings.ToLower(s)
		if low == ".inf" || low == "-.inf" || low == "+.inf" || low == "inf" || low == "+inf" || low == "-inf" || low == ".nan" || low == "nan" {
			return ""
		}
		if !realityShortIDRe.MatchString(s) {
			return ""
		}
		// mihomo hex.Decode：奇数长度会失败
		if len(s)%2 != 0 {
			return ""
		}
		return strings.ToLower(s)
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" || s == "<nil>" {
			return ""
		}
		return normalizeRealityShortID(s)
	}
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
