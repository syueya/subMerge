package netcheck

import (
	"encoding/json"
	"fmt"

	"github.com/submerge/submerge/backend/internal/database"
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
	db             *gorm.DB
	systemProxyURL func() string
}

func NewService(db *gorm.DB, systemProxyURL func() string) *Service {
	return &Service{db: db, systemProxyURL: systemProxyURL}
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
var proxyInfo, proxyMode string
		if req.Proxy != nil {
			proxy := *req.Proxy
			if proxy.Enabled {
				if proxy.URL == "" {
					// 启用代理但未填地址时回退到系统代理
					if s.systemProxyURL != nil {
						proxy.URL = s.systemProxyURL()
					}
					if proxy.URL != "" {
						proxyInfo = "使用系统代理"
						proxyMode = "proxy"
					} else {
						proxyInfo = "未使用代理（直连）"
						proxyMode = "direct"
					}
				} else {
					proxyInfo = "使用代理"
					proxyMode = "proxy"
				}
			} else {
				proxyInfo = "未使用代理（直连）"
				proxyMode = "direct"
			}
			cfgRun := runConfig{
				Proxy:       proxy,
				Timeout:     cfg.Timeout,
				AutoRefresh: cfg.AutoRefresh,
				Targets:     cfg.Targets,
			}
			result, err := runChecks(cfgRun)
			if err != nil {
				return CheckResponse{}, err
			}
			result.ProxyInfo = proxyInfo
			result.ProxyMode = proxyMode
			return result, nil
		}
		result, err := runChecks(runConfig{Timeout: cfg.Timeout, AutoRefresh: cfg.AutoRefresh, Targets: cfg.Targets})
		if err != nil {
			return CheckResponse{}, err
		}
		result.ProxyInfo = "未使用代理（直连）"
		result.ProxyMode = "direct"
		return result, nil
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
