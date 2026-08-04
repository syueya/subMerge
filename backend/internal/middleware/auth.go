package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/apiresp"
	"github.com/submerge/submerge/backend/internal/crypto"
	"github.com/submerge/submerge/backend/internal/database"
	"gorm.io/gorm"
)

const (
	ContextAdminID  = "adminID"
	ContextUsername = "username"
	ContextAuthType = "authType"
	ContextAPIKeyID = "apiKeyID"
	ContextScopes   = "scopes"
	HeaderAuth      = "Authorization"

	AuthTypeSession = "session"
	AuthTypeAPIKey  = "apikey"

	apiKeyPrefix = "smk_"
)

// APIKeyAuthenticator 由 apikey.Service 实现，避免 middleware 反向依赖具体包
type APIKeyAuthenticator interface {
	FindActiveByRaw(raw string) (database.APIKey, []common.APIKeyScope, error)
}

// AuthRequired 校验会话或 API Key（Bearer）。
// - Cookie / 非 smk_ Bearer → Session
// - smk_ 前缀 Bearer → API Key（不接受 Cookie 塞 key）
func AuthRequired(db *gorm.DB, apiKeys APIKeyAuthenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawHeader := c.GetHeader(HeaderAuth)
		fromCookie := false
		if rawHeader == "" {
			if cookie, err := c.Cookie("submerge_session"); err == nil && cookie != "" {
				rawHeader = "Bearer " + cookie
				fromCookie = true
			}
		}
		if !strings.HasPrefix(rawHeader, "Bearer ") {
			apiresp.Fail(c, http.StatusUnauthorized, "unauthorized", "missing session token")
			c.Abort()
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(rawHeader, "Bearer "))
		if token == "" {
			apiresp.Fail(c, http.StatusUnauthorized, "unauthorized", "missing session token")
			c.Abort()
			return
		}

		// API Key：仅 Header，且明文带 smk_ 前缀
		if strings.HasPrefix(token, apiKeyPrefix) {
			if fromCookie {
				apiresp.Fail(c, http.StatusUnauthorized, "unauthorized", "api key must use Authorization header")
				c.Abort()
				return
			}
			if apiKeys == nil {
				apiresp.Fail(c, http.StatusUnauthorized, "unauthorized", "api key auth unavailable")
				c.Abort()
				return
			}
			authenticateAPIKey(c, apiKeys, token)
			return
		}

		authenticateSession(c, db, token)
	}
}

func authenticateSession(c *gin.Context, db *gorm.DB, token string) {
	hash := crypto.HashToken(token)
	var sess database.Session
	err := db.Where("token_hash = ? AND expires_at > ?", hash, time.Now()).First(&sess).Error
	if err != nil {
		apiresp.Fail(c, http.StatusUnauthorized, "unauthorized", "invalid or expired session")
		c.Abort()
		return
	}

	var admin database.Admin
	if err := db.First(&admin, sess.AdminID).Error; err != nil {
		apiresp.Fail(c, http.StatusUnauthorized, "unauthorized", "admin not found")
		c.Abort()
		return
	}

	c.Set(ContextAdminID, admin.ID)
	c.Set(ContextUsername, admin.Username)
	c.Set(ContextAuthType, AuthTypeSession)
	c.Set(ContextScopes, []common.APIKeyScope{common.APIKeyScopeAll})
	c.Next()
}

func authenticateAPIKey(c *gin.Context, apiKeys APIKeyAuthenticator, token string) {
	row, scopes, err := apiKeys.FindActiveByRaw(token)
	if err != nil {
		msg := err.Error()
		code := "unauthorized"
		switch {
		case strings.Contains(msg, "revoked"):
			code = "apikey_revoked"
		case strings.Contains(msg, "disabled"):
			code = "apikey_disabled"
		case strings.Contains(msg, "expired"):
			code = "apikey_expired"
		}
		apiresp.Fail(c, http.StatusUnauthorized, code, msg)
		c.Abort()
		return
	}

	c.Set(ContextAdminID, uint(0))
	c.Set(ContextUsername, "apikey:"+row.Name)
	c.Set(ContextAuthType, AuthTypeAPIKey)
	c.Set(ContextAPIKeyID, row.ID)
	c.Set(ContextScopes, scopes)
	c.Next()
}

// RequireSession 仅允许管理员会话（API Key 管理、改密等）
func RequireSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		if GetAuthType(c) != AuthTypeSession {
			apiresp.Fail(c, http.StatusForbidden, "forbidden", "session required")
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireScope 粗粒度 scope 守卫；Session 视为拥有全部
func RequireScope(required common.APIKeyScope) gin.HandlerFunc {
	return func(c *gin.Context) {
		if GetAuthType(c) == AuthTypeSession {
			c.Next()
			return
		}
		scopes := GetScopes(c)
		if common.HasAPIKeyScope(scopes, required) {
			c.Next()
			return
		}
		apiresp.Fail(c, http.StatusForbidden, "forbidden", "insufficient scope: "+string(required))
		c.Abort()
	}
}

// GetUsername 从上下文取用户名（session 为管理员名；apikey 为 apikey:<name>）
func GetUsername(c *gin.Context) string {
	v, _ := c.Get(ContextUsername)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// GetAdminID 从上下文取管理员 ID（API Key 为 0）
func GetAdminID(c *gin.Context) uint {
	v, _ := c.Get(ContextAdminID)
	if id, ok := v.(uint); ok {
		return id
	}
	return 0
}

// GetAuthType session | apikey
func GetAuthType(c *gin.Context) string {
	v, _ := c.Get(ContextAuthType)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// GetAPIKeyID API Key 鉴权时的 id
func GetAPIKeyID(c *gin.Context) uint {
	v, _ := c.Get(ContextAPIKeyID)
	if id, ok := v.(uint); ok {
		return id
	}
	return 0
}

// GetScopes 当前调用方 scopes
func GetScopes(c *gin.Context) []common.APIKeyScope {
	v, _ := c.Get(ContextScopes)
	if s, ok := v.([]common.APIKeyScope); ok {
		return s
	}
	return nil
}
