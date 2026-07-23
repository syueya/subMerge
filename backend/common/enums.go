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
