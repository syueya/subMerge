package systemsettings

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/submerge/submerge/backend/internal/config"
	"github.com/submerge/submerge/backend/internal/outbound"
)

func TrustedProxyList(raw string) []string {
	items := make([]string, 0)
	for _, item := range strings.Split(raw, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}

func durationFromSeconds(seconds int) (time.Duration, error) {
	if seconds <= 0 {
		return 0, fmt.Errorf("timeout must be greater than zero")
	}
	return time.Duration(seconds) * time.Second, nil
}

func parseTimeout(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if seconds, err := strconv.Atoi(raw); err == nil {
		return durationFromSeconds(seconds)
	}
	duration, err := time.ParseDuration(raw)
	if err != nil || duration <= 0 {
		if err == nil {
			err = fmt.Errorf("must be greater than zero")
		}
		return 0, fmt.Errorf("invalid timeout %q: %w", raw, err)
	}
	return duration, nil
}

func assign(s *Settings, key, value string) error {
	value = strings.TrimSpace(value)
	switch key {
	case KeySourceFetchUA:
		s.SourceFetchUA = value
	case KeySourceFetchTimeout:
		v, err := parseTimeout(value)
		if err != nil {
			return fmt.Errorf("source fetch timeout: %w", err)
		}
		s.SourceFetchTimeout = v
	case KeySourceMaxBytes:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("source max bytes: %w", err)
		}
		s.SourceMaxBytes = v
	case KeySourceRefreshHours:
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("source refresh interval: %w", err)
		}
		s.RefreshInterval = time.Duration(v) * time.Hour
	case KeyGeoIPURL:
		s.GeoIPURL = value
	case KeyGeoSiteURL:
		s.GeoSiteURL = value
	case KeyGeoDBURL:
		s.GeoDBURL = value
	case KeyGeoASNURL:
		s.GeoASNURL = value
	case KeyIPGeoURL:
		s.IPGeoURL = value
	case KeyIPGeoTimeout:
		v, err := parseTimeout(value)
		if err != nil {
			return fmt.Errorf("ip geo timeout: %w", err)
		}
		s.IPGeoTimeout = v
	case KeyLogOutput:
		s.LogOutput = value
	case KeyDebugLogging:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("debug logging: %w", err)
		}
		s.DebugLogging = v
	case KeyLogRetentionDays:
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("log retention days: %w", err)
		}
		s.LogRetentionDays = v
	case KeyProxyEnabled:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("proxy enabled: %w", err)
		}
		s.ProxyEnabled = v
	case KeyProxyURL:
		s.ProxyURL = value
	case KeyPublicBaseURL:
		s.PublicBaseURL = strings.TrimRight(value, "/")
	case KeyTrustedProxies:
		s.TrustedProxies = value
	case KeyCookieSecure:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("cookie secure: %w", err)
		}
		s.CookieSecure = v
	default:
		return fmt.Errorf("unknown system setting %q", key)
	}
	return nil
}

func applyUpdate(s *Settings, req UpdateRequest) error {
	if req.SourceFetchUA != nil {
		s.SourceFetchUA = *req.SourceFetchUA
	}
	if req.SourceFetchTimeout != nil {
		v, err := durationFromSeconds(*req.SourceFetchTimeout)
		if err != nil {
			return err
		}
		s.SourceFetchTimeout = v
	}
	if req.SourceMaxBytes != nil {
		s.SourceMaxBytes = *req.SourceMaxBytes
	}
	if req.RefreshInterval != nil {
		s.RefreshInterval = time.Duration(*req.RefreshInterval) * time.Hour
	}
	if req.GeoIPURL != nil {
		s.GeoIPURL = *req.GeoIPURL
	}
	if req.GeoSiteURL != nil {
		s.GeoSiteURL = *req.GeoSiteURL
	}
	if req.GeoDBURL != nil {
		s.GeoDBURL = *req.GeoDBURL
	}
	if req.GeoASNURL != nil {
		s.GeoASNURL = *req.GeoASNURL
	}
	if req.IPGeoURL != nil {
		s.IPGeoURL = *req.IPGeoURL
	}
	if req.IPGeoTimeout != nil {
		v, err := durationFromSeconds(*req.IPGeoTimeout)
		if err != nil {
			return err
		}
		s.IPGeoTimeout = v
	}
	if req.LogOutput != nil {
		s.LogOutput = *req.LogOutput
	}
	if req.DebugLogging != nil {
		s.DebugLogging = *req.DebugLogging
	}
	if req.LogRetentionDays != nil {
		s.LogRetentionDays = *req.LogRetentionDays
	}
	if req.ProxyEnabled != nil {
		s.ProxyEnabled = *req.ProxyEnabled
	}
	if req.ProxyURL != nil {
		s.ProxyURL = strings.TrimSpace(*req.ProxyURL)
		if s.ProxyURL == "" {
			s.ProxyEnabled = false
		}
	}
	if req.PublicBaseURL != nil {
		s.PublicBaseURL = strings.TrimRight(strings.TrimSpace(*req.PublicBaseURL), "/")
	}
	if req.TrustedProxies != nil {
		s.TrustedProxies = strings.TrimSpace(*req.TrustedProxies)
	}
	if req.CookieSecure != nil {
		s.CookieSecure = *req.CookieSecure
	}
	return nil
}

