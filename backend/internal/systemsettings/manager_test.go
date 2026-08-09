package systemsettings

import (
	"path/filepath"
	"testing"

	"github.com/submerge/submerge/backend/internal/config"
	"github.com/submerge/submerge/backend/internal/crypto"
	"github.com/submerge/submerge/backend/internal/database"
)

func TestManagerWebValuesPersistAndResetToDefaults(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "settings.db"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	box, err := crypto.NewBox("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	var applied []Settings
	m, err := NewManager(db, box, false, func(s Settings) error { applied = append(applied, s); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if got := m.View().Settings.RefreshInterval; got != 24 {
		t.Fatalf("default interval = %d", got)
	}
	ua := "clash-verge/v9.9.9"
	proxy := "http://alice:secret@proxy.test:7890"
	if _, err := m.Save(UpdateRequest{SourceFetchUA: &ua, RefreshInterval: intPtr(6), ProxyEnabled: boolPtr(true), ProxyURL: &proxy}); err != nil {
		t.Fatal(err)
	}
	view := m.View()
	if view.Settings.SourceFetchUA != ua || view.Settings.RefreshInterval != 6 || view.Settings.ProxyMaskedURL != "http://alice:***@proxy.test:7890" {
		t.Fatalf("saved view = %+v", view)
	}
	m2, err := NewManager(db, box, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := m2.View().Settings; got.SourceFetchUA != ua || got.RefreshInterval != 6 || !got.ProxyConfigured {
		t.Fatalf("reload view = %+v", got)
	}
	if _, err := m2.Reset(); err != nil {
		t.Fatal(err)
	}
	view = m2.View()
	if view.Settings.SourceFetchUA != Defaults().SourceFetchUA || view.Settings.RefreshInterval != 24 || view.Settings.ProxyConfigured {
		t.Fatalf("reset view = %+v", view)
	}
	if len(applied) < 2 {
		t.Fatalf("apply calls = %d", len(applied))
	}
}

func TestManagerApplyFailureDoesNotPersist(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "settings.db"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	box, err := crypto.NewBox("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewManager(db, box, false, func(s Settings) error {
		if s.SourceMaxBytes == 123 {
			return errApplyFailed
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	value := int64(123)
	if _, err := m.Save(UpdateRequest{SourceMaxBytes: &value}); err == nil {
		t.Fatal("expected apply failure")
	}
	m2, err := NewManager(db, box, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := m2.View().Settings.SourceMaxBytes; got == 123 {
		t.Fatal("failed value was persisted")
	}
}

func TestTimeoutValuesUseSecondsAndLoadLegacyDurations(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "settings.db"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	box, err := crypto.NewBox("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&database.SystemSetting{Key: KeySourceFetchTimeout, Value: "45s"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&database.SystemSetting{Key: KeyIPGeoTimeout, Value: "7s"}).Error; err != nil {
		t.Fatal(err)
	}
	m, err := NewManager(db, box, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	view := m.View().Settings
	if view.SourceFetchTimeout != 45 || view.IPGeoTimeout != 7 {
		t.Fatalf("legacy timeout view = %+v", view)
	}
	fetch, geo := 90, 12
	if _, err := m.Save(UpdateRequest{SourceFetchTimeout: &fetch, IPGeoTimeout: &geo}); err != nil {
		t.Fatal(err)
	}
	view = m.View().Settings
	if view.SourceFetchTimeout != fetch || view.IPGeoTimeout != geo {
		t.Fatalf("seconds timeout view = %+v", view)
	}
	var rows []database.SystemSetting
	if err := db.Where("key IN ?", []string{KeySourceFetchTimeout, KeyIPGeoTimeout}).Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.Key == KeySourceFetchTimeout && row.Value != "1m30s" {
			t.Fatalf("source timeout storage = %q", row.Value)
		}
		if row.Key == KeyIPGeoTimeout && row.Value != "12s" {
			t.Fatalf("IP geo timeout storage = %q", row.Value)
		}
	}
}

func TestPublicBaseURLValidation(t *testing.T) {
	for _, raw := range []string{"http://localhost:8080", "https://example.com:8443/path", "http://[::1]:8080"} {
		if err := config.ValidatePublicBaseURL(raw); err != nil {
			t.Errorf("valid URL %q: %v", raw, err)
		}
	}
	for _, raw := range []string{"example.com", "https://example.com:0", "https://example.com:65536", "https://example.com/?token=x", "https://example.com/#fragment"} {
		if err := config.ValidatePublicBaseURL(raw); err == nil {
			t.Errorf("invalid URL accepted: %q", raw)
		}
	}
}

func TestRestartRequiredOnlyWhenTrustedProxiesChanged(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "settings.db"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	box, err := crypto.NewBox("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewManager(db, box, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	// 只改代理 URL，trustedProxies 提交相同的空值 → 不应要求重启
	proxy := "http://proxy.test:7890"
	empty := ""
	view, err := m.Save(UpdateRequest{ProxyURL: &proxy, TrustedProxies: &empty})
	if err != nil {
		t.Fatal(err)
	}
	if view.RestartRequired {
		t.Fatalf("proxy-only save reported restartRequired=true")
	}
	// 真的改了 trustedProxies → 应要求重启
	tp := "127.0.0.1"
	view, err = m.Save(UpdateRequest{TrustedProxies: &tp})
	if err != nil {
		t.Fatal(err)
	}
	if !view.RestartRequired {
		t.Fatalf("trusted-proxy change did not report restartRequired=true")
	}
	// 再次提交相同值，不再要求重启
	view, err = m.Save(UpdateRequest{TrustedProxies: &tp})
	if err != nil {
		t.Fatal(err)
	}
	if view.RestartRequired {
		t.Fatalf("unchanged trusted-proxy still reported restartRequired=true")
	}
}

var errApplyFailed = applyError("apply failed")

type applyError string

func (e applyError) Error() string { return string(e) }
func intPtr(v int) *int            { return &v }
func boolPtr(v bool) *bool         { return &v }
