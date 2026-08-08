package systemsettings

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/submerge/submerge/backend/internal/config"
	"github.com/submerge/submerge/backend/internal/crypto"
	"github.com/submerge/submerge/backend/internal/database"
	"github.com/submerge/submerge/backend/internal/outbound"
	"gorm.io/gorm"
)

const (
	KeySourceFetchUA      = "source.fetch.ua"
	KeySourceFetchTimeout = "source.fetch.timeout"
	KeySourceMaxBytes     = "source.fetch.max_bytes"
	KeySourceRefreshHours = "source.refresh.hours"
	KeyGeoIPURL           = "geo.geoip_url"
	KeyGeoSiteURL         = "geo.geosite_url"
	KeyGeoDBURL           = "geo.geodb_url"
	KeyGeoASNURL          = "geo.geoasn_url"
	KeyIPGeoURL           = "geo.ip_geo_url"
	KeyIPGeoTimeout       = "geo.ip_geo_timeout"
	KeyLogOutput          = "log.output"
	KeyDebugLogging       = "log.debug"
	KeyLogRetentionDays   = "log.retention_days"
	KeyProxyEnabled       = "outbound.proxy.enabled"
	KeyProxyURL           = "outbound.proxy.url"
	KeyPublicBaseURL      = "deployment.public_base_url"
	KeyTrustedProxies     = "deployment.trusted_proxies"
	KeyCookieSecure       = "deployment.cookie_secure"
)

type Settings struct {
	SourceFetchUA      string
	SourceFetchTimeout time.Duration
	SourceMaxBytes     int64
	RefreshInterval    time.Duration
	GeoIPURL           string
	GeoSiteURL         string
	GeoDBURL           string
	GeoASNURL          string
	IPGeoURL           string
	IPGeoTimeout       time.Duration
	LogOutput          string
	DebugLogging       bool
	LogRetentionDays   int
	ProxyEnabled       bool
	ProxyURL           string
	PublicBaseURL      string
	TrustedProxies     string
	CookieSecure       bool
}

type UpdateRequest struct {
	SourceFetchUA      *string `json:"sourceFetchUA"`
	SourceFetchTimeout *int    `json:"sourceFetchTimeout"`
	SourceMaxBytes     *int64  `json:"sourceMaxBytes"`
	RefreshInterval    *int    `json:"refreshInterval"`
	GeoIPURL           *string `json:"geoipUrl"`
	GeoSiteURL         *string `json:"geositeUrl"`
	GeoDBURL           *string `json:"geodbUrl"`
	GeoASNURL          *string `json:"geoasnUrl"`
	IPGeoURL           *string `json:"ipGeoUrl"`
	IPGeoTimeout       *int    `json:"ipGeoTimeout"`
	LogOutput          *string `json:"logOutput"`
	DebugLogging       *bool   `json:"debugLogging"`
	LogRetentionDays   *int    `json:"logRetentionDays"`
	ProxyEnabled       *bool   `json:"proxyEnabled"`
	ProxyURL           *string `json:"proxyUrl"`
	PublicBaseURL      *string `json:"publicBaseUrl"`
	TrustedProxies     *string `json:"trustedProxies"`
	CookieSecure       *bool   `json:"cookieSecure"`
}

type View struct {
	Settings        SettingsView      `json:"settings"`
	Source          map[string]string `json:"source"`
	Override        map[string]bool   `json:"override"`
	RestartRequired bool              `json:"restartRequired"`
}

type SettingsView struct {
	SourceFetchUA      string `json:"sourceFetchUA"`
	SourceFetchTimeout int    `json:"sourceFetchTimeout"`
	SourceMaxBytes     int64  `json:"sourceMaxBytes"`
	RefreshInterval    int    `json:"refreshInterval"`
	GeoIPURL           string `json:"geoipUrl"`
	GeoSiteURL         string `json:"geositeUrl"`
	GeoDBURL           string `json:"geodbUrl"`
	GeoASNURL          string `json:"geoasnUrl"`
	IPGeoURL           string `json:"ipGeoUrl"`
	IPGeoTimeout       int    `json:"ipGeoTimeout"`
	LogOutput          string `json:"logOutput"`
	DebugLogging       bool   `json:"debugLogging"`
	LogRetentionDays   int    `json:"logRetentionDays"`
	ProxyEnabled       bool   `json:"proxyEnabled"`
	ProxyConfigured    bool   `json:"proxyConfigured"`
	ProxyMaskedURL     string `json:"proxyMaskedUrl,omitempty"`
	PublicBaseURL      string `json:"publicBaseUrl"`
	TrustedProxies     string `json:"trustedProxies"`
	CookieSecure       bool   `json:"cookieSecure"`
}

type Manager struct {
	db              *gorm.DB
	box             *crypto.Box
	apply           func(Settings) error
	mu              sync.RWMutex
	current         Settings
	overrides       map[string]bool
	production      bool
	restartRequired bool
}

