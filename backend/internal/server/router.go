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
	"github.com/submerge/submerge/backend/internal/apiresp"
	"github.com/submerge/submerge/backend/internal/applog"
	"github.com/submerge/submerge/backend/internal/audit"
	"github.com/submerge/submerge/backend/internal/auth"
	"github.com/submerge/submerge/backend/internal/config"
	"github.com/submerge/submerge/backend/internal/geo"
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
	Geo      *geo.Handler
	NetCheck *netcheck.Handler
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

	api := r.Group("/api")
	{
		api.GET("/auth/setup-status", d.Auth.SetupStatus)
		api.POST("/auth/bootstrap", d.LoginRL, d.Auth.Bootstrap)
		api.POST("/auth/login", d.LoginRL, d.Auth.Login)

		secured := api.Group("")
		secured.Use(d.AuthMW)
		{
			secured.POST("/auth/logout", d.Auth.Logout)
			secured.GET("/auth/me", d.Auth.Me)
			secured.POST("/auth/password", d.Auth.ChangePassword)
			secured.PUT("/auth/profile", d.Auth.UpdateProfile)

			secured.GET("/sources", d.Source.List)
			secured.POST("/sources", d.Source.Create)
			secured.PUT("/sources/:id", d.Source.Update)
			secured.POST("/sources/batch-delete", d.Source.BatchDelete)
			secured.DELETE("/sources/:id", d.Source.Delete)
			secured.POST("/sources/refresh-all", d.Source.RefreshAll)
			secured.POST("/sources/:id/refresh", d.Source.Refresh)
			secured.GET("/regions", d.Source.ListRegions)
			secured.GET("/proxies", d.Source.ListProxies)
			secured.PUT("/proxies/batch", d.Source.BatchUpdateProxies)
			secured.PUT("/proxies/:id", d.Source.UpdateProxy)

			secured.GET("/geo/status", d.Geo.Status)
			secured.GET("/geo/categories", d.Geo.Categories)
			secured.POST("/geo/query", d.Geo.Query)
			secured.POST("/geo/reverse", d.Geo.Reverse)
			secured.POST("/geo/search", d.Geo.Search)
			secured.POST("/geo/update", d.Geo.Update)

			secured.GET("/net-check/config", d.NetCheck.GetConfig)
			secured.PUT("/net-check/config", d.NetCheck.SaveConfig)
			secured.POST("/net-check/check", d.NetCheck.Check)
			secured.POST("/net-check/reset", d.NetCheck.ResetConfig)

			secured.GET("/rules", d.Rule.ListRules)
			secured.POST("/rules", d.Rule.CreateRule)
			secured.POST("/rules/batch-import", d.Rule.BatchImportRules)
			secured.PUT("/rules/batch-target", d.Rule.BatchUpdateRulesTarget)
			secured.PUT("/rules/batch-enabled", d.Rule.BatchUpdateRulesEnabled)
			secured.PUT("/rules/batch-category", d.Rule.BatchUpdateRulesCategory)
			secured.POST("/rules/batch-delete", d.Rule.BatchDeleteRules)
			secured.PUT("/rules/:id", d.Rule.UpdateRule)
			secured.DELETE("/rules/:id", d.Rule.DeleteRule)
			secured.POST("/rules/reorder", d.Rule.ReorderRules)

			secured.GET("/groups", d.Rule.ListGroups)
			secured.POST("/groups", d.Rule.CreateGroup)
			secured.PUT("/groups/:id", d.Rule.UpdateGroup)
			secured.DELETE("/groups/:id", d.Rule.DeleteGroup)

			secured.GET("/tokens", d.Sub.List)
			secured.POST("/tokens", d.Sub.Create)
			secured.PUT("/tokens/:id", d.Sub.Update)
			secured.POST("/tokens/:id/revoke", d.Sub.Revoke)
			secured.POST("/tokens/:id/regenerate", d.Sub.Regenerate)
			secured.DELETE("/tokens/:id", d.Sub.Delete)

			secured.GET("/releases", d.Publish.List)
			secured.GET("/releases/current", d.Publish.Current)
			secured.GET("/releases/preview", d.Publish.Preview)
			secured.GET("/releases/draft-status", d.Publish.DraftStatus)
			secured.POST("/releases/publish", d.Publish.Publish)
			secured.GET("/releases/:id", d.Publish.Get)
			secured.POST("/releases/:id/rollback", d.Publish.Rollback)

			secured.GET("/audit", func(c *gin.Context) {
				res, err := d.Audit.List(50, 0)
				if err != nil {
					apiresp.Fail(c, http.StatusInternalServerError, "internal", "list audit failed")
					return
				}
				apiresp.OK(c, res)
			})
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
