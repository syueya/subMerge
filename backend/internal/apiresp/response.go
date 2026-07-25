package apiresp

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	common "github.com/submerge/submerge/backend/common"
)

// ParseID 解析路由参数 :id 为 uint（各 handler 共用）。
func ParseID(c *gin.Context) (uint, error) {
	n, err := strconv.ParseUint(c.Param("id"), 10, 64)
	return uint(n), err
}

func OK[T any](c *gin.Context, data T) {
	c.JSON(http.StatusOK, common.ApiResponse[T]{OK: true, Data: &data})
}

func Fail(c *gin.Context, status int, code, message string) {
	c.JSON(status, common.ApiResponse[any]{
		OK: false,
		Error: &common.ApiError{
			Code:    code,
			Message: message,
		},
	})
}

func FailDetails(c *gin.Context, status int, code, message, details string) {
	c.JSON(status, common.ApiResponse[any]{
		OK: false,
		Error: &common.ApiError{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}