func Defaults() Settings {
	return Settings{
		SourceFetchUA:      config.DefaultSourceFetchUA,
		SourceFetchTimeout: config.DefaultSourceFetchTimeout,
		SourceMaxBytes:     config.DefaultSourceMaxBytes,
		RefreshInterval:    config.DefaultRefreshInterval,
		GeoIPURL:           config.DefaultGeoIPURL,
		GeoSiteURL:         config.DefaultGeoSiteURL,
		GeoDBURL:           config.DefaultMetaDBURL,
		GeoASNURL:          config.DefaultASNURL,
		IPGeoURL:           config.DefaultIPGeoURL,
		IPGeoTimeout:       config.DefaultIPGeoTimeout,
		LogOutput:          "both",
		DebugLogging:       true,
		LogRetentionDays:   7,
		PublicBaseURL:      config.DefaultPublicBaseURL,
		TrustedProxies:     "",
		CookieSecure:       false,
	}
}

func NewManager(db *gorm.DB, box *crypto.Box, production bool, apply func(Settings) error) (*Manager, error) {
	m := &Manager{db: db, box: box, apply: apply, overrides: map[string]bool{}, production: production}
	settings, err := m.load(m.defaults())
	if err != nil {
		return nil, err
	}
	if err := validate(settings); err != nil {
		return nil, err
	}
	if apply != nil {
		if err := apply(settings); err != nil {
			return nil, err
		}
	}
	m.current = settings
	return m, nil
}

func (m *Manager) defaults() Settings {
	defaults := Defaults()
	if m.production {
		defaults.DebugLogging = false
	}
	return defaults
}

func (m *Manager) load(settings Settings) (Settings, error) {
	var rows []database.SystemSetting
	if err := m.db.Find(&rows).Error; err != nil {
		return settings, err
	}
	for _, row := range rows {
		value := row.Value
		if row.Encrypted {
			plain, err := m.box.Decrypt(value)
			if err != nil {
				return settings, fmt.Errorf("decrypt system setting: %w", err)
			}
			value = plain
		}
		if err := assign(&settings, row.Key, value); err != nil {
			return settings, err
		}
		m.overrides[row.Key] = true
	}
	// Compatibility with the former single-row proxy table.
	if !m.overrides[KeyProxyURL] {
		var old database.OutboundProxySetting
		if err := m.db.First(&old, 1).Error; err == nil && old.HasOverride {
			plain, err := m.box.Decrypt(old.URLCiphertext)
			if err != nil {
				return settings, err
			}
			tx := m.db.Begin()
			if tx.Error != nil {
				return settings, tx.Error
			}
			urlRow := database.SystemSetting{Key: KeyProxyURL, Value: old.URLCiphertext, Encrypted: true}
			enabledRow := database.SystemSetting{Key: KeyProxyEnabled, Value: strconv.FormatBool(old.Enabled), Encrypted: false}
			if err := tx.Where("key = ?", KeyProxyURL).Assign(urlRow).FirstOrCreate(&urlRow).Error; err != nil {
				tx.Rollback()
				return settings, err
			}
			if err := tx.Where("key = ?", KeyProxyEnabled).Assign(enabledRow).FirstOrCreate(&enabledRow).Error; err != nil {
				tx.Rollback()
				return settings, err
			}
			if err := tx.Delete(&database.OutboundProxySetting{}, 1).Error; err != nil {
				tx.Rollback()
				return settings, err
			}
			if err := tx.Commit().Error; err != nil {
				return settings, err
			}
			settings.ProxyURL, settings.ProxyEnabled = plain, old.Enabled
			m.overrides[KeyProxyURL], m.overrides[KeyProxyEnabled] = true, true
		}
	}
	return settings, nil
}

func (m *Manager) View() View {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viewLocked()
}

func (m *Manager) viewLocked() View {
	s := m.current
	sv := SettingsView{SourceFetchUA: s.SourceFetchUA, SourceFetchTimeout: int(s.SourceFetchTimeout / time.Second), SourceMaxBytes: s.SourceMaxBytes, RefreshInterval: int(s.RefreshInterval / time.Hour), GeoIPURL: s.GeoIPURL, GeoSiteURL: s.GeoSiteURL, GeoDBURL: s.GeoDBURL, GeoASNURL: s.GeoASNURL, IPGeoURL: s.IPGeoURL, IPGeoTimeout: int(s.IPGeoTimeout / time.Second), LogOutput: s.LogOutput, DebugLogging: s.DebugLogging, LogRetentionDays: s.LogRetentionDays, ProxyEnabled: s.ProxyEnabled, ProxyConfigured: s.ProxyURL != "", ProxyMaskedURL: outbound.MaskURL(s.ProxyURL), PublicBaseURL: s.PublicBaseURL, TrustedProxies: s.TrustedProxies, CookieSecure: s.CookieSecure}
	return View{Settings: sv, Source: m.sources(), Override: copyBoolMap(m.overrides), RestartRequired: m.restartRequired}
}

