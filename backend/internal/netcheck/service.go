package netcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/submerge/submerge/backend/internal/database"
	"golang.org/x/net/proxy"
	"gorm.io/gorm"
)

const (
	settingID     = 1
	maxTargets    = 50
	maxWorkers    = 8
	minTimeoutSec = 1
	maxTimeoutSec = 120
	maxAutoRefSec = 3600
)

// Service 网络检测配置与出站探测。
type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func DefaultConfig() Config {
	return Config{
		Timeout:     10,
		AutoRefresh: 0,
		Targets: []Target{
			{Name: "Google", URL: "https://www.google.com/generate_204", Enabled: true},
			{Name: "GitHub", URL: "https://github.com/", Enabled: true},
			{Name: "TMDB", URL: "https://api.themoviedb.org/3/configuration", Enabled: true},
			{Name: "Docker Hub", URL: "https://registry-1.docker.io/v2/", Enabled: true},
		},
	}
}

func (s *Service) GetConfig() (Config, error) {
	row, err := s.loadOrCreateRow()
	if err != nil {
		return Config{}, err
	}
	return rowToConfig(row)
}

func (s *Service) SaveConfig(cfg Config) (Config, error) {
	normalized, err := NormalizeConfig(cfg)
	if err != nil {
		return Config{}, err
	}
	if err := s.persist(normalized); err != nil {
		return Config{}, err
	}
	return normalized, nil
}

