package subscription

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/crypto"
	"github.com/submerge/submerge/backend/internal/database"
	"github.com/submerge/submerge/backend/internal/middleware"
	"github.com/submerge/submerge/backend/internal/publish"
	"github.com/submerge/submerge/backend/internal/rule"
	"github.com/submerge/submerge/backend/internal/source"
)

func TestTokenListMasksForAPIKeyAndAllowsSession(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "submerge.db"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	box, err := crypto.NewBox("test-encryption-key-32-bytes-long!!")
	if err != nil {
		t.Fatal(err)
	}
	publishSvc := publish.NewService(db, source.NewService(db, nil, 0, 0), rule.NewService(db))
	svc := NewService(db, publishSvc, box, "http://example.test")
	token, err := svc.Create("test", nil, common.TokenGroupModeAuto, nil)
	if err != nil {
		t.Fatal(err)
	}
	plain := token.Token

	masked, err := svc.List(false)
	if err != nil {
		t.Fatal(err)
	}
	if got := masked.Items[0]; got.Token != "" || got.SubscribeURL != "" || got.TokenMasked == plain {
		t.Fatalf("API key list leaked token: %+v", got)
	}
	full, err := svc.List(true)
	if err != nil {
		t.Fatal(err)
	}
	if got := full.Items[0]; got.Token != plain || got.SubscribeURL == "" {
		t.Fatalf("session list lost token: %+v", got)
	}
}

func TestTokenMutationMissingIDReturnsNotFound(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "submerge.db"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	box, err := crypto.NewBox("test-encryption-key-32-bytes-long!!")
	if err != nil {
		t.Fatal(err)
	}
	publishSvc := publish.NewService(db, source.NewService(db, nil, 0, 0), rule.NewService(db))
	h := NewHandler(NewService(db, publishSvc, box, "http://example.test"))
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		method string
		handle gin.HandlerFunc
		body   string
	}{
		{http.MethodPut, h.Update, `{}`},
		{http.MethodPost, h.Revoke, ``},
		{http.MethodPost, h.Regenerate, ``},
	} {
		req := httptest.NewRequest(tc.method, "/api/tokens/999", strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Request = req
		ctx.Params = gin.Params{{Key: "id", Value: "999"}}
		ctx.Set(middleware.ContextAuthType, middleware.AuthTypeSession)
		tc.handle(ctx)
		if w.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d, want 404: %s", tc.method, w.Code, w.Body.String())
		}
		var response common.ApiResponse[any]
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if response.Error == nil || response.Error.Code != "not_found" {
			t.Fatalf("%s error = %+v", tc.method, response.Error)
		}
	}
}

func TestSubscribeResponseIsNotCacheable(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "submerge.db"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	box, err := crypto.NewBox("test-encryption-key-32-bytes-long!!")
	if err != nil {
		t.Fatal(err)
	}
	publishSvc := publish.NewService(db, source.NewService(db, nil, 0, 0), rule.NewService(db))
	svc := NewService(db, publishSvc, box, "http://example.test")
	h := NewHandler(svc)

	invalid := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(invalid)
	ctx.Params = gin.Params{{Key: "token", Value: "not-a-real-token"}}
	h.Subscribe(ctx)
	if invalid.Code != 403 {
		t.Fatalf("invalid token status = %d", invalid.Code)
	}
	if body := invalid.Body.String(); body != "invalid token" {
		t.Fatalf("invalid token body = %q, want invalid token", body)
	}
	if invalid.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("invalid response cache control = %q", invalid.Header().Get("Cache-Control"))
	}

	token := "subscription-test-token"
	if err := db.Create(&database.ShareToken{
		Name:        "test",
		TokenHash:   crypto.HashToken(token),
		TokenPrefix: token[:4],
		Status:      string(common.TokenStatusActive),
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&database.Release{
		Version:    1,
		Status:     string(common.ReleaseStatusPublished),
		ConfigHash: "hash",
		ConfigYAML: "proxies: []\n",
		CreatedBy:  "test",
	}).Error; err != nil {
		t.Fatal(err)
	}

	valid := httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(valid)
	ctx.Params = gin.Params{{Key: "token", Value: token}}
	h.Subscribe(ctx)
	if valid.Code != 200 {
		t.Fatalf("valid token status = %d", valid.Code)
	}
	if valid.Header().Get("Cache-Control") != "no-store, private" {
		t.Fatalf("valid response cache control = %q", valid.Header().Get("Cache-Control"))
	}
	ct := valid.Header().Get("Content-Type")
	if ct == "" || !(strings.Contains(ct, "yaml") || strings.Contains(ct, "octet-stream")) {
		t.Fatalf("unexpected Content-Type: %q", ct)
	}
	if !strings.Contains(valid.Header().Get("Content-Disposition"), "SubMerge.yaml") {
		t.Fatalf("expected Content-Disposition filename, got %q", valid.Header().Get("Content-Disposition"))
	}
	if valid.Header().Get("Profile-Update-Interval") == "" {
		t.Fatal("missing Profile-Update-Interval")
	}
	if valid.Header().Get("Set-Cookie") != "" {
		t.Fatal("subscription response unexpectedly set a cookie")
	}
}
