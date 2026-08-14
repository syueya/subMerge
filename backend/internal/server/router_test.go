package server

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/submerge/submerge/backend/internal/appupdate"
	"github.com/submerge/submerge/backend/internal/config"
	"github.com/submerge/submerge/backend/internal/middleware"
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

func TestEmbeddedWebUIStaticAndSPAFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	web := fstest.MapFS{
		"index.html":        {Data: []byte("<main>SubMerge</main>")},
		"assets/app-123.js": {Data: []byte("console.log('ok')")},
	}
	mountWebUI(r, fs.FS(web), true)

	for _, tc := range []struct {
		path       string
		wantStatus int
		wantBody   string
		wantCache  string
	}{
		{path: "/assets/app-123.js", wantStatus: http.StatusOK, wantBody: "console.log", wantCache: "immutable"},
		{path: "/main/sources", wantStatus: http.StatusOK, wantBody: "SubMerge", wantCache: "no-store"},
		{path: "/api/missing", wantStatus: http.StatusNotFound, wantBody: "not_found"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			r.ServeHTTP(recorder, req)
			if recorder.Code != tc.wantStatus || !strings.Contains(recorder.Body.String(), tc.wantBody) {
				t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
			}
			if tc.wantCache != "" && !strings.Contains(recorder.Header().Get("Cache-Control"), tc.wantCache) {
				t.Fatalf("cache-control=%q", recorder.Header().Get("Cache-Control"))
			}
		})
	}
}

func TestUpdateRoutesRequireBrowserSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth := func(c *gin.Context) {
		if c.GetHeader("Authorization") == "Bearer smk_test" {
			c.Set(middleware.ContextAuthType, middleware.AuthTypeAPIKey)
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false})
	}
	noop := func(c *gin.Context) { c.Next() }
	router := NewRouter(Deps{
		Cfg:     &config.Config{},
		Update:  appupdate.NewHandler(nil),
		AuthMW:  auth,
		LoginRL: noop,
		SubRL:   noop,
		WebUI:   fstest.MapFS{"index.html": {Data: []byte("ok")}},
	})

	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/update/status"},
		{http.MethodPost, "/api/update/check"},
		{http.MethodPost, "/api/update/download"},
		{http.MethodPost, "/api/update/install"},
		{http.MethodPost, "/api/update/rollback"},
	} {
		t.Run(tc.path+"/anonymous", func(t *testing.T) {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(tc.method, tc.path, nil))
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
			}
		})
		t.Run(tc.path+"/api-key", func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tc.method, tc.path, nil)
			request.Header.Set("Authorization", "Bearer smk_test")
			router.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusForbidden {
				t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
			}
		})
	}
}
