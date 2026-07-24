package common

// ApiError 统一 API 错误
type ApiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ApiResponse 统一包装响应
type ApiResponse[T any] struct {
	OK    bool      `json:"ok"`
	Data  *T        `json:"data,omitempty"`
	Error *ApiError `json:"error,omitempty"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token string    `json:"token"`
	User  AdminUser `json:"user"`
}

// SetupStatusResponse 是否需要首次创建管理员
type SetupStatusResponse struct {
	NeedsSetup bool `json:"needsSetup"`
}

// BootstrapRequest 首次创建管理员（仅空库）
type BootstrapRequest struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required,min=8"`
	DisplayName string `json:"displayName"`
}

// ChangePasswordRequest 修改密码
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8"`
}

// UpdateProfileRequest 更新个人资料（登录名 / 昵称 / 头像）
type UpdateProfileRequest struct {
	// Username 登录名，可选修改
	Username    *string `json:"username"`
	DisplayName *string `json:"displayName"`
	// Avatar 传 data:image/...;base64,... 或空字符串清空
	Avatar *string `json:"avatar"`
}

// CreateSourceRequest 创建订阅源
// Region 可省略：自动模式默认 UNKNOWN；固定模式建议传具体地区码
type CreateSourceRequest struct {
	Name             string      `json:"name" binding:"required"`
	Region           Region      `json:"region"`
	URL              string      `json:"url" binding:"required,url"`
	Enabled          *bool       `json:"enabled"`
	RegionMode       *RegionMode `json:"regionMode"`
	ExcludeNameRegex *string     `json:"excludeNameRegex"`
	ExcludeServers   *string     `json:"excludeServers"`
	IncludeNameRegex *string     `json:"includeNameRegex"`
}

// UpdateSourceRequest 更新订阅源
type UpdateSourceRequest struct {
	Name             *string     `json:"name"`
	Region           *Region     `json:"region"`
	URL              *string     `json:"url"`
	Enabled          *bool       `json:"enabled"`
	RegionMode       *RegionMode `json:"regionMode"`
	ExcludeNameRegex *string     `json:"excludeNameRegex"`
	ExcludeServers   *string     `json:"excludeServers"`
	IncludeNameRegex *string     `json:"includeNameRegex"`
}

// SourceListResponse 订阅源列表
type SourceListResponse struct {
	Items []SubscriptionSource `json:"items"`
}

// BatchDeleteSourcesRequest 批量删除订阅源
type BatchDeleteSourcesRequest struct {
	IDs []uint `json:"ids" binding:"required"`
}

// ProxyListResponse 节点列表
type ProxyListResponse struct {
	Items []ProxyNode `json:"items"`
}

// UpdateProxyRequest 更新节点（目前仅 enabled）
type UpdateProxyRequest struct {
	Enabled *bool `json:"enabled" binding:"required"`
}

// BatchUpdateProxiesRequest 批量更新节点启用状态
type BatchUpdateProxiesRequest struct {
	IDs     []uint `json:"ids" binding:"required"`
	Enabled bool   `json:"enabled"`
}

// RefreshSourceResponse 刷新结果
type RefreshSourceResponse struct {
	Source SubscriptionSource `json:"source"`
	// UpstreamTotal 上游 proxies 列表原始条数
	UpstreamTotal int `json:"upstreamTotal"`
	// Parsed 解析成功（具备 name/type/server/port）条数
	Parsed  int `json:"parsed"`
	Added   int `json:"added"`
	Removed int `json:"removed"`
	// Skipped 过滤规则丢弃条数
	Skipped int `json:"skipped"`
	// ParseDropped 解析阶段丢弃原因 → 数量（缺字段/端口非法/重名等）
	ParseDropped map[string]int `json:"parseDropped,omitempty"`
	// FilterDropped 过滤阶段丢弃原因 → 数量
	FilterDropped map[string]int `json:"filterDropped,omitempty"`
	// FilteredNames 被过滤掉的上游节点名（完整列表；含信息节点与规则排除）
	FilteredNames []string `json:"filteredNames,omitempty"`
	// ParseDroppedNames 解析失败的条目摘要（最多 20 条）
	ParseDroppedNames []string       `json:"parseDroppedNames,omitempty"`
	RegionCounts      map[string]int `json:"regionCounts,omitempty"`
}

// RefreshAllItem 批量刷新单项结果
type RefreshAllItem struct {
	SourceID uint   `json:"sourceId"`
	Name     string `json:"name"`
	OK       bool   `json:"ok"`
	Added    int    `json:"added,omitempty"`
	Skipped  int    `json:"skipped,omitempty"`
	Error    string `json:"error,omitempty"`
}

// RefreshAllResponse 批量刷新结果
type RefreshAllResponse struct {
	Total   int              `json:"total"`
	OK      int              `json:"ok"`
	Failed  int              `json:"failed"`
	Results []RefreshAllItem `json:"results"`
}

// DraftStatusResponse 草稿相对已发布是否有变更
type DraftStatusResponse struct {
	HasPublished     bool   `json:"hasPublished"`
	Dirty            bool   `json:"dirty"`
	PublishedHash    string `json:"publishedHash,omitempty"`
	DraftHash        string `json:"draftHash,omitempty"`
	PublishedVersion int    `json:"publishedVersion,omitempty"`
	// BuildError 草稿无法生成时的错误（仍可视为 dirty）
	BuildError string `json:"buildError,omitempty"`
}

// RuleListResponse 规则列表
type RuleListResponse struct {
	Items []Rule `json:"items"`
}

