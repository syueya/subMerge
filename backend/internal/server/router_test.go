package server

import (
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestSafeLogFormatterRedactsSubscriptionToken(t *testing.T) {
	token := "secret-token-value"
	line := safeLogFormatter()(gin.LogFormatterParams{
		TimeStamp:  time.Now(),
		StatusCode: 200,
		Latency:    time.Millisecond,
		ClientIP:   "127.0.0.1",
		Method:     "GET",
		Path:       "/subscribe/" + token,
	})
	if strings.Contains(line, token) {
		t.Fatalf("log contains subscription token: %q", line)
	}
	if !strings.Contains(line, "/subscribe/:token") {
		t.Fatalf("log does not contain redacted path: %q", line)
	}
}

func TestSafeLogFormatterPreservesNonSubscriptionPath(t *testing.T) {
	line := safeLogFormatter()(gin.LogFormatterParams{Method: "GET", Path: "/api/health"})
	if !strings.Contains(line, "/api/health") {
		t.Fatalf("unexpected log line: %q", line)
	}
}
