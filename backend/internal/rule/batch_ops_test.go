package rule

import (
	"path/filepath"
	"testing"

	"github.com/submerge/submerge/backend/internal/database"
)

func TestBatchUpdateRulesEnabledAndCategoryAndDelete(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "batch_ops.db"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })

	rows := []database.Rule{
		{Type: "DOMAIN-SUFFIX", Payload: "a.com", Target: "直连", Enabled: true, SortOrder: 10, Category: "其它"},
		{Type: "DOMAIN-SUFFIX", Payload: "b.com", Target: "直连", Enabled: true, SortOrder: 20, Category: "其它"},
		{Type: "DOMAIN", Payload: "c.com", Target: "日本JP", Enabled: true, SortOrder: 30, Category: "其它"},
	}
	for i := range rows {
		if err := db.Create(&rows[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	svc := NewService(db)

	n, err := svc.BatchUpdateRulesEnabled([]uint{rows[0].ID, rows[1].ID}, false)
	if err != nil {
		t.Fatalf("BatchUpdateRulesEnabled: %v", err)
	}
	if n != 2 {
		t.Fatalf("enabled updated = %d, want 2", n)
	}
	var a database.Rule
	if err := db.First(&a, rows[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	if a.Enabled {
		t.Fatal("row0 should be disabled")
	}

	n, err = svc.BatchUpdateRulesCategory([]uint{rows[0].ID, rows[2].ID}, "海外AI")
	if err != nil {
		t.Fatalf("BatchUpdateRulesCategory: %v", err)
	}
	if n != 2 {
		t.Fatalf("category updated = %d, want 2", n)
	}
	if err := db.First(&a, rows[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	if a.Category != "海外AI" {
		t.Fatalf("row0 category = %q", a.Category)
	}

	n, err = svc.BatchDeleteRules([]uint{rows[1].ID, rows[1].ID})
	if err != nil {
		t.Fatalf("BatchDeleteRules: %v", err)
	}
	if n != 1 {
		t.Fatalf("deleted = %d, want 1", n)
	}
	var count int64
	if err := db.Model(&database.Rule{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("remaining = %d, want 2", count)
	}
}

func TestInferAndBackfillRuleCategories(t *testing.T) {
	if got := inferRuleCategory("DOMAIN-SUFFIX", "openai.com", "美国US", "AI-OpenAI"); got != "海外AI" {
		t.Fatalf("overseas AI: %q", got)
	}
	if got := inferRuleCategory("DOMAIN-SUFFIX", "deepseek.com", "直连", "AI-DeepSeek"); got != "国内AI" {
		t.Fatalf("domestic AI: %q", got)
	}
if got := inferRuleCategory("MATCH", "", "日本JP", "默认走代理"); got != "系统分类" {
			t.Fatalf("match: %q", got)
		}
		if got := inferRuleCategory("GEOSITE", "category-ads-all", "拒绝", "广告"); got != "系统分类" {
			t.Fatalf("ads: %q", got)
		}
		if got := inferRuleCategory("GEOIP", "CN", "直连", "国内直连"); got != "系统分类" {
			t.Fatalf("geoip cn: %q", got)
		}
	if got := inferRuleCategory("DOMAIN", "pt.example.com", "直连", "馒头"); got != "" {
		t.Fatalf("PT nickname should not invent category, got %q", got)
	}
}
