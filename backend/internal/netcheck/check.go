package netcheck

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/submerge/submerge/backend/internal/outbound"
)

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
	transport, err := outbound.NewTransport(proxyURL(proxyCfg), timeout)
	if err != nil {
		return nil, fmt.Errorf("代理解析失败：%w", err)
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}, nil
}

func proxyURL(proxyCfg ProxyConfig) string {
	if !proxyCfg.Enabled {
		return ""
	}
	return proxyCfg.URL
}

func checkTarget(client *http.Client, target Target, timeoutSec int) TargetResult {
	// 先 HEAD，失败再 GET；耗时累计两次探测，避免只显示最后一次
	result := doHTTP(client, target.URL, http.MethodHead, timeoutSec)
	if !result.OK {
		second := doHTTP(client, target.URL, http.MethodGet, timeoutSec)
		second.TimeMs += result.TimeMs
		second.Timing.TotalMs = second.TimeMs
		result = second
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
	raw := err.Error()
	message := strings.ToLower(raw)
	switch {
	case strings.Contains(message, "timeout"), strings.Contains(message, "deadline exceeded"):
		return fmt.Sprintf("访问超时：%d 秒内没有完成连接或响应", timeoutSec)
	case strings.Contains(message, "no such host"), strings.Contains(message, "lookup"):
		return "域名解析失败：无法解析目标网站或代理地址"
	case strings.Contains(message, "connection refused"),
		strings.Contains(message, "actively refused"),
		strings.Contains(message, "connectex"):
		// Windows connectex 文案不含 "connection refused"，需单独识别
		return "连接失败：目标或代理拒绝连接（" + raw + "）"
	case strings.Contains(message, "connection reset"):
		return "连接中断：连接被重置"
	case strings.Contains(message, "tls"), strings.Contains(message, "certificate"), strings.Contains(message, "x509"):
		return "TLS 握手失败：HTTPS 连接没有建立成功"
	default:
		// 保留完整原始错误，供前端表格省略 + 悬浮完整展示
		return "访问失败：" + raw
	}
}

var _ = context.Background
