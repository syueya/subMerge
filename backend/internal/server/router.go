package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/apikey"
	"github.com/submerge/submerge/backend/internal/apiresp"
	"github.com/submerge/submerge/backend/internal/applog"
	"github.com/submerge/submerge/backend/internal/audit"
	"github.com/submerge/submerge/backend/internal/auth"
	"github.com/submerge/submerge/backend/internal/config"
	"github.com/submerge/submerge/backend/internal/geo"
	"github.com/submerge/submerge/backend/internal/logs"
	"github.com/submerge/submerge/backend/internal/middleware"
	"github.com/submerge/submerge/backend/internal/netcheck"
	"github.com/submerge/submerge/backend/internal/publish"
	"github.com/submerge/submerge/backend/internal/rule"
	"github.com/submerge/submerge/backend/internal/source"
	"github.com/submerge/submerge/backend/internal/subscription"
)

// Deps 路由依赖
type Deps struct {
	Cfg      *config.Config
	Auth     *auth.Handler
	Source   *source.Handler
	Rule     *rule.Handler
	Publish  *publish.Handler
	Sub      *subscription.Handler
	APIKey   *apikey.Handler
	Geo      *geo.Handler
	NetCheck *netcheck.Handler
	Logs     *logs.Handler
	Audit    *audit.Service
	AuthMW   gin.HandlerFunc
	LoginRL  gin.HandlerFunc
	SubRL    gin.HandlerFunc
}

func safeLogFormatter() gin.LogFormatter {
	return func(param gin.LogFormatterParams) string {
		path := param.Path
		if strings.HasPrefix(path, "/subscribe/") {
			path = "/subscribe/:token"
		}
		return fmt.Sprintf("[GIN] %v | %3d | %13v | %15s | %-7s %s\n",
			param.TimeStamp.Format(time.RFC3339),
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			path,
		)
	}
}

