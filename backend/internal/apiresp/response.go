package apiresp

import (
	"net/http"

	"github.com/gin-gonic/gin"
	common "github.com/submerge/submerge/backend/common"
)

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