func (s *Service) ResetConfig() (Config, error) {
	cfg := DefaultConfig()
	if err := s.persist(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Check 执行检测。请求中的代理和其他覆盖项只在本次运行生效。
func (s *Service) Check(req CheckRequest) (CheckResponse, error) {
	cfg, err := s.GetConfig()
	if err != nil {
		return CheckResponse{}, err
	}
	if req.Timeout != nil {
		cfg.Timeout = *req.Timeout
	}
	if req.AutoRefresh != nil {
		cfg.AutoRefresh = *req.AutoRefresh
	}
	if req.Targets != nil {
		cfg.Targets = req.Targets
	}
	if req.Proxy != nil {
		cfgRun := runConfig{
			Proxy:       *req.Proxy,
			Timeout:     cfg.Timeout,
			AutoRefresh: cfg.AutoRefresh,
			Targets:     cfg.Targets,
		}
		return runChecks(cfgRun)
	}
	return runChecks(runConfig{Timeout: cfg.Timeout, AutoRefresh: cfg.AutoRefresh, Targets: cfg.Targets})
}

func (s *Service) loadOrCreateRow() (database.NetCheckSetting, error) {
	var row database.NetCheckSetting
	err := s.db.First(&row, settingID).Error
	if err == nil {
		return row, nil
	}
	if err != gorm.ErrRecordNotFound {
		return row, err
	}
	if err := s.persist(DefaultConfig()); err != nil {
		return row, err
	}
	if err := s.db.First(&row, settingID).Error; err != nil {
		return row, err
	}
	return row, nil
}

func (s *Service) persist(cfg Config) error {
	targetsJSON, err := json.Marshal(cfg.Targets)
	if err != nil {
		return err
	}
	row := database.NetCheckSetting{
		ID:             settingID,
		TimeoutSec:     cfg.Timeout,
		AutoRefreshSec: cfg.AutoRefresh,
		TargetsJSON:    string(targetsJSON),
	}
	return s.db.Save(&row).Error
}

func rowToConfig(row database.NetCheckSetting) (Config, error) {
	var targets []Target
	if row.TargetsJSON != "" {
		if err := json.Unmarshal([]byte(row.TargetsJSON), &targets); err != nil {
			return Config{}, fmt.Errorf("decode targets: %w", err)
		}
	}
	return NormalizeConfig(Config{
		Timeout:     row.TimeoutSec,
		AutoRefresh: row.AutoRefreshSec,
		Targets:     targets,
	})
}

func NormalizeConfig(cfg Config) (Config, error) {
	out := Config{}
	out.Timeout = cfg.Timeout
	if out.Timeout < minTimeoutSec {
		out.Timeout = minTimeoutSec
	}
	if out.Timeout > maxTimeoutSec {
		out.Timeout = maxTimeoutSec
	}
	out.AutoRefresh = cfg.AutoRefresh
	if out.AutoRefresh < 0 {
		out.AutoRefresh = 0
	}
	if out.AutoRefresh > maxAutoRefSec {
		out.AutoRefresh = maxAutoRefSec
	}

	if len(cfg.Targets) > maxTargets {
		return Config{}, fmt.Errorf("目标数量不能超过 %d 个", maxTargets)
	}
	targets := make([]Target, 0, len(cfg.Targets))
	for i, target := range cfg.Targets {
		targetURL, err := normalizeTargetURL(target.URL)
		if err != nil {
			return Config{}, fmt.Errorf("目标 #%d: %w", i+1, err)
		}
		name := strings.TrimSpace(target.Name)
		if name == "" {
			parsed, _ := url.Parse(targetURL)
			name = parsed.Hostname()
			if name == "" {
				name = fmt.Sprintf("Target %d", i+1)
			}
		}
		if len(name) > 80 {
			name = name[:80]
		}
		targets = append(targets, Target{Name: name, URL: targetURL, Enabled: target.Enabled})
	}
	if len(targets) == 0 {
		targets = DefaultConfig().Targets
	}
	out.Targets = targets
	return out, nil
}

func normalizeTargetURL(value string) (string, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return "", fmt.Errorf("URL 不能为空")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" {
		return "", fmt.Errorf("目标 URL 无效")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("目标 URL 只支持 http/https")
	}
	return raw, nil
}

func normalizeProxy(proxyCfg ProxyConfig) (ProxyConfig, error) {
	raw := strings.TrimSpace(proxyCfg.URL)
	if !proxyCfg.Enabled {
		return ProxyConfig{Enabled: false, URL: raw}, nil
	}
	if raw == "" {
		return ProxyConfig{}, fmt.Errorf("启用代理时必须填写代理地址")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" {
		return ProxyConfig{}, fmt.Errorf("代理地址无效")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https", "socks5", "socks5h":
	default:
		return ProxyConfig{}, fmt.Errorf("代理只支持 http/https/socks5/socks5h")
	}
	return ProxyConfig{Enabled: true, URL: raw}, nil
}

func runChecks(cfg runConfig) (CheckResponse, error) {
	proxyCfg, err := normalizeProxy(cfg.Proxy)
	if err != nil {
		return CheckResponse{}, err
	}
	base := Config{Timeout: cfg.Timeout, AutoRefresh: cfg.AutoRefresh, Targets: cfg.Targets}
	normalized, err := NormalizeConfig(base)
	if err != nil {
		return CheckResponse{}, err
	}
	enabled := make([]Target, 0, len(normalized.Targets))
	for _, target := range normalized.Targets {
		if target.Enabled {
			enabled = append(enabled, target)
		}
	}
	started := time.Now()
	if len(enabled) == 0 {
		now := time.Now().UTC().Format(time.RFC3339)
		return CheckResponse{Summary: Summary{CheckedAt: now}, Results: []TargetResult{}}, nil
	}

	client, err := buildHTTPClient(proxyCfg, normalized.Timeout)
	if err != nil {
		return CheckResponse{}, err
	}
	results := make([]TargetResult, len(enabled))
	workers := len(enabled)
	if workers > maxWorkers {
		workers = maxWorkers
	}
	jobs := make(chan struct {
		index  int
		target Target
	})
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				results[job.index] = checkTarget(client, job.target, normalized.Timeout)
			}
		}()
	}
	for i, target := range enabled {
		jobs <- struct {
			index  int
			target Target
		}{i, target}
	}
	close(jobs)
	wg.Wait()

	okCount := 0
	for _, result := range results {
		if result.Status == "OK" {
			okCount++
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	return CheckResponse{
		Summary: Summary{
			Total:      len(results),
			OK:         okCount,
			Fail:       len(results) - okCount,
			DurationMs: int(time.Since(started).Milliseconds()),
			CheckedAt:  now,
		},
		Results: results,
	}, nil
}

func buildHTTPClient(proxyCfg ProxyConfig, timeoutSec int) (*http.Client, error) {
	timeout := time.Duration(timeoutSec) * time.Second
	dialer := &net.Dialer{Timeout: timeout, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          32,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   timeout,
		ExpectContinueTimeout: time.Second,
	}
	if proxyCfg.Enabled {
		parsed, err := url.Parse(proxyCfg.URL)
		if err != nil {
			return nil, fmt.Errorf("代理解析失败：%w", err)
		}
		switch strings.ToLower(parsed.Scheme) {
		case "http", "https":
			transport.Proxy = http.ProxyURL(parsed)
		case "socks5", "socks5h":
			var auth *proxy.Auth
			if parsed.User != nil {
				password, _ := parsed.User.Password()
				auth = &proxy.Auth{User: parsed.User.Username(), Password: password}
			}
			address := parsed.Host
			if !strings.Contains(address, ":") {
				address = net.JoinHostPort(address, "1080")
			}
			dialer, err := proxy.SOCKS5("tcp", address, auth, proxy.Direct)
			if err != nil {
				return nil, fmt.Errorf("代理解析失败：%w", err)
			}
			if contextDialer, ok := dialer.(proxy.ContextDialer); ok {
				transport.DialContext = contextDialer.DialContext
			} else {
				transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
					return dialer.Dial(network, address)
				}
			}
		default:
			return nil, fmt.Errorf("不支持的代理协议：%s", parsed.Scheme)
		}
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}, nil
}

