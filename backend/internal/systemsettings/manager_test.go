package systemsettings

import (
	"path/filepath"
	"testing"

	"github.com/submerge/submerge/backend/internal/crypto"
	"github.com/submerge/submerge/backend/internal/database"
)

func TestManagerWebValuesPersistAndResetToDefaults(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "settings.db"))
	if err != nil { t.Fatal(err) }
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	box, err := crypto.NewBox("0123456789abcdef0123456789abcdef")
	if err != nil { t.Fatal(err) }
	var applied []Settings
	m, err := NewManager(db, box, false, func(s Settings) error { applied = append(applied, s); return nil })
	if err != nil { t.Fatal(err) }
	if got := m.View().Settings.RefreshInterval; got != 24 { t.Fatalf("default interval = %d", got) }
	ua := "clash-verge/v9.9.9"
	proxy := "http://alice:secret@proxy.test:7890"
	if _, err := m.Save(UpdateRequest{SourceFetchUA: &ua, RefreshInterval: intPtr(6), ProxyEnabled: boolPtr(true), ProxyURL: &proxy}); err != nil { t.Fatal(err) }
	view := m.View()
	if view.Settings.SourceFetchUA != ua || view.Settings.RefreshInterval != 6 || view.Settings.ProxyMaskedURL != "http://alice:***@proxy.test:7890" { t.Fatalf("saved view = %+v", view) }
	m2, err := NewManager(db, box, false, nil)
	if err != nil { t.Fatal(err) }
	if got := m2.View().Settings; got.SourceFetchUA != ua || got.RefreshInterval != 6 || !got.ProxyConfigured { t.Fatalf("reload view = %+v", got) }
	if _, err := m2.Reset(); err != nil { t.Fatal(err) }
	view = m2.View()
	if view.Settings.SourceFetchUA != Defaults().SourceFetchUA || view.Settings.RefreshInterval != 24 || view.Settings.ProxyConfigured { t.Fatalf("reset view = %+v", view) }
	if len(applied) < 2 { t.Fatalf("apply calls = %d", len(applied)) }
}

func TestManagerApplyFailureDoesNotPersist(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "settings.db"))
	if err != nil { t.Fatal(err) }
	sqlDB, _ := db.DB(); t.Cleanup(func() { _ = sqlDB.Close() })
	box, err := crypto.NewBox("0123456789abcdef0123456789abcdef")
	if err != nil { t.Fatal(err) }
	m, err := NewManager(db, box, false, func(s Settings) error { if s.SourceMaxBytes == 123 { return errApplyFailed }; return nil })
	if err != nil { t.Fatal(err) }
	value := int64(123)
	if _, err := m.Save(UpdateRequest{SourceMaxBytes: &value}); err == nil { t.Fatal("expected apply failure") }
	m2, err := NewManager(db, box, false, nil)
	if err != nil { t.Fatal(err) }
	if got := m2.View().Settings.SourceMaxBytes; got == 123 { t.Fatal("failed value was persisted") }
}

var errApplyFailed = applyError("apply failed")
type applyError string
func (e applyError) Error() string { return string(e) }
func intPtr(v int) *int { return &v }
func boolPtr(v bool) *bool { return &v }