// NewRouter 组装 Gin 路由
func NewRouter(d Deps) *gin.Engine {
	if d.Cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	// 默认不信任任何代理：c.ClientIP() 取 socket 源地址，防止 X-Forwarded-For 伪造。
	// 部署在反向代理/Docker 后时，用 TRUSTED_PROXIES 配置可信代理 CIDR，
	// 这样限流与审计才能拿到真实客户端 IP（否则所有请求都记成代理 IP）。
	if len(d.Cfg.TrustedProxies) > 0 {
		if err := r.SetTrustedProxies(d.Cfg.TrustedProxies); err != nil {
			applog.Warn("invalid TRUSTED_PROXIES, falling back to trusting none: %v", err)
			_ = r.SetTrustedProxies(nil)
		}
	} else {
		_ = r.SetTrustedProxies(nil)
	}
	r.Use(gin.Recovery())
	r.Use(gin.LoggerWithFormatter(safeLogFormatter()))
	r.Use(middleware.CORS(d.Cfg.CORSOrigins))

	r.GET("/api/health", func(c *gin.Context) {
		apiresp.OK(c, common.HealthResponse{
			Status:  "ok",
			Version: d.Cfg.Version,
			Time:    time.Now().UTC().Format(time.RFC3339),
		})
	})

	// 公开订阅
	r.GET("/subscribe/:token", d.SubRL, d.Sub.Subscribe)

	scopeRead := middleware.RequireScope(common.APIKeyScopeRead)
	scopeWrite := middleware.RequireScope(common.APIKeyScopeWrite)
	scopePublish := middleware.RequireScope(common.APIKeyScopePublish)
	sessionOnly := middleware.RequireSession()

	api := r.Group("/api")
	{
		api.GET("/auth/setup-status", d.Auth.SetupStatus)
		api.POST("/auth/bootstrap", d.LoginRL, d.Auth.Bootstrap)
		api.POST("/auth/login", d.LoginRL, d.Auth.Login)

		secured := api.Group("")
		secured.Use(d.AuthMW)
		{
			// 账户相关：仅 Session
			secured.POST("/auth/logout", sessionOnly, d.Auth.Logout)
			secured.GET("/auth/me", sessionOnly, d.Auth.Me)
			secured.POST("/auth/password", sessionOnly, d.Auth.ChangePassword)
			secured.PUT("/auth/profile", sessionOnly, d.Auth.UpdateProfile)

			// API 密钥管理：仅 Session（agent 不能自管 key）
			if d.APIKey != nil {
				secured.GET("/apikeys", sessionOnly, d.APIKey.List)
				secured.POST("/apikeys", sessionOnly, d.APIKey.Create)
				secured.PUT("/apikeys/:id", sessionOnly, d.APIKey.Update)
				secured.GET("/apikeys/:id/secret", sessionOnly, d.APIKey.Secret)
				secured.POST("/apikeys/:id/revoke", sessionOnly, d.APIKey.Revoke)
				secured.POST("/apikeys/:id/regenerate", sessionOnly, d.APIKey.Regenerate)
				secured.DELETE("/apikeys/:id", sessionOnly, d.APIKey.Delete)
			}

			secured.GET("/sources", scopeRead, d.Source.List)
			secured.POST("/sources", scopeWrite, d.Source.Create)
			secured.PUT("/sources/:id", scopeWrite, d.Source.Update)
			secured.POST("/sources/batch-delete", scopeWrite, d.Source.BatchDelete)
			secured.DELETE("/sources/:id", scopeWrite, d.Source.Delete)
			secured.POST("/sources/refresh-all", scopeWrite, d.Source.RefreshAll)
			secured.POST("/sources/:id/refresh", scopeWrite, d.Source.Refresh)
			secured.GET("/regions", scopeRead, d.Source.ListRegions)
			secured.GET("/proxies", scopeRead, d.Source.ListProxies)
			secured.PUT("/proxies/batch", scopeWrite, d.Source.BatchUpdateProxies)
			secured.PUT("/proxies/:id", scopeWrite, d.Source.UpdateProxy)

			secured.GET("/geo/status", scopeRead, d.Geo.Status)
			secured.GET("/geo/categories", scopeRead, d.Geo.Categories)
			secured.POST("/geo/query", scopeRead, d.Geo.Query)
			secured.POST("/geo/ip-geo", scopeRead, d.Geo.IPGeo)
			secured.POST("/geo/reverse", scopeRead, d.Geo.Reverse)
			secured.POST("/geo/search", scopeRead, d.Geo.Search)
			secured.POST("/geo/update", scopeWrite, d.Geo.Update)

			secured.GET("/net-check/config", scopeRead, d.NetCheck.GetConfig)
			secured.PUT("/net-check/config", scopeWrite, d.NetCheck.SaveConfig)
			secured.POST("/net-check/check", scopeWrite, d.NetCheck.Check)
			secured.POST("/net-check/reset", scopeWrite, d.NetCheck.ResetConfig)

			secured.GET("/rules", scopeRead, d.Rule.ListRules)
			secured.POST("/rules", scopeWrite, d.Rule.CreateRule)
			secured.POST("/rules/match", scopeRead, d.Rule.MatchRules)
			secured.POST("/rules/batch-import", scopeWrite, d.Rule.BatchImportRules)
			secured.PUT("/rules/batch-target", scopeWrite, d.Rule.BatchUpdateRulesTarget)
			secured.PUT("/rules/batch-enabled", scopeWrite, d.Rule.BatchUpdateRulesEnabled)
			secured.PUT("/rules/batch-category", scopeWrite, d.Rule.BatchUpdateRulesCategory)
			secured.POST("/rules/batch-delete", scopeWrite, d.Rule.BatchDeleteRules)
			secured.PUT("/rules/:id", scopeWrite, d.Rule.UpdateRule)
			secured.DELETE("/rules/:id", scopeWrite, d.Rule.DeleteRule)
			secured.POST("/rules/reorder", scopeWrite, d.Rule.ReorderRules)

			secured.GET("/groups", scopeRead, d.Rule.ListGroups)
			secured.POST("/groups", scopeWrite, d.Rule.CreateGroup)
			secured.PUT("/groups/:id", scopeWrite, d.Rule.UpdateGroup)
			secured.DELETE("/groups/:id", scopeWrite, d.Rule.DeleteGroup)

			secured.GET("/tokens", scopeRead, d.Sub.List)
			secured.POST("/tokens", scopeWrite, d.Sub.Create)
			secured.PUT("/tokens/:id", scopeWrite, d.Sub.Update)
			secured.POST("/tokens/:id/revoke", scopeWrite, d.Sub.Revoke)
			secured.POST("/tokens/:id/regenerate", scopeWrite, d.Sub.Regenerate)
			secured.DELETE("/tokens/:id", scopeWrite, d.Sub.Delete)

			secured.GET("/releases", scopeRead, d.Publish.List)
			secured.GET("/releases/current", scopeRead, d.Publish.Current)
			secured.GET("/releases/preview", scopeRead, d.Publish.Preview)
			secured.GET("/releases/draft-status", scopeRead, d.Publish.DraftStatus)
			secured.POST("/releases/publish", scopePublish, d.Publish.Publish)
			secured.GET("/releases/:id", scopeRead, d.Publish.Get)
			secured.POST("/releases/:id/rollback", scopePublish, d.Publish.Rollback)

			secured.GET("/audit", scopeRead, func(c *gin.Context) {
				res, err := d.Audit.List(50, 0)
				if err != nil {
					apiresp.Fail(c, http.StatusInternalServerError, "internal", "list audit failed")
					return
				}
				apiresp.OK(c, res)
			})

			if d.Logs != nil {
				secured.GET("/logs", scopeRead, d.Logs.List)
				secured.GET("/logs/details", scopeRead, d.Logs.Details)
			}
		}
	}

	// 托管前端静态资源（Angular browser 输出）
	staticDir := d.Cfg.StaticDir
	if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
		r.StaticFS("/assets", http.Dir(filepath.Join(staticDir, "assets")))
		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/subscribe") {
				apiresp.Fail(c, http.StatusNotFound, "not_found", "not found")
				return
			}
			// 优先返回真实静态文件（main-*.js / styles-*.css / favicon 等）
			candidate := filepath.Join(staticDir, filepath.Clean("/"+path))
			if !strings.HasPrefix(candidate, filepath.Clean(staticDir)) {
				apiresp.Fail(c, http.StatusNotFound, "not_found", "not found")
				return
			}
			if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
				c.File(candidate)
				return
			}
			index := filepath.Join(staticDir, "index.html")
			if _, err := os.Stat(index); err == nil {
				c.File(index)
				return
			}
			c.String(http.StatusNotFound, "frontend not built")
		})
	}

	return r
}