func checkTarget(client *http.Client, target Target, timeoutSec int) TargetResult {
	result := doHTTP(client, target.URL, http.MethodHead, timeoutSec)
	if !result.OK {
		result = doHTTP(client, target.URL, http.MethodGet, timeoutSec)
	}
	return TargetResult{
		Name:      target.Name,
		URL:       target.URL,
		Status:    result.Status,
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
		HTTP:      result,
	}
}

func doHTTP(client *http.Client, rawURL, method string, timeoutSec int) HTTPResult {
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
	if err != nil {
		return failedHTTP(int(time.Since(started).Milliseconds()), "构建请求失败："+err.Error())
	}
	req.Header.Set("User-Agent", "SubMerge-NetCheck/1.0")
	req.Header.Set("Accept", "*/*")
	resp, err := client.Do(req)
	elapsed := int(time.Since(started).Milliseconds())
	if err != nil {
		return failedHTTP(elapsed, humanError(err, timeoutSec))
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode <= 0 {
		return failedHTTP(elapsed, "访问失败：目标没有返回有效响应")
	}
	effectiveURL := rawURL
	if resp.Request != nil && resp.Request.URL != nil {
		effectiveURL = resp.Request.URL.String()
	}
	return HTTPResult{
		OK:           true,
		Status:       "OK",
		Code:         resp.StatusCode,
		TimeMs:       elapsed,
		Timing:       Timing{TotalMs: elapsed, FirstByteMs: elapsed},
		EffectiveURL: effectiveURL,
	}
}

func failedHTTP(elapsed int, message string) HTTPResult {
	return HTTPResult{
		Status: "FAIL",
		TimeMs: elapsed,
		Timing: Timing{TotalMs: elapsed},
		Error:  message,
	}
}

func humanError(err error, timeoutSec int) string {
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "timeout"), strings.Contains(message, "deadline exceeded"):
		return fmt.Sprintf("访问超时：%d 秒内没有完成连接或响应", timeoutSec)
	case strings.Contains(message, "no such host"), strings.Contains(message, "lookup"):
		return "域名解析失败：无法解析目标网站或代理地址"
	case strings.Contains(message, "connection refused"):
		return "连接失败：目标或代理拒绝连接"
	case strings.Contains(message, "connection reset"):
		return "连接中断：连接被重置"
	case strings.Contains(message, "tls"), strings.Contains(message, "certificate"), strings.Contains(message, "x509"):
		return "TLS 握手失败：HTTPS 连接没有建立成功"
	default:
		if len(err.Error()) > 160 {
			return "访问失败：" + err.Error()[:160] + "…"
		}
		return "访问失败：" + err.Error()
	}
}

var _ = context.Background
