package source

import (
	"encoding/base64"
	"path/filepath"
	"strings"
	"testing"

	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/crypto"
	"github.com/submerge/submerge/backend/internal/database"
)

func newManualTestService(t *testing.T) *Service {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "manual.db"))
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
	return NewService(db, box, 0, 0)
}

func manualLinks() string {
	vmessJSON := `{"v":"2","ps":"VMess","add":"vmess.example.com","port":"443","id":"11111111-1111-1111-1111-111111111111","aid":"0","net":"ws","tls":"tls"}`
	vmess := "vmess://" + base64.StdEncoding.EncodeToString([]byte(vmessJSON))
	ssUser := base64.StdEncoding.EncodeToString([]byte("aes-256-gcm:secret"))
	return strings.Join([]string{
		"vless://22222222-2222-2222-2222-222222222222@vless.example.com:443?encryption=none&security=tls&type=ws#VLESS",
		vmess,
		"ss://" + ssUser + "@ss.example.com:8388#SS",
		"trojan://password@trojan.example.com:443?sni=trojan.example.com#Trojan",
		"hy2://password@hy.example.com:443#Hysteria2",
	}, "\n")
}

func TestManualSourceImportsSupportedShareLinks(t *testing.T) {
	svc := newManualTestService(t)
	content := manualLinks()
	result, err := svc.CreateManual(common.ManualSourceRequest{
		Name: "自建节点", Region: "UNK", Content: content,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.InputTotal != 5 || result.Parsed != 5 || result.Added != 5 || result.Source.Kind != common.SourceKindManual {
		t.Fatalf("unexpected import: %+v", result)
	}
	if result.Source.URL != "" || result.Source.URLMasked != "" || result.Source.ManualContent != content {
		t.Fatalf("manual source leaked/missed fields: %+v", result.Source)
	}

	var row database.Source
	if err := svc.db.First(&row, result.Source.ID).Error; err != nil {
		t.Fatal(err)
	}
	if row.SnapshotYAML != "" || row.ManualContentEncrypted == "" || strings.Contains(row.ManualContentEncrypted, "vless://") {
		t.Fatalf("manual content persistence is not encrypted/isolated: %+v", row)
	}
	items, err := svc.EnabledProxiesBySourceIDs([]uint{row.ID})
	if err != nil || len(items) != 5 {
		t.Fatalf("manual source was not included in source filter: nodes=%d err=%v", len(items), err)
	}
}

func TestManualSourceKeepsValidLinksAndRejectsAllInvalid(t *testing.T) {
	svc := newManualTestService(t)
	valid := manualLinks()
	result, err := svc.CreateManual(common.ManualSourceRequest{
		Name: "mixed", Region: "UNK", Content: valid + "\nnot-a-link\ntuic://bad.example:443",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.InputTotal != 7 || result.Parsed != 5 || result.ParseDropped["uri_parse"] != 2 {
		t.Fatalf("mixed import statistics = %+v", result)
	}
	if _, err := svc.CreateManual(common.ManualSourceRequest{Name: "invalid", Region: "UNK", Content: "tuic://bad\nnot-a-link"}); err == nil {
		t.Fatal("all-invalid manual import succeeded")
	}
	var count int64
	if err := svc.db.Model(&database.Source{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("all-invalid import wrote a source, count=%d", count)
	}
}

func TestManualSourceUpdateIsAtomicAndPreservesEnabled(t *testing.T) {
	svc := newManualTestService(t)
	content := manualLinks()
	created, err := svc.CreateManual(common.ManualSourceRequest{Name: "batch", Region: "UNK", Content: content})
	if err != nil {
		t.Fatal(err)
	}
	var proxies []database.Proxy
	if err := svc.db.Where("source_id = ?", created.Source.ID).Find(&proxies).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.db.Model(&database.Proxy{}).Where("id = ?", proxies[0].ID).Update("enabled", false).Error; err != nil {
		t.Fatal(err)
	}
	updated, err := svc.UpdateManual(created.Source.ID, common.ManualSourceRequest{Name: "batch", Region: "UNK", Content: content})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Kept != 5 || updated.Added != 0 || updated.Removed != 0 {
		t.Fatalf("same-content update diff = %+v", updated)
	}
	var retained database.Proxy
	if err := svc.db.Where("source_id = ? AND fingerprint = ?", created.Source.ID, proxies[0].Fingerprint).First(&retained).Error; err != nil {
		t.Fatal(err)
	}
	if retained.Enabled {
		t.Fatal("matching fingerprint did not retain disabled state")
	}

	before := updated.Source.ManualContent
	if _, err := svc.UpdateManual(created.Source.ID, common.ManualSourceRequest{Name: "batch", Region: "UNK", Content: "tuic://unsupported"}); err == nil {
		t.Fatal("all-invalid update succeeded")
	}
	view, err := svc.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Items) != 1 || view.Items[0].ManualContent != before || view.Items[0].ProxyCount != 5 {
		t.Fatalf("failed update changed prior snapshot: %+v", view.Items)
	}
}

func TestManualSourceCannotRefreshAndIsExcludedFromRefreshAll(t *testing.T) {
	svc := newManualTestService(t)
	created, err := svc.CreateManual(common.ManualSourceRequest{Name: "manual", Region: "UNK", Content: manualLinks()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Refresh(created.Source.ID); err == nil || !strings.Contains(err.Error(), "cannot be refreshed") {
		t.Fatalf("manual refresh error = %v", err)
	}
	var row database.Source
	if err := svc.db.First(&row, created.Source.ID).Error; err != nil {
		t.Fatal(err)
	}
	if row.RefreshStatus != string(common.RefreshStatusSuccess) {
		t.Fatalf("manual refresh changed status to %q", row.RefreshStatus)
	}
	all := svc.RefreshAll()
	if all.Total != 0 || len(all.Results) != 0 {
		t.Fatalf("refresh-all included manual source: %+v", all)
	}
}

func TestDatabaseMigratesEmptySourceKindToRemote(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "migrate.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	row := database.Source{Name: "legacy", Region: "UNK", URLEncrypted: "cipher", Kind: "remote", Enabled: true, RegionMode: "auto"}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&database.Source{}).Where("id = ?", row.ID).Update("kind", "").Error; err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()

	reopened, err := database.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { sql, _ := reopened.DB(); _ = sql.Close() }()
	if err := reopened.First(&row, row.ID).Error; err != nil {
		t.Fatal(err)
	}
	if row.Kind != string(common.SourceKindRemote) {
		t.Fatalf("legacy kind = %q", row.Kind)
	}
}
