package netcheck

// Target 检测目标
type Target struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

// ProxyConfig 代理设置（仅检测时临时使用，不落库）
type ProxyConfig struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
}

// Config 可持久化配置（不含代理）
type Config struct {
	Timeout     int      `json:"timeout"`
	AutoRefresh int      `json:"autoRefresh"`
	Targets     []Target `json:"targets"`
}

// CheckRequest 检测请求：可选覆盖目标/超时，并带临时代理
type CheckRequest struct {
	Proxy       *ProxyConfig `json:"proxy,omitempty"`
	Timeout     *int         `json:"timeout,omitempty"`
	AutoRefresh *int         `json:"autoRefresh,omitempty"`
	Targets     []Target     `json:"targets,omitempty"`
}

// Timing 分阶段耗时（毫秒）
type Timing struct {
	ConnectMs   int `json:"connectMs"`
	TLSMs       int `json:"tlsMs"`
	FirstByteMs int `json:"firstByteMs"`
	TotalMs     int `json:"totalMs"`
}

// HTTPResult 单次 HTTP 探测结果
type HTTPResult struct {
	OK           bool   `json:"ok"`
	Status       string `json:"status"` // OK | FAIL
	Code         int    `json:"code"`
	TimeMs       int    `json:"timeMs"`
	Timing       Timing `json:"timing"`
	RemoteIP     string `json:"remoteIp,omitempty"`
	EffectiveURL string `json:"effectiveUrl,omitempty"`
	Error        string `json:"error,omitempty"`
}

// TargetResult 单个目标的检测结果
type TargetResult struct {
	Name      string     `json:"name"`
	URL       string     `json:"url"`
	Status    string     `json:"status"` // OK | FAIL
	CheckedAt string     `json:"checkedAt"`
	HTTP      HTTPResult `json:"http"`
}

// Summary 汇总
type Summary struct {
	Total      int    `json:"total"`
	OK         int    `json:"ok"`
	Fail       int    `json:"fail"`
	DurationMs int    `json:"durationMs"`
	CheckedAt  string `json:"checkedAt"`
}

// CheckResponse 检测 API 响应
type CheckResponse struct {
	Summary Summary        `json:"summary"`
	Results []TargetResult `json:"results"`
}

// runConfig 内部完整运行配置（持久项 + 临时代理）
type runConfig struct {
	Proxy       ProxyConfig
	Timeout     int
	AutoRefresh int
	Targets     []Target
}
