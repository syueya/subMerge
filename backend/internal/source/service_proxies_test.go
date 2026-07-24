package source

import (
	"path/filepath"
	"testing"

	"github.com/submerge/submerge/backend/internal/crypto"
	"github.com/submerge/submerge/backend/internal/database"
)

func TestEnabledProxiesBySourceIDsInjectsSourceMeta(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })

	srcA := database.Source{Name: "良心云", Region: "US", URLEncrypted: "x", Enabled: true, RegionMode: "auto"}
	srcB := database.Source{Name: "机场B", Region: "JP", URLEncrypted: "y", Enabled: true, RegionMode: "auto"}
	if err := db.Create(&srcA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&srcB).Error; err != nil {
		t.Fatal(err)
	}
	proxies := []database.Proxy{
		{SourceID: srcA.ID, Name: "US-a-良心云", Region: "US", Type: "ss", Server: "1.1.1.1", Port: 443, Enabled: true,
			RawJSON: `{"name":"US-a-良心云","type":"ss","server":"1.1.1.1","port":443}`},
		{SourceID: srcB.ID, Name: "JP-b-机场B", Region: "JP", Type: "ss", Server: "2.2.2.2", Port: 443, Enabled: true,
			RawJSON: `{"name":"JP-b-机场B","type":"ss","server":"2.2.2.2","port":443}`},
		// 禁用节点不应出现（GORM Create 会忽略 bool 零值，故先建再改）
		{SourceID: srcA.ID, Name: "US-off", Region: "US", Type: "ss", Server: "3.3.3.3", Port: 443, Enabled: true,
			RawJSON: `{"name":"US-off","type":"ss","server":"3.3.3.3","port":443}`},
	}
	for i := range proxies {
		if err := db.Create(&proxies[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Model(&database.Proxy{}).Where("name = ?", "US-off").Update("enabled", false).Error; err != nil {
		t.Fatal(err)
	}

	box, err := crypto.NewBox("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, box, 0, 0)

	all, err := svc.EnabledProxiesBySourceIDs(nil)
	if err != nil {
		t.Fatalf("EnabledProxiesBySourceIDs(nil): %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("want 2 enabled proxies, got %d", len(all))
	}
	for _, p := range all {
		if _, ok := p["_source_id"]; !ok {
			t.Fatalf("missing _source_id: %+v", p)
		}
		if name, _ := p["_source_name"].(string); name == "" {
			t.Fatalf("missing _source_name: %+v", p)
		}
	}

	onlyA, err := svc.EnabledProxiesBySourceIDs([]uint{srcA.ID})
	if err != nil {
		t.Fatalf("EnabledProxiesBySourceIDs(A): %v", err)
	}
	if len(onlyA) != 1 {
		t.Fatalf("want 1 proxy for source A, got %d", len(onlyA))
	}
	if onlyA[0]["_source_name"] != "良心云" {
		t.Fatalf("source name = %v, want 良心云", onlyA[0]["_source_name"])
	}
}
