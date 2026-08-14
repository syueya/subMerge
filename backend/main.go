package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/submerge/submerge/backend/internal/apikey"
	"github.com/submerge/submerge/backend/internal/applog"
	"github.com/submerge/submerge/backend/internal/appupdate"
	"github.com/submerge/submerge/backend/internal/auth"
	"github.com/submerge/submerge/backend/internal/config"
	"github.com/submerge/submerge/backend/internal/crypto"
	"github.com/submerge/submerge/backend/internal/database"
	"github.com/submerge/submerge/backend/internal/geo"
	"github.com/submerge/submerge/backend/internal/ipgeo"
	"github.com/submerge/submerge/backend/internal/logs"
	"github.com/submerge/submerge/backend/internal/middleware"
	"github.com/submerge/submerge/backend/internal/netcheck"
	"github.com/submerge/submerge/backend/internal/publish"
	"github.com/submerge/submerge/backend/internal/rule"
	"github.com/submerge/submerge/backend/internal/server"
	"github.com/submerge/submerge/backend/internal/source"
	"github.com/submerge/submerge/backend/internal/subscription"
	"github.com/submerge/submerge/backend/internal/systemsettings"
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
	applog.SetDebugEnabled(cfg.DebugLogging)
	defer applog.Close()

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		applog.Fatalf("database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		applog.Fatalf("database handle: %v", err)
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

	authSvc := auth.NewService(db, cfg.SessionTTL)
	// 管理员仅通过网页首次注册创建，不在环境变量里配置

	sourceSvc := source.NewServiceWithUA(db, box, cfg.SourceFetchTimeout, cfg.SourceMaxBytes, cfg.SourceFetchUA)
	ipGeoClient, err := ipgeo.NewClient(cfg.IPGeoURL, cfg.IPGeoTimeout)
	if err != nil {
		applog.Fatalf("IP geo client: %v", err)
	}
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
	authHandler := auth.NewHandler(authSvc, cfg.SessionTTL, cfg.CookieSecure)
	apiKeySvc := apikey.NewService(db, box)
	geoSvc := geo.NewService(cfg.GeoDir, geo.URLs{
		GeoIP: cfg.GeoIPURL, GeoSite: cfg.GeoSiteURL, MetaDB: cfg.MetaDBURL, ASN: cfg.ASNURL,
	})
	geoSvc.SetIPGeoClient(ipGeoClient)
	geoSvc.Load()
	var sysProxyURL string
	netCheckSvc := netcheck.NewService(db, func() string { return sysProxyURL })
	refreshScheduler := systemsettings.NewRefreshScheduler(cfg.RefreshInterval, func() { sourceSvc.RefreshAll() })
	var closeSchedulerOnce sync.Once
	closeScheduler := func() { closeSchedulerOnce.Do(refreshScheduler.Close) }
	defer closeScheduler()
	applySettings := func(settings systemsettings.Settings) error {
		if err := sourceSvc.SetRuntimeOptions(settings.SourceFetchTimeout, settings.SourceMaxBytes, settings.SourceFetchUA); err != nil {
			return err
		}
		proxyURL := settings.ProxyURL
		if !settings.ProxyEnabled {
			proxyURL = ""
		}
		sysProxyURL = proxyURL
		if err := sourceSvc.SetProxy(proxyURL); err != nil {
			return err
		}
		if err := geoSvc.SetURLs(geo.URLs{GeoIP: settings.GeoIPURL, GeoSite: settings.GeoSiteURL, MetaDB: settings.GeoDBURL, ASN: settings.GeoASNURL}); err != nil {
			return err
		}
		if err := geoSvc.SetProxy(proxyURL); err != nil {
			return err
		}
		if err := ipGeoClient.SetConfig(settings.IPGeoURL, settings.IPGeoTimeout); err != nil {
			return err
		}
		if err := subSvc.SetBaseURL(settings.PublicBaseURL); err != nil {
			return err
		}
		authHandler.SetCookieSecure(settings.CookieSecure)
		if err := applog.Setup(settings.LogOutput, cfg.LogDir, settings.LogRetentionDays); err != nil {
			return err
		}
		applog.SetDebugEnabled(settings.DebugLogging)
		cfg.SourceFetchTimeout = settings.SourceFetchTimeout
		cfg.SourceMaxBytes = settings.SourceMaxBytes
		cfg.SourceFetchUA = settings.SourceFetchUA
		cfg.PublicBaseURL = settings.PublicBaseURL
		cfg.CookieSecure = settings.CookieSecure
		cfg.RefreshInterval = settings.RefreshInterval
		cfg.GeoIPURL = settings.GeoIPURL
		cfg.GeoSiteURL = settings.GeoSiteURL
		cfg.MetaDBURL = settings.GeoDBURL
		cfg.ASNURL = settings.GeoASNURL
		cfg.IPGeoURL = settings.IPGeoURL
		cfg.IPGeoTimeout = settings.IPGeoTimeout
		cfg.LogOutput = settings.LogOutput
		cfg.LogRetentionDays = settings.LogRetentionDays
		cfg.DebugLogging = settings.DebugLogging
		if refreshScheduler != nil {
			refreshScheduler.SetInterval(settings.RefreshInterval)
		}
		return nil
	}
	settingsManager, err := systemsettings.NewManager(db, box, cfg.Env == "production", applySettings)
	if err != nil {
		applog.Fatalf("system settings: %v", err)
	}
	startupSettings := settingsManager.View().Settings
	cfg.TrustedProxies = systemsettings.TrustedProxyList(startupSettings.TrustedProxies)

	// Geo：任一必需文件不可用时后台自动拉取一次（Docker 空 volume 首次启动）；失败只记日志
	go func() {
		if !geoSvc.NeedsBootstrap() {
			return
		}
		applog.Info("geo: required data missing under %s, bootstrapping from remote", filepath.Clean(cfg.GeoDir))
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		result := geoSvc.Update(ctx)
		ok, fail := 0, 0
		for _, item := range result.Items {
			if item.Updated {
				ok++
				continue
			}
			fail++
			applog.Warn("geo bootstrap %s: %s", item.Name, item.Error)
		}
		if fail == 0 {
			applog.Info("geo bootstrap completed (%d resources)", ok)
		} else {
			applog.Warn("geo bootstrap finished with failures (ok=%d fail=%d); use /geo「更新数据」or restart", ok, fail)
		}
	}()

	// 启动后异步拉一次全部启用源，再按间隔定时刷新；首次发布由用户在面板完成
	go func() {
		// 略延迟，避免与 HTTP 启动争抢
		time.Sleep(3 * time.Second)
		sourceSvc.RefreshAll()
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

	shutdownRequests := make(chan appupdate.ShutdownRequest, 1)
	updateService := appupdate.NewService(cfg.Version, "", cfg.DBPath, func(request appupdate.ShutdownRequest) {
		shutdownRequests <- request
	})

	r := server.NewRouter(server.Deps{
		Cfg:            cfg,
		Auth:           authHandler,
		Source:         source.NewHandler(sourceSvc),
		Rule:           rule.NewHandler(ruleSvc, geoSvc),
		Publish:        publish.NewHandler(publishSvc),
		Sub:            subscription.NewHandler(subSvc),
		APIKey:         apikey.NewHandler(apiKeySvc),
		Geo:            geo.NewHandler(geoSvc),
		NetCheck:       netcheck.NewHandler(netCheckSvc),
		SystemSettings: systemsettings.NewHandler(settingsManager),
		Logs:           logs.NewHandler(logs.NewService(cfg.LogDir)),
		Update:         appupdate.NewHandler(updateService),
		AuthMW:         middleware.AuthRequired(db, apiKeySvc),
		LoginRL:        middleware.RateLimit(cfg.RateLimitLogin),
		SubRL:          middleware.RateLimit(cfg.RateLimitSub),
	})

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}
	runContext, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- httpServer.ListenAndServe() }()

	// 正式签名构建后台检查；开发构建因没有内嵌公钥会直接跳过。
	if updateService.Status().Enabled {
		go backgroundUpdateChecks(runContext, updateService)
	}

	applog.Info("submerge %s listening on %s (db=%s log=%s dir=%s retain=%dd)",
		cfg.Version, cfg.HTTPAddr, filepath.Clean(cfg.DBPath),
		cfg.LogOutput, filepath.Clean(cfg.LogDir), cfg.LogRetentionDays)

	var updateRequest *appupdate.ShutdownRequest
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			applog.Fatalf("server: %v", err)
		}
		return
	case <-runContext.Done():
		applog.Info("shutdown signal received")
	case request := <-shutdownRequests:
		updateRequest = &request
		applog.Info("update %s requested; stopping HTTP before replacing files", request.Action)
	}

	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelShutdown()
	if err := httpServer.Shutdown(shutdownContext); err != nil {
		applog.Warn("HTTP shutdown: %v", err)
	}
	closeScheduler()
	if err := sqlDB.Close(); err != nil {
		applog.Warn("database close: %v", err)
	}
	if updateRequest != nil {
		installContext, cancelInstall := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancelInstall()
		if err := updateRequest.Execute(installContext); err != nil {
			applog.Fatalf("update %s failed after shutdown: %v", updateRequest.Action, err)
		}
		applog.Info("update %s completed; exiting for supervisor restart", updateRequest.Action)
	}
}

func backgroundUpdateChecks(ctx context.Context, service *appupdate.Service) {
	check := func() {
		checkContext, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		status, err := service.Check(checkContext)
		if err != nil {
			applog.Warn("background update check: %v", err)
			return
		}
		if status.Available {
			applog.Info("update %s is available (current %s)", status.LatestVersion, status.CurrentVersion)
		}
	}
	initial := time.NewTimer(15 * time.Second)
	defer initial.Stop()
	select {
	case <-ctx.Done():
		return
	case <-initial.C:
		check()
	}
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			check()
		}
	}
}