// UpsertRuleRequest 创建/更新规则
	type UpsertRuleRequest struct {
		Type      string `json:"type" binding:"required"`
		Payload   string `json:"payload"`
		Target    string `json:"target" binding:"required"`
		Enabled   *bool  `json:"enabled"`
		SortOrder *int   `json:"sortOrder"`
		Note      string `json:"note"`
		// Category 业务分类（面板分组）
		Category  string `json:"category"`
	}

	// BatchImportRulesRequest 批量导入规则（文本一行一条）
	// 每行格式：
	//   TYPE,payload,target[,note[,category]]
	//   或仅 payload（用 defaultType / defaultTarget / defaultNote / defaultCategory）
	// 空行与 # 注释忽略。新规则插在 GEOIP CN / MATCH 之前。
	type BatchImportRulesRequest struct {
		Text            string `json:"text" binding:"required"`
		DefaultType     string `json:"defaultType"`
		DefaultTarget   string `json:"defaultTarget"`
		DefaultNote     string `json:"defaultNote"`
		DefaultCategory string `json:"defaultCategory"`
		Enabled         *bool  `json:"enabled"`
	}

// BatchImportRulesResponse 批量导入结果
type BatchImportRulesResponse struct {
	Created int      `json:"created"`
	Skipped int      `json:"skipped"`
	Items   []Rule   `json:"items"`
	Errors  []string `json:"errors,omitempty"`
}

// ReorderRulesRequest 规则排序
type ReorderRulesRequest struct {
	OrderedIDs []uint `json:"orderedIds" binding:"required"`
}

// BatchUpdateRulesTargetRequest 批量修改规则目标出口
	type BatchUpdateRulesTargetRequest struct {
		IDs    []uint `json:"ids" binding:"required"`
		Target string `json:"target" binding:"required"`
	}

	// BatchUpdateRulesTargetResponse 批量改出口结果
	type BatchUpdateRulesTargetResponse struct {
		Updated int `json:"updated"`
	}

	// BatchUpdateRulesEnabledRequest 批量启用/禁用规则
	type BatchUpdateRulesEnabledRequest struct {
		IDs     []uint `json:"ids" binding:"required"`
		Enabled bool   `json:"enabled"`
	}

	// BatchUpdateRulesEnabledResponse 批量启用/禁用结果
	type BatchUpdateRulesEnabledResponse struct {
		Updated int `json:"updated"`
	}

	// BatchUpdateRulesCategoryRequest 批量修改规则业务分类
	type BatchUpdateRulesCategoryRequest struct {
		IDs      []uint `json:"ids" binding:"required"`
		Category string `json:"category"`
	}

	// BatchUpdateRulesCategoryResponse 批量改分类结果
	type BatchUpdateRulesCategoryResponse struct {
		Updated int `json:"updated"`
	}

	// BatchDeleteRulesRequest 批量删除规则
	type BatchDeleteRulesRequest struct {
		IDs []uint `json:"ids" binding:"required"`
	}

	// BatchDeleteRulesResponse 批量删除结果
	type BatchDeleteRulesResponse struct {
		Deleted int `json:"deleted"`
	}

// ProxyGroupListResponse 策略组列表
type ProxyGroupListResponse struct {
	Items []ProxyGroup `json:"items"`
}

// UpsertProxyGroupRequest 创建/更新策略组
type UpsertProxyGroupRequest struct {
	Name      string   `json:"name" binding:"required"`
	Type      string   `json:"type" binding:"required"`
	Proxies   []string `json:"proxies"`
	URL       string   `json:"url"`
	Interval  *int     `json:"interval"`
	Enabled   *bool    `json:"enabled"`
	SortOrder *int     `json:"sortOrder"`
}

// TokenListResponse 令牌列表
type TokenListResponse struct {
	Items []ShareToken `json:"items"`
}

// CreateTokenRequest 创建令牌
	// SourceIDs 可选；省略或空数组 = 全部订阅源；非空则仅分享所列源
	// GroupMode 默认 auto；custom 时用 GroupNames 白名单
	type CreateTokenRequest struct {
		Name       string         `json:"name" binding:"required"`
		SourceIDs  []uint         `json:"sourceIds"`
		GroupMode  TokenGroupMode `json:"groupMode"`
		GroupNames []string       `json:"groupNames"`
	}

	// UpdateTokenRequest 更新令牌
	// SourceIDs 传非 nil 指针时更新允许源列表（可为空数组表示改回全部）
	type UpdateTokenRequest struct {
		Name       *string         `json:"name"`
		Status     *TokenStatus    `json:"status"`
		SourceIDs  *[]uint         `json:"sourceIds"`
		GroupMode  *TokenGroupMode `json:"groupMode"`
		GroupNames *[]string       `json:"groupNames"`
	}

// ReleaseListResponse 发布列表
type ReleaseListResponse struct {
	Items []Release `json:"items"`
}

// PublishRequest 发布请求
type PublishRequest struct {
	Note string `json:"note"`
}

// PublishResponse 发布结果
type PublishResponse struct {
	Release Release        `json:"release"`
	Preview ReleasePreview `json:"preview"`
}

// HealthResponse 健康检查
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Time    string `json:"time"`
}

// RegionCatalogEntry 地区目录项（UI 下拉）
type RegionCatalogEntry struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// RegionCatalogResponse 地区目录
type RegionCatalogResponse struct {
	Items          []RegionCatalogEntry `json:"items"`
	FallbackRegion string               `json:"fallbackRegion"` // 自动模式默认回退，通常 UNKNOWN
}

// MeResponse 当前用户
type MeResponse struct {
	User AdminUser `json:"user"`
}

// AuditListResponse 审计列表
type AuditListResponse struct {
	Items []AuditLog `json:"items"`
	Total int64      `json:"total"`
}
