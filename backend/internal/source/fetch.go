package source

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"

	"github.com/submerge/submerge/backend/internal/applog"
	"github.com/submerge/submerge/backend/internal/config"
	"github.com/submerge/submerge/backend/internal/crypto"
	"github.com/submerge/submerge/backend/internal/outbound"
)

// newFetchHTTPClient 构造带 SSRF 防护的出站客户端：
// Dialer.Control 在真正建连时校验目标 IP，使校验与拨号使用同一次 DNS 解析，
// 避免低 TTL rebinding / TOCTOU。
func newFetchHTTPClient(timeout time.Duration) *http.Client {
	client, _ := newFetchHTTPClientWithProxy(timeout, "")
	return client
}

func newFetchHTTPClientWithProxy(timeout time.Duration, proxyURL string) (*http.Client, error) {
	var transport *http.Transport
	if strings.TrimSpace(proxyURL) != "" {
		var err error
		transport, err = outbound.NewTransport(proxyURL, 10*time.Second)
		if err != nil {
			return nil, err
		}
	} else {
		dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
		dialer.Control = func(network, address string, c syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return fmt.Errorf("解析拨号地址失败: %w", err)
			}
			ip := net.ParseIP(host)
			if ip == nil || isBlockedIP(ip) {
				return fmt.Errorf("上游主机解析到被禁止的地址")
			}
			return nil
		}
		transport = &http.Transport{
			DialContext:           dialer.DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if err := validateFetchURL(req.URL); err != nil {
				return err
			}
			if len(via) >= 5 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}, nil
}

// fetchResult 上游拉取结果（正文 + 可选流量元信息）
type fetchResult struct {
	Body     []byte
	UserInfo SubscriptionUserInfo
}

func (s *Service) fetch(rawURL string) (fetchResult, error) {
	// masked 用于所有对外错误信息，避免把含凭据的完整 URL 泄露到日志/DB/响应
	masked := crypto.MaskURL(rawURL)
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fetchResult{}, fmt.Errorf("URL 无效: %s", masked)
	}
	if err := validateFetchURL(u); err != nil {
		return fetchResult{}, err
	}
	s.clientMu.RLock()
	client := s.httpClient
	options := s.options
	s.clientMu.RUnlock()
	ctx, cancel := context.WithTimeout(context.Background(), options.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fetchResult{}, fmt.Errorf("构建 %s 的请求失败", masked)
	}
	// 机场常按 UA 裁剪节点列表：必须用 Clash Verge Rev 同款格式，
	// 否则可能少返回线路，并塞入「过滤掉N条线路」信息节点。
	ua := options.userAgent
	if ua == "" {
		ua = config.DefaultSourceFetchUA
	}
	req.Header.Set("User-Agent", ua)
	// 与常见客户端一致：不要额外加 Accept-Language 等易被风控的头
	req.Header.Set("Accept", "*/*")
	resp, err := client.Do(req)
	if err != nil {
		// *url.Error 会带上含 query 的完整 URL（机场订阅常含 token/密码），必须脱敏
		return fetchResult{}, fmt.Errorf("拉取 %s 失败: %s", masked, cleanFetchErr(err, rawURL))
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fetchResult{}, fmt.Errorf("上游返回状态码 %d", resp.StatusCode)
	}
	// 流量信息：与 Clash Verge 相同，读 Subscription-Userinfo（含 meta 前缀）
	userInfo, _ := ParseSubscriptionUserInfoFromHeaders(resp.Header)

	limited := io.LimitReader(resp.Body, options.maxBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return fetchResult{}, fmt.Errorf("读取上游响应失败: %s", cleanFetchErr(err, rawURL))
	}
	if int64(len(body)) > options.maxBytes {
		return fetchResult{}, fmt.Errorf("响应内容超过大小限制")
	}
	applog.Info("[fetch] 拉取成功 ua=%q status=%d bytes=%d userinfo=%v",
		ua, resp.StatusCode, len(body), userInfo.HasAny())
	return fetchResult{Body: body, UserInfo: userInfo}, nil
}

func validateFetchURL(u *url.URL) error {
	if u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return fmt.Errorf("不支持的 URL 协议或主机")
	}
	ips, err := net.LookupIP(u.Hostname())
	if err != nil {
		return fmt.Errorf("解析上游主机失败: %w", err)
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return fmt.Errorf("upstream host resolves to a blocked address")
		}
	}
	return nil
}

// cgnatNet CGNAT 段 100.64.0.0/10（RFC 6598），net.IP.IsPrivate 不覆盖
var cgnatNet = &net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}

// isBlockedIP 判断目标 IP 是否属于禁止直连的内网/保留地址（SSRF 防护）。
// 覆盖 loopback / RFC1918 / link-local / unspecified / multicast / CGNAT。
func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified() ||
		cgnatNet.Contains(ip)
}

// cleanFetchErr 剥离 *url.Error 内含的完整 URL（可能带 token/密码），
// 仅保留底层原因文本，并把偶然出现的原始 URL 片段替换为脱敏串。
func cleanFetchErr(err error, rawURL string) string {
	var uErr *url.Error
	msg := err.Error()
	if errors.Is(err, context.DeadlineExceeded) {
		return "请求超时"
	}
	if errors.Is(err, context.Canceled) {
		return "请求已取消"
	}
	if errors.As(err, &uErr) && uErr.Err != nil {
		msg = uErr.Err.Error()
	}
	masked := crypto.MaskURL(rawURL)
	if rawURL != "" && rawURL != masked {
		msg = strings.ReplaceAll(msg, rawURL, masked)
	}
	// 再扫一遍常见「scheme://user:pass@host」形态，防止 err 里是重编码后的变体
	if strings.Contains(msg, "://") && strings.Contains(msg, "@") {
		if u, parseErr := url.Parse(rawURL); parseErr == nil && u.User != nil {
			if ui := u.User.String(); ui != "" {
				msg = strings.ReplaceAll(msg, ui+"@", "***@")
			}
		}
	}
	return msg
}
