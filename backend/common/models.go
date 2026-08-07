package common

// RegionMode 订阅源地区模式
type RegionMode string

const (
	RegionModeAuto  RegionMode = "auto"
	RegionModeFixed RegionMode = "fixed"
)

// SubscriptionSource 订阅源（脱敏视图）
type SubscriptionSource struct {
	ID               uint          `json:"id"`
	Name             string        `json:"name"`
	Region           Region        `json:"region"`
	URLMasked        string        `json:"urlMasked"`
	Enabled          bool          `json:"enabled"`
	RegionMode       RegionMode    `json:"regionMode"`
	ExcludeNameRegex string        `json:"excludeNameRegex"`
	ExcludeServers   string        `json:"excludeServers"`
	IncludeNameRegex string        `json:"includeNameRegex"`
	RefreshStatus    RefreshStatus `json:"refreshStatus"`
	LastRefreshAt    *string       `json:"lastRefreshAt,omitempty"`
	LastError        *string       `json:"lastError,omitempty"`
	ProxyCount       int           `json:"proxyCount"`
	// 上游 Subscription-Userinfo（Clash Verge 同源约定，字节）
	// 无该头时均为 0；前端按 0 视为「未知」
	TrafficUpload   int64 `json:"trafficUpload"`
	TrafficDownload int64 `json:"trafficDownload"`
	TrafficTotal    int64 `json:"trafficTotal"`
	// TrafficExpire Unix 秒；0 = 未知/不限
	TrafficExpire int64  `json:"trafficExpire"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

// ProxyNode 节点摘要
type ProxyNode struct {
	ID       uint   `json:"id"`
	SourceID uint   `json:"sourceId"`
	Name     string `json:"name"`
	Region   Region `json:"region"`
	Type     string `json:"type"`
	Server   string `json:"server"`
	Port     int    `json:"port"`
	Enabled  bool   `json:"enabled"`
	// UDP 订阅配置；nil 表示上游未提供该字段。
	UDP *bool `json:"udp,omitempty"`
	// OK 是否像正常代理节点（非信息节点、字段完整、地区可识别等）
	OK bool `json:"ok"`
	// Issue 非正常时的原因（给前端「识别」列）
	Issue string                 `json:"issue,omitempty"`
	Raw   map[string]interface{} `json:"raw,omitempty"`
}

// Rule 分流规则
type Rule struct {
	ID        uint     `json:"id"`
	Type      RuleType `json:"type"`
	Payload   string   `json:"payload"`
	Target    string   `json:"target"`
	Enabled   bool     `json:"enabled"`
	SortOrder int      `json:"sortOrder"`
	Note      string   `json:"note,omitempty"`
	// Category 业务分类（仅面板；不进 Clash）
	Category string `json:"category,omitempty"`
}

// ProxyGroup 策略组
type ProxyGroup struct {
	ID        uint           `json:"id"`
	Name      string         `json:"name"`
	Type      ProxyGroupType `json:"type"`
	Proxies   []string       `json:"proxies"`
	URL       string         `json:"url,omitempty"`
	Interval  *int           `json:"interval,omitempty"`
	Enabled   bool           `json:"enabled"`
	SortOrder int            `json:"sortOrder"`
}

// ShareToken 订阅链接
// SourceIDs 为空表示该链接包含全部启用订阅源；非空则仅包含所列源的节点
// GroupMode 控制策略组投影：auto / all / custom
type ShareToken struct {
	ID          uint        `json:"id"`
	Name        string      `json:"name"`
	Token       string      `json:"token,omitempty"`
	TokenMasked string      `json:"tokenMasked"`
	Status      TokenStatus `json:"status"`
	// SourceIDs 允许的订阅源；空数组 = 全部源
	SourceIDs []uint `json:"sourceIds"`
	// SourceNames 与 SourceIDs 对应的源名称（列表展示用；已删除的源会标为已失效）
	SourceNames []string `json:"sourceNames,omitempty"`
	// GroupMode auto=按节点剪空组；all=尽量保留模板组；custom=仅 groupNames
	GroupMode TokenGroupMode `json:"groupMode"`
	// GroupNames custom 模式下的策略组白名单；其它模式可为空
	GroupNames   []string `json:"groupNames,omitempty"`
	AccessCount  int64    `json:"accessCount"`
	LastAccessAt *string  `json:"lastAccessAt,omitempty"`
	CreatedAt    string   `json:"createdAt"`
	UpdatedAt    string   `json:"updatedAt"`
	SubscribeURL string   `json:"subscribeUrl,omitempty"`
}

// APIKey 管理端 API 密钥视图
// Key 仅在 create / regenerate / secret 接口返回；列表只有 KeyMasked
type APIKey struct {
	ID         uint         `json:"id"`
	Name       string       `json:"name"`
	Key        string       `json:"key,omitempty"`
	KeyMasked  string       `json:"keyMasked"`
	Scopes     []string     `json:"scopes"`
	Status     APIKeyStatus `json:"status"`
	Note       string       `json:"note,omitempty"`
	ExpiresAt  *string      `json:"expiresAt,omitempty"`
	LastUsedAt *string      `json:"lastUsedAt,omitempty"`
	CreatedBy  string       `json:"createdBy"`
	CreatedAt  string       `json:"createdAt"`
	UpdatedAt  string       `json:"updatedAt"`
}

// Release 发布版本
type Release struct {
	ID          uint          `json:"id"`
	Version     int           `json:"version"`
	Status      ReleaseStatus `json:"status"`
	Note        string        `json:"note,omitempty"`
	ProxyCount  int           `json:"proxyCount"`
	RuleCount   int           `json:"ruleCount"`
	ConfigHash  string        `json:"configHash"`
	PublishedAt *string       `json:"publishedAt,omitempty"`
	CreatedAt   string        `json:"createdAt"`
	CreatedBy   string        `json:"createdBy"`
}

// ReleaseRuleLine 发布配置中的单条规则（从 YAML 解析，便于历史查看/匹配测试）
type ReleaseRuleLine struct {
	Type    string `json:"type"`
	Payload string `json:"payload,omitempty"`
	Target  string `json:"target"`
	Raw     string `json:"raw"`
}

// ReleaseDetail 发布版本详情（含完整 YAML 与规则列表）
type ReleaseDetail struct {
	Release
	ConfigYAML string            `json:"configYaml"`
	Rules      []ReleaseRuleLine `json:"rules"`
	Groups     []string          `json:"groups"`
}

// ReleasePreview 发布预览
type ReleasePreview struct {
	ProxyCount  int      `json:"proxyCount"`
	RuleCount   int      `json:"ruleCount"`
	Groups      []string `json:"groups"`
	YAMLPreview string   `json:"yamlPreview"`
	Warnings    []string `json:"warnings"`
}

// AdminUser 管理员
type AdminUser struct {
	ID          uint    `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"displayName"`
	Avatar      string  `json:"avatar,omitempty"` // data URL 或空
	CreatedAt   string  `json:"createdAt"`
	LastLoginAt *string `json:"lastLoginAt,omitempty"`
}

// AuditLog 审计日志
type AuditLog struct {
	ID        uint   `json:"id"`
	Actor     string `json:"actor"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
	Detail    string `json:"detail,omitempty"`
	IP        string `json:"ip,omitempty"`
	CreatedAt string `json:"createdAt"`
}
