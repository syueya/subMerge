package outbound

import (
	"fmt"
	"strings"
	"sync"

	"github.com/submerge/submerge/backend/internal/crypto"
	"github.com/submerge/submerge/backend/internal/database"
	"gorm.io/gorm"
)

const settingID uint = 1

type RuntimeConfig struct {
	Enabled bool
	URL     string
	Source  string
}

type View struct {
	Enabled     bool   `json:"enabled"`
	Configured  bool   `json:"configured"`
	Source      string `json:"source"`
	MaskedURL   string `json:"maskedUrl,omitempty"`
	HasOverride bool   `json:"hasOverride"`
}

type UpdateRequest struct {
	Enabled *bool  `json:"enabled"`
	URL     string `json:"url"`
}

type Manager struct {
	db       *gorm.DB
	box      *crypto.Box
	envURL   string
	apply    func(string) error
	mu       sync.RWMutex
	current  RuntimeConfig
	hasValue bool
}

func NewManager(db *gorm.DB, box *crypto.Box, envURL string, apply func(string) error) (*Manager, error) {
	envURL = strings.TrimSpace(envURL)
	if err := ValidateProxyURL(envURL); err != nil {
		return nil, fmt.Errorf("SOURCE_GEO_PROXY: %w", err)
	}
	m := &Manager{db: db, box: box, envURL: envURL, apply: apply}
	if err := m.load(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Manager) load() error {
	cfg := RuntimeConfig{Enabled: envURLEnabled(m.envURL), URL: m.envURL, Source: sourceEnv}
	var row database.OutboundProxySetting
	err := m.db.First(&row, settingID).Error
	if err == nil && row.HasOverride {
		plain, decErr := m.box.Decrypt(row.URLCiphertext)
		if decErr != nil {
			return fmt.Errorf("decrypt outbound proxy setting: %w", decErr)
		}
		if err := ValidateProxyURL(plain); err != nil {
			return fmt.Errorf("stored outbound proxy setting: %w", err)
		}
		cfg = RuntimeConfig{Enabled: row.Enabled, URL: plain, Source: sourceWeb}
	} else if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if m.apply != nil {
		if err := m.apply(cfg.URLIfEnabled()); err != nil {
			return err
		}
	}
	m.current = cfg
	m.hasValue = cfg.Source == sourceWeb
	return nil
}

const (
	sourceEnv = "environment"
	sourceWeb = "web"
)

func envURLEnabled(raw string) bool { return strings.TrimSpace(raw) != "" }

func (c RuntimeConfig) URLIfEnabled() string {
	if !c.Enabled {
		return ""
	}
	return c.URL
}

func (m *Manager) View() View {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return View{Enabled: m.current.Enabled, Configured: m.current.URL != "", Source: m.current.Source, MaskedURL: MaskURL(m.current.URL), HasOverride: m.hasValue}
}

func (m *Manager) Save(req UpdateRequest) (View, error) {
	raw := strings.TrimSpace(req.URL)
	m.mu.RLock()
	current := m.current
	m.mu.RUnlock()
	if raw == "" && current.URL != "" {
		raw = current.URL
	}
	if err := ValidateProxyURL(raw); err != nil {
		return View{}, err
	}
	enabled := current.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	if raw == "" {
		enabled = false
	}
	ciphertext, err := m.box.Encrypt(raw)
	if err != nil {
		return View{}, err
	}
	row := database.OutboundProxySetting{ID: settingID, Enabled: enabled, HasOverride: true, URLCiphertext: ciphertext}
	if err := m.db.Save(&row).Error; err != nil {
		return View{}, err
	}
	if m.apply != nil {
		if err := m.apply(func() string {
			if enabled {
				return raw
			}
			return ""
		}()); err != nil {
			return View{}, err
		}
	}
	m.mu.Lock()
	m.current = RuntimeConfig{Enabled: enabled, URL: raw, Source: sourceWeb}
	m.hasValue = true
	m.mu.Unlock()
	return m.View(), nil
}

func (m *Manager) Reset() (View, error) {
	if err := m.db.Delete(&database.OutboundProxySetting{}, settingID).Error; err != nil {
		return View{}, err
	}
	cfg := RuntimeConfig{Enabled: envURLEnabled(m.envURL), URL: m.envURL, Source: sourceEnv}
	if m.apply != nil {
		if err := m.apply(cfg.URLIfEnabled()); err != nil {
			return View{}, err
		}
	}
	m.mu.Lock()
	m.current = cfg
	m.hasValue = false
	m.mu.Unlock()
	return m.View(), nil
}
