package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/submerge/submerge/backend/internal/applog"
	"github.com/submerge/submerge/backend/internal/crypto"
	"github.com/submerge/submerge/backend/version"
)

// DefaultSourceFetchUA 是订阅源请求的默认 User-Agent。
const DefaultSourceFetchUA = "clash-verge/v2.5.3"

const DefaultPublicBaseURL = "http://localhost:8080"

const (
	DefaultSourceFetchTimeout = 30 * time.Second
	DefaultSourceMaxBytes     = 8 << 20
	DefaultRefreshInterval    = 24 * time.Hour
	DefaultIPGeoTimeout       = 5 * time.Second
	DefaultGeoIPURL           = "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geoip.dat"
	DefaultGeoSiteURL         = "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geosite.dat"
	DefaultMetaDBURL          = "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geoip.metadb"
	DefaultASNURL             = "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/GeoLite2-ASN.mmdb"
	DefaultIPGeoURL           = "https://ipwho.is/{ip}"
)

// Config 应用配置
type Config struct {
	Env                string
	HTTPAddr           string
	PublicBaseURL      string
	DataDir            string
	DBPath             string
	StaticDir          string
	EncryptionKey      string
	SessionTTL         time.Duration
	SourceFetchTimeout time.Duration
	SourceMaxBytes     int64
	SourceFetchUA      string
	RefreshInterval    time.Duration
	RateLimitLogin     int
	RateLimitSub       int
	CORSOrigins        []string
	// TrustedProxies 可信反向代理的 IP/CIDR 列表，由系统设置加载。
	// 仅当请求来自这些代理时才采信 X-Forwarded-For/X-Real-IP 得到真实客户端 IP，
	// 否则用连接对端地址。留空=不信任任何代理（ClientIP 恒为对端地址）。
	// 与 CookieSecure 独立：前者管「真实 IP」，后者管「会话 Cookie 是否仅 HTTPS」。
	TrustedProxies []string
	// CookieSecure 会话 Cookie 是否带 Secure，由系统设置加载。默认 false，便于 http://IP:端口 登录；
	// HTTPS 反代/公网请在系统设置中开启。不跟 APP_ENV、PUBLIC_BASE_URL 联动。
	CookieSecure bool
	Version      string
	// LogOutput: console | file | both | none；未设置时默认 both
	LogOutput string
	// LogDir 固定目录（不走环境变量）：backend/log 或 ./log，按日 submerge-YYYY-MM-DD.log
	LogDir string
	// LogRetentionDays 日志保留天数；默认 7，0 表示不自动清理
	LogRetentionDays int
	// DebugLogging 是否输出地区/过滤等详细 DEBUG 日志。
	DebugLogging     bool
	GeoDir           string
	GeoIPURL         string
	GeoSiteURL       string
	MetaDBURL        string
	ASNURL           string
	IPGeoURL         string
	IPGeoTimeout     time.Duration
	OutboundProxyURL string
}

