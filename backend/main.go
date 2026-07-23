package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/submerge/submerge/backend/internal/applog"
	"github.com/submerge/submerge/backend/internal/audit"
	"github.com/submerge/submerge/backend/internal/auth"
	"github.com/submerge/submerge/backend/internal/config"
	"github.com/submerge/submerge/backend/internal/crypto"
	"github.com/submerge/submerge/backend/internal/database"
	"github.com/submerge/submerge/backend/internal/middleware"
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

	box, err := crypto.NewBox(cfg.EncryptionKey)
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
	// 已有节点时自动发布默认 v1，避免订阅接口因「未发布」一直 403
	if ok, err := publishSvc.EnsureDefaultPublished(); err != nil {
		applog.Warn("default publish: %v", err)
	} else if ok {
		applog.Info("default publish: created v1 from current draft")
	}
	subSvc := subscription.NewService(db, publishSvc, box, cfg.PublicBaseURL)

	// 启动后异步拉一次全部启用源，再按间隔定时刷新；首次拉取后若仍无发布版再尝试 v1
	go func() {
		// 略延迟，避免与 HTTP 启动争抢
		time.Sleep(3 * time.Second)
		sourceSvc.RefreshAll()
		if ok, err := publishSvc.EnsureDefaultPublished(); err != nil {
			applog.Warn("default publish after refresh: %v", err)
		} else if ok {
			applog.Info("default publish: created v1 after first source refresh")
		}
		ticker := time.NewTicker(cfg.RefreshInterval)
		defer ticker.Stop()
		for range ticker.C {
			sourceSvc.RefreshAll()
		}
	}()

	r := server.NewRouter(server.Deps{
		Cfg:     cfg,
		Auth:    auth.NewHandler(authSvc, auditSvc, cfg.SessionTTL, cfg.Env == "production"),
		Source:  source.NewHandler(sourceSvc, auditSvc),
		Rule:    rule.NewHandler(ruleSvc, auditSvc),
		Publish: publish.NewHandler(publishSvc, auditSvc),
		Sub:     subscription.NewHandler(subSvc, auditSvc),
		Audit:   auditSvc,
		AuthMW:  middleware.AuthRequired(db),
		LoginRL: middleware.RateLimit(cfg.RateLimitLogin),
		SubRL:   middleware.RateLimit(cfg.RateLimitSub),
	})

	applog.Info("submerge %s listening on %s (db=%s log=%s dir=%s retain=%dd)",
		cfg.Version, cfg.HTTPAddr, filepath.Clean(cfg.DBPath),
		cfg.LogOutput, filepath.Clean(cfg.LogDir), cfg.LogRetentionDays)
	if err := r.Run(cfg.HTTPAddr); err != nil {
		applog.Fatalf("server: %v", err)
	}
}
