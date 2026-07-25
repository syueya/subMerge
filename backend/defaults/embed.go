// Package defaults 仅内嵌 YAML 配置（go:embed），不含业务逻辑。
//
// 解析/使用请放到 internal 对应包，例如：
//
//   - regioncatalog 解析 regions.yaml（目录 + 识别表 + primary）
//
//   - source 使用 regioncatalog 做节点名识别
//
//   - rule 解析 groups.yaml / rules.yaml
//
//     defaults/groups.yaml          空库默认策略组
//     defaults/rules.yaml           空库默认分流规则
//     defaults/regions.yaml         地区统一配置（目录/识别/常用标记）
//     defaults/source_filter.yaml   新建源默认过滤
package defaults

import _ "embed"

// GroupsYAML 空库默认策略组
//
//go:embed groups.yaml
var GroupsYAML []byte

// RulesYAML 空库默认分流规则
//
//go:embed rules.yaml
var RulesYAML []byte

// RegionsYAML 地区统一配置（code/name/primary/flag/keywords）
//
//go:embed regions.yaml
var RegionsYAML []byte

// SourceFilterYAML 新建源默认过滤
//
//go:embed source_filter.yaml
var SourceFilterYAML []byte
