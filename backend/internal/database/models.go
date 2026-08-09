package database

import (
	"time"

	"gorm.io/gorm"
)

// Admin 管理员
type Admin struct {
	ID           uint   `gorm:"primaryKey"`
	Username     string `gorm:"size:64;uniqueIndex;not null"`
	PasswordHash string `gorm:"size:255;not null"`
	DisplayName  string `gorm:"size:64"`   // 昵称，展示用
	Avatar       string `gorm:"type:text"` // 头像 data URL 或空
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// Session 登录会话（无软删：失效即硬删，token_hash 需可复用）
type Session struct {
	ID        uint      `gorm:"primaryKey"`
	AdminID   uint      `gorm:"index;not null"`
	TokenHash string    `gorm:"size:64;uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"index;not null"`
	CreatedAt time.Time
}

// Source 订阅源
type Source struct {
	ID               uint   `gorm:"primaryKey"`
	Name             string `gorm:"size:128;not null"`
	Region           string `gorm:"size:16;not null;index"` // 默认/回退地区；fixed 模式强制使用
	URLEncrypted     string `gorm:"type:text;not null"`
	Enabled          bool   `gorm:"not null;default:true"`
	RegionMode       string `gorm:"size:16;not null;default:auto"` // auto | fixed
	ExcludeNameRegex string `gorm:"type:text"`
	ExcludeServers   string `gorm:"type:text"`
	IncludeNameRegex string `gorm:"type:text"`
	RefreshStatus    string `gorm:"size:16;not null;default:idle"`
	LastRefreshAt    *time.Time
	LastError        string `gorm:"type:text"`
	SnapshotYAML     string `gorm:"type:text"` // 上次成功快照
	// 上游 Subscription-Userinfo（字节；0 表示未知/未提供）
	TrafficUpload   int64 `gorm:"not null;default:0"`
	TrafficDownload int64 `gorm:"not null;default:0"`
	TrafficTotal    int64 `gorm:"not null;default:0"`
	// TrafficExpire Unix 秒；0 表示未知/不限
	TrafficExpire int64 `gorm:"not null;default:0"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

// Proxy 解析后的节点（刷新时整批硬删重建，无需软删）
type Proxy struct {
	ID          uint   `gorm:"primaryKey"`
	SourceID    uint   `gorm:"index;not null"`
	Name        string `gorm:"size:255;not null;index"`
	Region      string `gorm:"size:16;not null;index"`
	Type        string `gorm:"size:32;not null"`
	Server      string `gorm:"size:255;not null"`
	Port        int    `gorm:"not null"`
	Enabled     bool   `gorm:"not null;default:true"`
	Fingerprint string `gorm:"size:64;index"` // 刷新时用于保留 enabled
	RawJSON     string `gorm:"type:text;not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Rule 规则草稿
type Rule struct {
	ID        uint   `gorm:"primaryKey"`
	Type      string `gorm:"size:32;not null"`
	Payload   string `gorm:"size:512"`
	Target    string `gorm:"size:128;not null"`
	Enabled   bool   `gorm:"not null;default:true"`
	SortOrder int    `gorm:"not null;default:0;index"`
	Note      string `gorm:"size:255"`
	// Category 业务分类（仅面板分组，不写入 Clash）
	Category  string `gorm:"size:64;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// ProxyGroup 策略组草稿（Name 唯一；删除用硬删以便同名重建）
type ProxyGroup struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"size:128;uniqueIndex;not null"`
	Type      string `gorm:"size:32;not null"`
	Proxies   string `gorm:"type:text;not null"` // JSON string array
	URL       string `gorm:"size:255"`
	Interval  *int
	Enabled   bool `gorm:"not null;default:true"`
	SortOrder int  `gorm:"not null;default:0;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ShareToken 订阅链接（订阅链接密钥）。
//
// 生命周期语义：
//   - Revoke：Status=revoked，行保留，旧链接立即 403；可再 Regenerate 恢复。
//   - Delete：硬删整行，TokenHash 唯一索引释放。
//
// TokenHash 用于校验；TokenEncrypted 存 AES 密文以便管理端展示完整订阅链接。
// SourceIDsJSON 允许的订阅源 ID 列表（JSON 数组）；空 / [] 表示全部源。
// GroupMode: auto | all | custom；GroupNamesJSON 仅 custom 时为策略组名白名单。
type ShareToken struct {
	ID             uint   `gorm:"primaryKey"`
	Name           string `gorm:"size:128;not null"`
	TokenHash      string `gorm:"size:64;uniqueIndex;not null"`
	TokenPrefix    string `gorm:"size:8;not null"` // 旧数据/兜底展示
	TokenEncrypted string `gorm:"type:text"`       // AES-GCM 密文；空表示历史令牌需重新生成
	// SourceIDsJSON 例如 [1,3]；空字符串或 [] 表示不限制（全部启用源）
	SourceIDsJSON string `gorm:"type:text"`
	// GroupMode auto=按节点剪空组；all=保留模板；custom=GroupNamesJSON 白名单
	GroupMode string `gorm:"size:16;not null;default:auto"`
	// GroupNamesJSON 例如 ["美国US","菲律宾PH"]；仅 custom 有意义
	GroupNamesJSON string `gorm:"type:text"`
	Status         string `gorm:"size:16;not null;default:active;index"`
	AccessCount    int64  `gorm:"not null;default:0"`
	LastAccessAt   *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// APIKey 自动化 / agent 调用管理 API 的密钥。
//
// 与 ShareToken 分离：APIKey 只用于 /api/*，不能拉 /subscribe。
// KeyHash 鉴权；KeyEncrypted 可再解密以便管理员忘记后查看；
// 管理 CRUD 仅 Session，API Key 自身不能管理其它 API Key。
type APIKey struct {
	ID           uint   `gorm:"primaryKey"`
	Name         string `gorm:"size:128;not null"`
	KeyHash      string `gorm:"size:64;uniqueIndex;not null"`
	KeyPrefix    string `gorm:"size:16;not null"`
	KeyEncrypted string `gorm:"type:text;not null"`
	// ScopesJSON 例如 ["read","publish"] 或 ["*"]
	ScopesJSON string `gorm:"type:text;not null"`
	Status     string `gorm:"size:16;not null;default:active;index"`
	Note       string `gorm:"size:512"`
	ExpiresAt  *time.Time
	LastUsedAt *time.Time
	CreatedBy  string `gorm:"size:64;not null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Release 发布版本
type Release struct {
	ID          uint   `gorm:"primaryKey"`
	Version     int    `gorm:"uniqueIndex;not null"`
	Status      string `gorm:"size:16;not null;index"`
	Note        string `gorm:"size:255"`
	ProxyCount  int    `gorm:"not null;default:0"`
	RuleCount   int    `gorm:"not null;default:0"`
	ConfigHash  string `gorm:"size:64;not null"`
	ConfigYAML  string `gorm:"type:text;not null"`
	PublishedAt *time.Time
	CreatedBy   string `gorm:"size:64;not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type SystemSetting struct {
	ID        uint   `gorm:"primaryKey"`
	Key       string `gorm:"size:64;uniqueIndex;not null"`
	Value     string `gorm:"type:text;not null"`
	Encrypted bool   `gorm:"not null;default:false"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NetCheckSetting 网络检测持久配置（单行 ID=1）。
// 代理地址不落库，仅检测请求临时传入。
type NetCheckSetting struct {
	ID             uint   `gorm:"primaryKey"`
	TimeoutSec     int    `gorm:"not null;default:10"`
	AutoRefreshSec int    `gorm:"not null;default:0"`
	TargetsJSON    string `gorm:"type:text;not null"` // [{name,url,enabled}]
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
