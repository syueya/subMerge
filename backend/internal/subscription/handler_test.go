package subscription

import (
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/audit"
	"github.com/submerge/submerge/backend/internal/crypto"
	"github.com/submerge/submerge/backend/internal/database"
	"github.com/submerge/submerge/backend/internal/publish"
	"github.com/submerge/submerge/backend/internal/rule"
	"github.com/submerge/submerge/backend/internal/source"
)

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
	h := NewHandler(svc, (*audit.Service)(nil))

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
