package outbound

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/submerge/submerge/backend/internal/crypto"
	"github.com/submerge/submerge/backend/internal/database"
)

func TestProxyURLValidationAndMasking(t *testing.T) {
	valid := []string{
		"http://127.0.0.1:7890",
		"https://user:secret@example.test:443",
		"socks5://proxy.internal",
		"socks5h://user:secret@[::1]:1080",
	}
	for _, raw := range valid {
		if err := ValidateProxyURL(raw); err != nil {
			t.Errorf("ValidateProxyURL(%q): %v", raw, err)
		}
	}
	for _, raw := range []string{"ftp://proxy.test:21", "http://proxy.test:0", "http://proxy.test:65536", "http://:7890"} {
		if err := ValidateProxyURL(raw); err == nil {
			t.Errorf("ValidateProxyURL(%q) succeeded, want error", raw)
		}
	}
	masked := MaskURL("https://alice:secret@example.test:443/path")
	if strings.Contains(masked, "secret") || !strings.Contains(masked, "alice:***@") {
		t.Fatalf("masked URL = %q", masked)
	}
}

func TestManagerWebOverrideResetAndCredentialRetention(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "proxy.db"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	box, err := crypto.NewBox("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	var applied []string
	manager, err := NewManager(db, box, "http://env-proxy:8080", func(proxyURL string) error {
		applied = append(applied, proxyURL)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := manager.View(); got.Source != sourceEnv || got.MaskedURL != "http://env-proxy:8080" {
		t.Fatalf("initial view = %+v", got)
	}
	if _, err := manager.Save(UpdateRequest{Enabled: boolPtr(true), URL: "http://alice:secret@web-proxy:7890"}); err != nil {
		t.Fatal(err)
	}
	view := manager.View()
	if view.Source != sourceWeb || !view.HasOverride || view.MaskedURL != "http://alice:***@web-proxy:7890" {
		t.Fatalf("web view = %+v", view)
	}
	if _, err := manager.Save(UpdateRequest{Enabled: boolPtr(true), URL: ""}); err != nil {
		t.Fatal(err)
	}
	if got := manager.View().MaskedURL; got != "http://alice:***@web-proxy:7890" {
		t.Fatalf("empty URL replaced credentials: %q", got)
	}
	if _, err := manager.Reset(); err != nil {
		t.Fatal(err)
	}
	view = manager.View()
	if view.Source != sourceEnv || view.HasOverride || view.MaskedURL != "http://env-proxy:8080" {
		t.Fatalf("reset view = %+v", view)
	}
	if len(applied) != 4 || applied[0] != "http://env-proxy:8080" || applied[3] != "http://env-proxy:8080" {
		t.Fatalf("applied proxy values = %#v", applied)
	}
}

func boolPtr(value bool) *bool { return &value }
