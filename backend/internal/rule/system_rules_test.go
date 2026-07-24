package rule

import (
	"path/filepath"
	"testing"

	"github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/database"
)

func TestSystemRulesProtectedOnCRUD(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "sys.db"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })

	svc := NewService(db)
	if err := svc.SeedDefaults(); err != nil {
		t.Fatal(err)
	}

	list, err := svc.ListRules()
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) < 3 {
		t.Fatalf("rules=%d", len(list.Items))
	}
	first := list.Items[0]
	last := list.Items[len(list.Items)-1]
	secondLast := list.Items[len(list.Items)-2]
	if first.Type != "GEOSITE" || first.Payload != "category-ads-all" {
		t.Fatalf("first=%+v", first)
	}
	if secondLast.Type != "GEOIP" || secondLast.Payload != "CN" {
		t.Fatalf("secondLast=%+v", secondLast)
	}
	if last.Type != "MATCH" || last.Target != "美国US" {
		t.Fatalf("last=%+v", last)
	}

	// 删除系统规则应失败
	if err := svc.DeleteRule(first.ID); err == nil {
		t.Fatal("delete ads should fail")
	}
	if err := svc.DeleteRule(last.ID); err == nil {
		t.Fatal("delete match should fail")
	}

	// 编辑系统规则：改 type/payload 无效，target 可改
	en := true
	got, err := svc.UpdateRule(last.ID, common.UpsertRuleRequest{
		Type: "DOMAIN", Payload: "evil.com", Target: "日本JP", Enabled: &en, Note: "我改兜底", Category: "其它",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != "MATCH" || got.Payload != "" || got.Category != "系统分类" {
		t.Fatalf("system identity changed: %+v", got)
	}
	if got.Target != "日本JP" || got.Note != "我改兜底" {
		t.Fatalf("allowed fields not updated: %+v", got)
	}

	// 新增业务规则应落在国内/MATCH 之前，且系统仍钉死两端
	created, err := svc.CreateRule(common.UpsertRuleRequest{
		Type: "DOMAIN-SUFFIX", Payload: "example-new.test", Target: "直连", Enabled: &en, Category: "其它",
	})
	if err != nil {
		t.Fatal(err)
	}
	list2, err := svc.ListRules()
	if err != nil {
		t.Fatal(err)
	}
	items := list2.Items
	if items[0].Type != "GEOSITE" || items[0].Payload != "category-ads-all" {
		t.Fatalf("ads not first after create: %+v", items[0])
	}
	if items[len(items)-1].Type != "MATCH" {
		t.Fatalf("match not last: %+v", items[len(items)-1])
	}
	if items[len(items)-2].Type != "GEOIP" {
		t.Fatalf("geoip not second last: %+v", items[len(items)-2])
	}
	// created should be before geoip/match
	var createdOrder, geoOrder, matchOrder int
	for _, r := range items {
		if r.ID == created.ID {
			createdOrder = r.SortOrder
		}
		if r.Type == "GEOIP" && r.Payload == "CN" {
			geoOrder = r.SortOrder
		}
		if r.Type == "MATCH" {
			matchOrder = r.SortOrder
		}
	}
	if !(createdOrder < geoOrder && geoOrder < matchOrder) {
		t.Fatalf("order create=%d geo=%d match=%d", createdOrder, geoOrder, matchOrder)
	}

	// 重排试图把 match 挪到最前，仍应被钉回
	ids := make([]uint, len(items))
	// reverse
	for i, r := range items {
		ids[len(items)-1-i] = r.ID
	}
	if err := svc.ReorderRules(ids); err != nil {
		t.Fatal(err)
	}
	list3, _ := svc.ListRules()
	if list3.Items[0].Type != "GEOSITE" || list3.Items[len(list3.Items)-1].Type != "MATCH" {
		t.Fatalf("reorder broke system ends: first=%s last=%s", list3.Items[0].Type, list3.Items[len(list3.Items)-1].Type)
	}
}