func validate(s Settings) error {
	if strings.TrimSpace(s.SourceFetchUA) == "" {
		return fmt.Errorf("source user-agent cannot be empty")
	}
	if s.SourceFetchTimeout <= 0 {
		return fmt.Errorf("source fetch timeout must be greater than zero")
	}
	if s.SourceMaxBytes <= 0 {
		return fmt.Errorf("source max bytes must be greater than zero")
	}
	if s.RefreshInterval <= 0 || s.RefreshInterval%time.Hour != 0 {
		return fmt.Errorf("source refresh interval must be a positive number of hours")
	}
	for key, raw := range map[string]string{KeyGeoIPURL: s.GeoIPURL, KeyGeoSiteURL: s.GeoSiteURL, KeyGeoDBURL: s.GeoDBURL, KeyGeoASNURL: s.GeoASNURL, KeyIPGeoURL: s.IPGeoURL} {
		u, err := url.Parse(strings.TrimSpace(raw))
		if err != nil || u.Scheme != "https" || u.Host == "" {
			return fmt.Errorf("%s must be a valid HTTPS URL", key)
		}
	}
	if s.IPGeoTimeout <= 0 {
		return fmt.Errorf("ip geo timeout must be greater than zero")
	}
	if s.LogOutput != "console" && s.LogOutput != "file" && s.LogOutput != "both" && s.LogOutput != "none" {
		return fmt.Errorf("invalid log output")
	}
	if s.LogRetentionDays < 0 {
		return fmt.Errorf("log retention days cannot be negative")
	}
	if err := config.ValidatePublicBaseURL(s.PublicBaseURL); err != nil {
		return err
	}
	for _, raw := range strings.Split(s.TrustedProxies, ",") {
		item := strings.TrimSpace(raw)
		if item == "" {
			continue
		}
		if net.ParseIP(item) != nil {
			continue
		}
		if _, _, err := net.ParseCIDR(item); err != nil {
			return fmt.Errorf("trusted proxies contains invalid IP or CIDR %q", item)
		}
	}
	if s.ProxyURL != "" {
		if err := outbound.ValidateProxyURL(s.ProxyURL); err != nil {
			return err
		}
	}
	return nil
}

func changedKeys(req UpdateRequest) []string {
	keys := []string{}
	if req.SourceFetchUA != nil {
		keys = append(keys, KeySourceFetchUA)
	}
	if req.SourceFetchTimeout != nil {
		keys = append(keys, KeySourceFetchTimeout)
	}
	if req.SourceMaxBytes != nil {
		keys = append(keys, KeySourceMaxBytes)
	}
	if req.RefreshInterval != nil {
		keys = append(keys, KeySourceRefreshHours)
	}
	if req.GeoIPURL != nil {
		keys = append(keys, KeyGeoIPURL)
	}
	if req.GeoSiteURL != nil {
		keys = append(keys, KeyGeoSiteURL)
	}
	if req.GeoDBURL != nil {
		keys = append(keys, KeyGeoDBURL)
	}
	if req.GeoASNURL != nil {
		keys = append(keys, KeyGeoASNURL)
	}
	if req.IPGeoURL != nil {
		keys = append(keys, KeyIPGeoURL)
	}
	if req.IPGeoTimeout != nil {
		keys = append(keys, KeyIPGeoTimeout)
	}
	if req.LogOutput != nil {
		keys = append(keys, KeyLogOutput)
	}
	if req.DebugLogging != nil {
		keys = append(keys, KeyDebugLogging)
	}
	if req.LogRetentionDays != nil {
		keys = append(keys, KeyLogRetentionDays)
	}
	if req.ProxyEnabled != nil {
		keys = append(keys, KeyProxyEnabled)
	}
	if req.ProxyURL != nil {
		keys = append(keys, KeyProxyURL)
	}
	if req.PublicBaseURL != nil {
		keys = append(keys, KeyPublicBaseURL)
	}
	if req.TrustedProxies != nil {
		keys = append(keys, KeyTrustedProxies)
	}
	if req.CookieSecure != nil {
		keys = append(keys, KeyCookieSecure)
	}
	return keys
}

func valueFor(s Settings, key string, box interface{ Encrypt(string) (string, error) }) (string, bool, error) {
	var value string
	encrypted := false
	switch key {
	case KeySourceFetchUA:
		value = s.SourceFetchUA
	case KeySourceFetchTimeout:
		value = s.SourceFetchTimeout.String()
	case KeySourceMaxBytes:
		value = strconv.FormatInt(s.SourceMaxBytes, 10)
	case KeySourceRefreshHours:
		value = strconv.Itoa(int(s.RefreshInterval / time.Hour))
	case KeyGeoIPURL:
		value = s.GeoIPURL
	case KeyGeoSiteURL:
		value = s.GeoSiteURL
	case KeyGeoDBURL:
		value = s.GeoDBURL
	case KeyGeoASNURL:
		value = s.GeoASNURL
	case KeyIPGeoURL:
		value = s.IPGeoURL
	case KeyIPGeoTimeout:
		value = s.IPGeoTimeout.String()
	case KeyLogOutput:
		value = s.LogOutput
	case KeyDebugLogging:
		value = strconv.FormatBool(s.DebugLogging)
	case KeyLogRetentionDays:
		value = strconv.Itoa(s.LogRetentionDays)
	case KeyProxyEnabled:
		value = strconv.FormatBool(s.ProxyEnabled)
	case KeyProxyURL:
		value = s.ProxyURL
		encrypted = true
	case KeyPublicBaseURL:
		value = s.PublicBaseURL
	case KeyTrustedProxies:
		value = s.TrustedProxies
	case KeyCookieSecure:
		value = strconv.FormatBool(s.CookieSecure)
	default:
		return "", false, fmt.Errorf("unknown system setting %q", key)
	}
	if encrypted {
		cipher, err := box.Encrypt(value)
		return cipher, true, err
	}
	return value, false, nil
}
