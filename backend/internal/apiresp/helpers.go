package apiresp

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// RequireID 解析路由参数 :id 为 uint；失败时自动写入 400 响应并返回 false。
func RequireID(c *gin.Context) (uint, bool) {
	id, err := ParseID(c)
	if err != nil {
		Fail(c, http.StatusBadRequest, "bad_request", "invalid id")
		return 0, false
	}
	return id, true
}

// BindJSON 绑定请求 JSON 到 dst；失败时自动写入 400 响应并返回 false。
func BindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
		return false
	}
	return true
}

// Success 返回统一的成功响应 map[string]bool{"success": true}。
func Success(c *gin.Context) {
	OK(c, map[string]bool{"success": true})
}

// FormatRFC3339 将时间格式化为 RFC3339 字符串；nil 时返回空串。
func FormatRFC3339(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
