package netcheck

import (
	"fmt"
	"net/url"
	"strings"
)

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
