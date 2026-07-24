package rule

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/submerge/submerge/backend/internal/database"
	"gorm.io/gorm"
)

// 单元测试用的假数据（不是 defaults/rules.yaml）。
// 复现你线上的历史错位：GEOIP=900、MATCH=343、后面还有个人域名；
// 迁移补缺 + ensureSystemRuleOrder 后：广告最先、GEOIP 倒数第二、MATCH 最后。
func TestEnsureMatchIsLastAfterSeedSync(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "order.db"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	// 故意构造「不完整老库」，用来测迁移，不是第二份默认规则定义
	fixture := []database.Rule{
		{Type: "DOMAIN-SUFFIX", Payload: "github.com", Target: "美国US", Enabled: true, SortOrder: 100, Note: "GitHub"},
		{Type: "DOMAIN", Payload: "zrpt.cc", Target: "直连", Enabled: true, SortOrder: 311, Note: "自然"},
		{Type: "GEOIP", Payload: "CN", Target: "直连", Enabled: true, SortOrder: 900, Note: "国内直连", Category: "国内"},
		// 历史错位：MATCH 在 GEOIP 前
		{Type: "MATCH", Payload: "", Target: "日本JP", Enabled: true, SortOrder: 343, Note: "默认走代理", Category: "兜底"},
		{Type: "DOMAIN-SUFFIX", Payload: "xiaxiazi.ccwu.cc", Target: "直连", Enabled: true, SortOrder: 312, Note: "个人域名"},
	}
	for i := range fixture {
		if err := db.Create(&fixture[i]).Error; err != nil {
			t.Fatal(err)
		}
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := ensureSystemRules(tx); err != nil {
			return err
		}
		if err := syncMissingDefaultRulesFromSeed(tx); err != nil {
			return err
		}
		if err := migrateSystemCategories(tx); err != nil {
			return err
		}
		return ensureSystemRuleOrder(tx)
	}); err != nil {
		t.Fatal(err)
	}

	var rows []database.Rule
	if err := db.Order("sort_order asc, id asc").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("no rules")
	}
	first := rows[0]
	if first.Type != "GEOSITE" || first.Payload != "category-ads-all" {
		t.Fatalf("first rule type=%s payload=%s, want ads GEOSITE first; all=%v",
			first.Type, first.Payload, summarizeOrders(rows))
	}
	last := rows[len(rows)-1]
	if last.Type != "MATCH" {
		t.Fatalf("last rule type=%s payload=%s order=%d, want MATCH last; all=%v",
			last.Type, last.Payload, last.SortOrder, summarizeOrders(rows))
	}
	// 旧默认 MATCH（日本JP + 默认走代理）应对齐到美国US
	if last.Target != "美国US" {
		t.Fatalf("MATCH target=%s, want 美国US", last.Target)
	}
	if len(rows) < 2 {
		t.Fatal("need at least ads + match")
	}
	secondLast := rows[len(rows)-2]
	if secondLast.Type != "GEOIP" || !strings.EqualFold(secondLast.Payload, "CN") {
		t.Fatalf("second-last type=%s payload=%s, want GEOIP CN; all=%v",
			secondLast.Type, secondLast.Payload, summarizeOrders(rows))
	}

	// 应从 rules.yaml 补上缺失公共规则（如 docker.io），且排在 MATCH 前
	foundDocker := false
	for _, r := range rows {
		if r.Payload == "docker.io" {
			foundDocker = true
			if r.SortOrder >= last.SortOrder {
				t.Fatalf("docker rule order %d must be before MATCH %d", r.SortOrder, last.SortOrder)
			}
		}
	}
	if !foundDocker {
		t.Fatal("expected docker.io filled from rules.yaml")
	}

	// 旧分类名应迁成「系统分类」
	for _, r := range rows {
		if r.Type == "MATCH" || (r.Type == "GEOIP" && strings.EqualFold(r.Payload, "CN")) ||
			(r.Type == "GEOSITE" && r.Payload == "category-ads-all") {
			if r.Category != "系统分类" {
				t.Fatalf("%s:%s category=%q, want 系统分类", r.Type, r.Payload, r.Category)
			}
		}
	}
}

// 已有 type+payload 时不改用户出口；只补缺
func TestSyncMissingDefaultRulesKeepsUserTarget(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "keep.db"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := db.Create(&database.Rule{
		Type: "DOMAIN-SUFFIX", Payload: "github.com", Target: "直连",
		Enabled: true, SortOrder: 10, Note: "我改过",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&database.Rule{
		Type: "MATCH", Payload: "", Target: "日本JP",
		Enabled: true, SortOrder: 100, Note: "默认走代理",
	}).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		return syncMissingDefaultRulesFromSeed(tx)
	}); err != nil {
		t.Fatal(err)
	}

	var gh database.Rule
	if err := db.Where("type = ? AND payload = ?", "DOMAIN-SUFFIX", "github.com").First(&gh).Error; err != nil {
		t.Fatal(err)
	}
	if gh.Target != "直连" {
		t.Fatalf("user target overwritten: got %s", gh.Target)
	}
	if gh.Note != "我改过" {
		t.Fatalf("user note overwritten: got %s", gh.Note)
	}
}

func summarizeOrders(rows []database.Rule) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, fmt.Sprintf("%s:%s@%d", r.Type, r.Payload, r.SortOrder))
	}
	return out
}
