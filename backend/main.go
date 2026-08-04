package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/submerge/submerge/backend/internal/apikey"
	"github.com/submerge/submerge/backend/internal/applog"
	"github.com/submerge/submerge/backend/internal/audit"
	"github.com/submerge/submerge/backend/internal/auth"
	"github.com/submerge/submerge/backend/internal/config"
	"github.com/submerge/submerge/backend/internal/crypto"
	"github.com/submerge/submerge/backend/internal/database"
	"github.com/submerge/submerge/backend/internal/geo"
	"github.com/submerge/submerge/backend/internal/logs"
	"github.com/submerge/submerge/backend/internal/middleware"
	"github.com/submerge/submerge/backend/internal/netcheck"
	"github.com/submerge/submerge/backend/internal/publish"
	"github.com/submerge/submerge/backend/internal/rule"
	"github.com/submerge/submerge/backend/internal/server"
	"github.com/submerge/submerge/backend/internal/source"
	"github.com/submerge/submerge/backend/internal/subscription"
)

func main() {
	// 日志时间戳 / 按日切分默认 Asia/Shanghai；可用环境变量 TZ 覆盖
	if err := applog.InitTimezone(); err != nil {
		applog.Fatalf("timezone: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		// Setup 尚未完成，走标准库
		applog.Fatalf("config: %v", err)
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		applog.Fatalf("data dir: %v", err)
	}
	// 日志：console / file / both / none；文件默认 backend/log/submerge-YYYY-MM-DD.log
	if err := applog.Setup(cfg.LogOutput, cfg.LogDir, cfg.LogRetentionDays); err != nil {
		applog.Fatalf("log setup: %v", err)
	}
	defer applog.Close()

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		applog.Fatalf("database: %v", err)
	}

	// argon2id 强派生需要持久化盐（首次运行生成，0600 权限，存于数据目录）
	salt, err := crypto.LoadOrCreateSalt(filepath.Join(cfg.DataDir, "crypto.salt"))
	if err != nil {
		applog.Fatalf("crypto salt: %v", err)
	}
	box, err := crypto.NewBoxWithSalt(cfg.EncryptionKey, salt)
	if err != nil {
		applog.Fatalf("crypto: %v", err)
	}

	auditSvc := audit.NewService(db)
	authSvc := auth.NewService(db, cfg.SessionTTL)
	// 管理员仅通过网页首次注册创建，不在环境变量里配置

	sourceSvc := source.NewServiceWithUA(db, box, cfg.SourceFetchTimeout, cfg.SourceMaxBytes, cfg.SourceFetchUA)
	// 复位上次运行遗留的 running 状态，避免源永久卡在「刷新中」
	if err := sourceSvc.ResetStuckRefresh(); err != nil {
		applog.Warn("reset stuck refresh: %v", err)
	}
	ruleSvc := rule.NewService(db)
	if err := ruleSvc.SeedDefaults(); err != nil {
		applog.Fatalf("seed rules: %v", err)
	}
	publishSvc := publish.NewService(db, sourceSvc, ruleSvc)
		subSvc := subscription.NewService(db, publishSvc, box, cfg.PublicBaseURL)
		apiKeySvc := apikey.NewService(db, box)
		geoSvc := geo.NewService(cfg.GeoDir, geo.URLs{
			GeoIP: cfg.GeoIPURL, GeoSite: cfg.GeoSiteURL, MetaDB: cfg.MetaDBURL, ASN: cfg.ASNURL,
		})
		geoSvc.Load()
		netCheckSvc := netcheck.NewService(db)

	// 启动后异步拉一次全部启用源，再按间隔定时刷新；首次发布由用户在面板完成
	go func() {
		// 略延迟，避免与 HTTP 启动争抢
		time.Sleep(3 * time.Second)
		sourceSvc.RefreshAll()
		ticker := time.NewTicker(cfg.RefreshInterval)
		defer ticker.Stop()
		for range ticker.C {
			sourceSvc.RefreshAll()
		}
	}()

	// 定期清理过期会话，避免 sessions 表无限堆积
	go func() {
		if n, err := authSvc.PurgeExpiredSessions(); err != nil {
			applog.Warn("purge expired sessions: %v", err)
		} else if n > 0 {
			applog.Info("purged %d expired sessions", n)
		}
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if _, err := authSvc.PurgeExpiredSessions(); err != nil {
				applog.Warn("purge expired sessions: %v", err)
			}
		}
	}()

		r := server.NewRouter(server.Deps{
			Cfg:      cfg,
			Auth:     auth.NewHandler(authSvc, auditSvc, cfg.SessionTTL, cfg.CookieSecure),
			Source:   source.NewHandler(sourceSvc, auditSvc),
			Rule:     rule.NewHandler(ruleSvc, auditSvc, geoSvc),
			Publish:  publish.NewHandler(publishSvc, auditSvc),
			Sub:      subscription.NewHandler(subSvc, auditSvc),
			APIKey:   apikey.NewHandler(apiKeySvc, auditSvc),
			Geo:      geo.NewHandler(geoSvc, auditSvc),
			NetCheck: netcheck.NewHandler(netCheckSvc, auditSvc),
			Logs:     logs.NewHandler(logs.NewService(cfg.LogDir)),
			Audit:    auditSvc,
			AuthMW:   middleware.AuthRequired(db, apiKeySvc),
			LoginRL:  middleware.RateLimit(cfg.RateLimitLogin),
			SubRL:    middleware.RateLimit(cfg.RateLimitSub),
		})

	applog.Info("submerge %s listening on %s (db=%s log=%s dir=%s retain=%dd)",
		cfg.Version, cfg.HTTPAddr, filepath.Clean(cfg.DBPath),
		cfg.LogOutput, filepath.Clean(cfg.LogDir), cfg.LogRetentionDays)
	if err := r.Run(cfg.HTTPAddr); err != nil {
		applog.Fatalf("server: %v", err)
	}
}