func (m *Manager) sources() map[string]string {
	result := map[string]string{}
	for _, key := range allKeys {
		if m.overrides[key] {
			result[key] = "web"
		} else {
			result[key] = "default"
		}
	}
	return result
}
func copyBoolMap(in map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func containsKey(keys []string, target string) bool {
	for _, key := range keys {
		if key == target {
			return true
		}
	}
	return false
}

func (m *Manager) Save(req UpdateRequest) (View, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	candidate := m.current
	if err := applyUpdate(&candidate, req); err != nil {
		return View{}, err
	}
	if err := validate(candidate); err != nil {
		return View{}, err
	}
	if m.apply != nil {
		if err := m.apply(candidate); err != nil {
			return View{}, err
		}
	}
	if err := m.persist(candidate, req); err != nil {
		_ = m.apply(m.current)
		return View{}, err
	}
	m.current = candidate
	if containsKey(changedKeys(req), KeyTrustedProxies) {
		m.restartRequired = true
	}
	return m.viewLocked(), nil
}

func (m *Manager) Reset() (View, error) { return m.ResetKeys(allKeys) }
func (m *Manager) ResetKeys(keys []string) (View, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	candidate := m.defaults()
	if err := m.loadExcept(&candidate, keys); err != nil {
		return View{}, err
	}
	if err := validate(candidate); err != nil {
		return View{}, err
	}
	if m.apply != nil {
		if err := m.apply(candidate); err != nil {
			return View{}, err
		}
	}
	tx := m.db.Begin()
	if tx.Error != nil {
		return View{}, tx.Error
	}
	for _, key := range keys {
		if err := tx.Where("key = ?", key).Delete(&database.SystemSetting{}).Error; err != nil {
			tx.Rollback()
			_ = m.apply(m.current)
			return View{}, err
		}
	}
	if err := tx.Commit().Error; err != nil {
		_ = m.apply(m.current)
		return View{}, err
	}
	m.current = candidate
	for _, key := range keys {
		delete(m.overrides, key)
	}
	if containsKey(keys, KeyTrustedProxies) {
		m.restartRequired = true
	}
	return m.viewLocked(), nil
}

func (m *Manager) loadExcept(settings *Settings, excluded []string) error {
	ignore := map[string]bool{}
	for _, k := range excluded {
		ignore[k] = true
	}
	var rows []database.SystemSetting
	if err := m.db.Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if ignore[row.Key] {
			continue
		}
		value := row.Value
		if row.Encrypted {
			plain, err := m.box.Decrypt(value)
			if err != nil {
				return err
			}
			value = plain
		}
		if err := assign(settings, row.Key, value); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) persist(settings Settings, req UpdateRequest) error {
	keys := changedKeys(req)
	tx := m.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	for _, key := range keys {
		value, encrypted, err := valueFor(settings, key, m.box)
		if err != nil {
			tx.Rollback()
			return err
		}
		row := database.SystemSetting{Key: key, Value: value, Encrypted: encrypted}
		if err := tx.Where("key = ?", key).Assign(row).FirstOrCreate(&row).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}
	for _, key := range keys {
		m.overrides[key] = true
	}
	return nil
}

func (m *Manager) proxyView() outbound.View {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return proxyView(m.viewLocked().Settings, m.overrides)
}

func (m *Manager) SaveProxy(req outbound.UpdateRequest) (outbound.View, error) {
	reqURL := req.URL
	enabled := req.Enabled
	update := UpdateRequest{ProxyURL: &reqURL, ProxyEnabled: enabled}
	m.mu.RLock()
	current := m.current
	m.mu.RUnlock()
	if strings.TrimSpace(req.URL) == "" && current.ProxyURL != "" {
		update.ProxyURL = nil
	}
	view, err := m.Save(update)
	if err != nil {
		return outbound.View{}, err
	}
	return proxyView(view.Settings, view.Override), nil
}
func (m *Manager) ResetProxy() (outbound.View, error) {
	view, err := m.ResetKeys([]string{KeyProxyURL, KeyProxyEnabled})
	if err != nil {
		return outbound.View{}, err
	}
	return proxyView(view.Settings, view.Override), nil
}
func (m *Manager) SaveProxyCompat(req outbound.UpdateRequest) (outbound.View, error) {
	return m.SaveProxy(req)
}
func proxyView(s SettingsView, overrides map[string]bool) outbound.View {
	return outbound.View{Enabled: s.ProxyEnabled, Configured: s.ProxyConfigured, Source: map[bool]string{true: "web", false: "default"}[overrides[KeyProxyURL]], MaskedURL: s.ProxyMaskedURL, HasOverride: overrides[KeyProxyURL]}
}

var allKeys = []string{KeySourceFetchUA, KeySourceFetchTimeout, KeySourceMaxBytes, KeySourceRefreshHours, KeyGeoIPURL, KeyGeoSiteURL, KeyGeoDBURL, KeyGeoASNURL, KeyIPGeoURL, KeyIPGeoTimeout, KeyLogOutput, KeyDebugLogging, KeyLogRetentionDays, KeyProxyEnabled, KeyProxyURL, KeyPublicBaseURL, KeyTrustedProxies, KeyCookieSecure}
