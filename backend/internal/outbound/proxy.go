package outbound

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

var supportedSchemes = map[string]bool{
	"http":    true,
	"https":   true,
	"socks5":  true,
	"socks5h": true,
}

// ValidateProxyURL validates a deployment or user supplied proxy URL without
// exposing credentials in the returned error.
func ValidateProxyURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	u, err := parseProxyURL(raw)
	if err != nil {
		return err
	}
	if u.User != nil && u.User.Username() == "" {
		return errors.New("proxy username cannot be empty")
	}
	return nil
}

// MaskURL returns a safe display value for a proxy URL.
func MaskURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "已配置代理"
	}
	if u.User == nil {
		return u.String()
	}
	username := u.User.Username()
	u.User = nil
	base := u.String()
	prefix := u.Scheme + "://"
	if !strings.HasPrefix(base, prefix) {
		return "已配置代理"
	}
	return prefix + url.User(username).String() + ":***@" + strings.TrimPrefix(base, prefix)
}

// NewTransport creates a transport with an optional HTTP(S) or SOCKS5 proxy.
// The caller remains responsible for validating the destination URL.
func NewTransport(proxyURL string, timeout time.Duration) (*http.Transport, error) {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	dialer := &net.Dialer{Timeout: timeout, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   timeout,
		ExpectContinueTimeout: time.Second,
	}
	if err := ConfigureTransport(transport, proxyURL); err != nil {
		return nil, err
	}
	return transport, nil
}

// ConfigureTransport applies proxy routing to an existing transport while
// preserving its caller-specific dial and SSRF settings.
func ConfigureTransport(transport *http.Transport, proxyURL string) error {
	if transport == nil {
		return errors.New("proxy transport is nil")
	}
	proxyURL = strings.TrimSpace(proxyURL)
	if proxyURL == "" {
		return nil
	}
	u, err := parseProxyURL(proxyURL)
	if err != nil {
		return err
	}
	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "http", "https":
		transport.Proxy = http.ProxyURL(u)
	case "socks5", "socks5h":
		address := u.Host
		if u.Port() == "" {
			address = net.JoinHostPort(u.Hostname(), "1080")
		}
		var auth *proxy.Auth
		if u.User != nil {
			password, _ := u.User.Password()
			auth = &proxy.Auth{User: u.User.Username(), Password: password}
		}
		dialer, err := proxy.SOCKS5("tcp", address, auth, proxy.Direct)
		if err != nil {
			return errors.New("proxy dialer could not be created")
		}
		if contextDialer, ok := dialer.(proxy.ContextDialer); ok {
			transport.DialContext = contextDialer.DialContext
		} else {
			transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
				return dialer.Dial(network, address)
			}
		}
	default:
		return fmt.Errorf("unsupported proxy scheme: %s", scheme)
	}
	return nil
}

func parseProxyURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Hostname() == "" || !supportedSchemes[strings.ToLower(u.Scheme)] {
		return nil, errors.New("proxy URL must use http/https/socks5/socks5h with a host")
	}
	if u.Port() != "" {
		port := u.Port()
		for _, c := range port {
			if c < '0' || c > '9' {
				return nil, errors.New("proxy port is invalid")
			}
		}
		portValue := 0
		for _, c := range port {
			portValue = portValue*10 + int(c-'0')
			if portValue > 65535 {
				return nil, errors.New("proxy port is invalid")
			}
		}
		if portValue == 0 {
			return nil, errors.New("proxy port is invalid")
		}
	}
	return u, nil
}
