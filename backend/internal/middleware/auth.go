package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/submerge/submerge/backend/internal/crypto"
	"github.com/submerge/submerge/backend/internal/database"
	"github.com/submerge/submerge/backend/internal/apiresp"
	"gorm.io/gorm"
)

const (
	ContextAdminID   = "adminID"
	ContextUsername  = "username"
	HeaderAuth       = "Authorization"
)

// AuthRequired 校验会话 token
func AuthRequired(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader(HeaderAuth)
		if raw == "" {
			// 也支持 cookie
			if cookie, err := c.Cookie("submerge_session"); err == nil {
				raw = "Bearer " + cookie
			}
		}
		if !strings.HasPrefix(raw, "Bearer ") {
			apiresp.Fail(c, http.StatusUnauthorized, "unauthorized", "missing session token")
			c.Abort()
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(raw, "Bearer "))
		if token == "" {
			apiresp.Fail(c, http.StatusUnauthorized, "unauthorized", "missing session token")
			c.Abort()
			return
		}

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
		c.Next()
	}
}

// GetUsername 从上下文取用户名
func GetUsername(c *gin.Context) string {
	v, _ := c.Get(ContextUsername)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// GetAdminID 从上下文取管理员 ID
func GetAdminID(c *gin.Context) uint {
	v, _ := c.Get(ContextAdminID)
	if id, ok := v.(uint); ok {
		return id
	}
	return 0
}
