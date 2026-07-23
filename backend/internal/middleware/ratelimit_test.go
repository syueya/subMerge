package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRateLimitRejectsRequestsOverLimit(t *testing.T) {
	handler := RateLimit(1)

	first := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(first)
	ctx.Request = httptest.NewRequest("GET", "/", nil)
	ctx.Request.RemoteAddr = "127.0.0.1:1234"
	handler(ctx)
	if first.Code == 429 {
		t.Fatal("first request was rate limited")
	}

	second := httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(second)
	ctx.Request = httptest.NewRequest("GET", "/", nil)
	ctx.Request.RemoteAddr = "127.0.0.1:1234"
	handler(ctx)
	if second.Code != 429 {
		t.Fatalf("expected second request to be rate limited, got %d", second.Code)
	}
}
