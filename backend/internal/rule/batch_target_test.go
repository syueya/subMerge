package rule

import (
	"path/filepath"
	"testing"

	"github.com/submerge/submerge/backend/internal/database"
)

func TestBatchUpdateRulesTarget(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "batch_target.db"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })

	rows := []database.Rule{
		{Type: "DOMAIN-SUFFIX", Payload: "a.com", Target: "直连", Enabled: true, SortOrder: 10},
		{Type: "DOMAIN-SUFFIX", Payload: "b.com", Target: "直连", Enabled: true, SortOrder: 20},
		{Type: "DOMAIN", Payload: "c.com", Target: "日本JP", Enabled: true, SortOrder: 30},
	}
	for i := range rows {
		if err := db.Create(&rows[i]).Error; err != nil {
			t.Fatal(err)
		}
	}

	svc := NewService(db)
	n, err := svc.BatchUpdateRulesTarget([]uint{rows[0].ID, rows[1].ID, rows[0].ID}, "美国US")
	if err != nil {
		t.Fatalf("BatchUpdateRulesTarget: %v", err)
	}
	if n != 2 {
		t.Fatalf("updated = %d, want 2", n)
	}

	var got []database.Rule
	if err := db.Order("sort_order asc").Find(&got).Error; err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("rules = %d, want 3", len(got))
	}
	if got[0].Target != "美国US" || got[1].Target != "美国US" {
		t.Fatalf("first two targets = %q/%q, want 美国US", got[0].Target, got[1].Target)
	}
	if got[2].Target != "日本JP" {
		t.Fatalf("third target = %q, want 日本JP", got[2].Target)
	}

	if _, err := svc.BatchUpdateRulesTarget([]uint{rows[0].ID}, "  "); err == nil {
		t.Fatal("empty target should fail")
	}
	n, err = svc.BatchUpdateRulesTarget(nil, "直连")
	if err != nil || n != 0 {
		t.Fatalf("empty ids: n=%d err=%v", n, err)
	}
}
