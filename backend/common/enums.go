package common

import "strings"

// Region 订阅源地区码（自由文本，如 US / PH / JP / HK，不限死）
type Region string

// RuleType Clash 规则类型
type RuleType string

const (
	RuleTypeDomain        RuleType = "DOMAIN"
	RuleTypeDomainSuffix  RuleType = "DOMAIN-SUFFIX"
	RuleTypeDomainKeyword RuleType = "DOMAIN-KEYWORD"
	RuleTypeGeoSite       RuleType = "GEOSITE"
	RuleTypeGeoIP         RuleType = "GEOIP"
	RuleTypeIPCIDR        RuleType = "IP-CIDR"
	RuleTypeIPCIDR6       RuleType = "IP-CIDR6"
	RuleTypeSrcIPCIDR     RuleType = "SRC-IP-CIDR"
	RuleTypeSrcPort       RuleType = "SRC-PORT"
	RuleTypeDstPort       RuleType = "DST-PORT"
	RuleTypeProcessName   RuleType = "PROCESS-NAME"
	RuleTypeProcessPath   RuleType = "PROCESS-PATH"
	RuleTypeRuleSet       RuleType = "RULE-SET"
	RuleTypeMatch         RuleType = "MATCH"
)

// TokenStatus 分享令牌状态
type TokenStatus string

const (
	TokenStatusActive   TokenStatus = "active"
	TokenStatusDisabled TokenStatus = "disabled"
	TokenStatusRevoked  TokenStatus = "revoked"
)

// TokenGroupMode 令牌策略组投影：auto=按节点剪空组；all=保留模板组；custom=白名单
type TokenGroupMode string

const (
	TokenGroupModeAuto   TokenGroupMode = "auto"
	TokenGroupModeAll    TokenGroupMode = "all"
	TokenGroupModeCustom TokenGroupMode = "custom"
)

// NormalizeTokenGroupMode 非法/空值 → auto
func NormalizeTokenGroupMode(m TokenGroupMode) TokenGroupMode {
	switch TokenGroupMode(strings.ToLower(strings.TrimSpace(string(m)))) {
	case TokenGroupModeAll:
		return TokenGroupModeAll
	case TokenGroupModeCustom:
		return TokenGroupModeCustom
	default:
		return TokenGroupModeAuto
	}
}

// ReleaseStatus 发布版本状态
type ReleaseStatus string

const (
	ReleaseStatusDraft      ReleaseStatus = "draft"
	ReleaseStatusPublished  ReleaseStatus = "published"
	ReleaseStatusRolledBack ReleaseStatus = "rolled_back"
)

// RefreshStatus 订阅源刷新状态
type RefreshStatus string

const (
	RefreshStatusIdle    RefreshStatus = "idle"
	RefreshStatusSuccess RefreshStatus = "success"
	RefreshStatusFailed  RefreshStatus = "failed"
	RefreshStatusRunning RefreshStatus = "running"
)

// ProxyGroupType 策略组类型
type ProxyGroupType string

const (
	ProxyGroupTypeSelect      ProxyGroupType = "select"
	ProxyGroupTypeURLTest     ProxyGroupType = "url-test"
	ProxyGroupTypeFallback    ProxyGroupType = "fallback"
	ProxyGroupTypeLoadBalance ProxyGroupType = "load-balance"
)

// Clash 引擎内建出口关键字（非策略组，写入 Clash 配置的保留字）。
const (
	TargetDirect = "DIRECT"
	TargetReject = "REJECT"
)

// 内置策略组名。这些名字在三处强耦合，必须一致，否则发布时会静默降级：
//   - defaults/groups.yaml   种子组定义
//   - internal/rule          seed/migrate 创建组、系统规则 target
//   - internal/publish       generator 校验 target 合法性与回退
//
// 例如 generator 找不到 GroupNameSelectAll 时会静默回退到 TargetDirect，
// 导致所有规则目标落到直连而不报错。改名时请同步以上各处。
const (
	GroupNameDirect    = "直连"     // 仅含 DIRECT 的选择组
	GroupNameReject    = "拒绝"     // 仅含 REJECT 的选择组
	GroupNameSelectAll = "节点选择"   // 总选择组，也是规则 target 缺失时的回退组
	GroupNameOther     = "其他国家"   // 非常用地区节点聚合组
	GroupNameDefaultUS = "美国US"   // MATCH 兜底默认出口
)

// 策略组成员展开 token（写入 ProxyGroup.Proxies，由 generator 展开为实际节点）。
const (
	MemberTokenAll    = "ALL"          // 全部节点
	RegionTokenPrefix = "REGION:"      // 地区前缀，如 REGION:US
	RegionTokenAll    = "REGION:ALL"   // 全部节点（等价 REGION:*）
	RegionTokenOther  = "REGION:OTHER" // 非常用地区节点聚合
)