// Load 从环境变量加载配置
func Load() (*Config, error) {
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")

	sessionTTL, err := getDuration("SESSION_TTL", 24*time.Hour)
	if err != nil {
		return nil, err
	}
	appEnv := getEnv("APP_ENV", "development")
	rateLimitLogin, err := getInt("RATE_LIMIT_LOGIN", 10)
	if err != nil {
		return nil, err
	}
	rateLimitSub, err := getInt("RATE_LIMIT_SUBSCRIBE", 60)
	if err != nil {
		return nil, err
	}
	// 数据 / 库 / 静态 / 日志目录一律按工作目录推导，不提供环境变量覆盖
	// （避免 Docker/.env 再抄一遍路径；部署只需选对工作目录）
	dataDir := defaultDataDir()
	encryptionKey, err := loadEncryptionKey(dataDir)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Env:                appEnv,
		HTTPAddr:           getEnv("HTTP_ADDR", ":8080"),
		PublicBaseURL:      DefaultPublicBaseURL,
		DataDir:            dataDir,
		DBPath:             filepath.Join(dataDir, "submerge.db"),
		StaticDir:          defaultStaticDir(),
		EncryptionKey:      encryptionKey,
		SessionTTL:         sessionTTL,
		SourceFetchTimeout: DefaultSourceFetchTimeout,
		SourceMaxBytes:     DefaultSourceMaxBytes,
		SourceFetchUA:      DefaultSourceFetchUA,
		RefreshInterval:    DefaultRefreshInterval,
		RateLimitLogin:     rateLimitLogin,
		RateLimitSub:       rateLimitSub,
		CORSOrigins:        splitCSV(getEnv("CORS_ORIGINS", "http://localhost:4200")),
		TrustedProxies:     nil,
		CookieSecure:       false,
		// 版本 / 路径固定，不走环境变量
		Version:          version.String(),
		LogOutput:        applog.NormalizeOutput("both"),
		LogDir:           defaultLogDir(),
		LogRetentionDays: 7,
		DebugLogging:     !strings.EqualFold(strings.TrimSpace(appEnv), "production"),
		GeoDir:           defaultGeoDir(),
		GeoIPURL:         DefaultGeoIPURL,
		GeoSiteURL:       DefaultGeoSiteURL,
		MetaDBURL:        DefaultMetaDBURL,
		ASNURL:           DefaultASNURL,
		IPGeoURL:         DefaultIPGeoURL,
		IPGeoTimeout:     DefaultIPGeoTimeout,
		OutboundProxyURL: "",
	}

	if len(cfg.EncryptionKey) < 32 {
		return nil, fmt.Errorf("ENCRYPTION_KEY must be at least 32 characters")
	}
	return cfg, nil
}

func loadEncryptionKey(dataDir string) (string, error) {
	raw := os.Getenv("ENCRYPTION_KEY")
	if strings.TrimSpace(raw) != "" {
		return raw, nil
	}
	return crypto.LoadOrCreateKey(filepath.Join(dataDir, "crypto.key"))
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// defaultDataDir 按当前工作目录选择（不依赖环境变量）：
//   - 项目根（存在 backend/ 子目录）→ ./backend/data
//   - backend/ 内（go run .）或已编译二进制（Docker WORKDIR=/app）→ ./data
func defaultDataDir() string {
	if isDir("backend") && (fileExists("backend/go.mod") || isDir("backend/internal")) {
		return filepath.Clean("./backend/data")
	}
	// cd backend && go run .  /  Docker 等二进制部署
	return filepath.Clean("./data")
}

// defaultLogDir 与 data 同策略（不提供 LOG_DIR 环境变量）：
//   - 项目根 → ./backend/log
//   - backend/ 内 / 已编译二进制 → ./log
func defaultLogDir() string {
	if isDir("backend") && (fileExists("backend/go.mod") || isDir("backend/internal")) {
		return filepath.Clean("./backend/log")
	}
	return filepath.Clean("./log")
}

// defaultStaticDir 相对工作目录；Docker 将静态资源放在 ./frontend/dist/submerge/browser
func defaultStaticDir() string {
	if isDir("frontend/dist/submerge/browser") {
		return filepath.Clean("./frontend/dist/submerge/browser")
	}
	if isDir("../frontend/dist/submerge/browser") {
		return filepath.Clean("../frontend/dist/submerge/browser")
	}
	return filepath.Clean("./frontend/dist/submerge/browser")
}

func defaultGeoDir() string {
	if isDir("backend/defaults/geo") {
		return filepath.Clean("./backend/defaults/geo")
	}
	if isDir("defaults/geo") {
		return filepath.Clean("./defaults/geo")
	}
	return filepath.Clean("./backend/defaults/geo")
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func getInt(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	if n <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", key)
	}
	return n, nil
}

// getIntAllowZero 允许 0（如 LOG_RETENTION_DAYS=0 关闭清理）
func getIntAllowZero(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	if n < 0 {
		return 0, fmt.Errorf("%s must be >= 0", key)
	}
	return n, nil
}

func getInt64(key string, def int64) (int64, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	if n <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", key)
	}
	return n, nil
}

func getDuration(key string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration: %w", key, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", key)
	}
	return d, nil
}

// getHoursDuration 解析「小时数」环境变量：只写整数，单位固定为小时。
// 例如 SOURCE_REFRESH_INTERVAL=24 → 24h。
func getHoursDuration(key string, def time.Duration) (time.Duration, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer number of hours: %w", key, err)
	}
	if n <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", key)
	}
	return time.Duration(n) * time.Hour, nil
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
