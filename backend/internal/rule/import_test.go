package rule

import (
	"testing"

	common "github.com/submerge/submerge/backend/common"
)

func TestParseBatchImportText(t *testing.T) {
	text := `
# comment
DOMAIN-SUFFIX,xiaxiazi.ccwu.cc,直连,个人-小鸡
DOMAIN,zcode.z.ai,直连,个人-工具
gpt-api.xxww.online
DOMAIN-KEYWORD,hybgzs,直连
MATCH,日本JP
IP-CIDR,156.225.79.36/32,直连,小鸡
`
	ok, errs := parseBatchImportText(text, "DOMAIN", "直连", "个人")
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %v", errs)
	}
	if len(ok) != 6 {
		t.Fatalf("want 6 rules, got %d: %+v", len(ok), ok)
	}
	if ok[0].Type != "DOMAIN-SUFFIX" || ok[0].Payload != "xiaxiazi.ccwu.cc" || ok[0].Target != "直连" {
		t.Fatalf("row0: %+v", ok[0])
	}
	if ok[2].Type != "DOMAIN" || ok[2].Payload != "gpt-api.xxww.online" || ok[2].Note != "个人" {
		t.Fatalf("payload-only row: %+v", ok[2])
	}
	if ok[4].Type != string(common.RuleTypeMatch) || ok[4].Payload != "" || ok[4].Target != "日本JP" {
		t.Fatalf("match row: %+v", ok[4])
	}
}

func TestParseBatchImportText_Errors(t *testing.T) {
	// 仅 payload 但无默认出口
	_, errs := parseBatchImportText("only.domain.com\n", "DOMAIN", "", "")
	if len(errs) == 0 {
		t.Fatal("expected error for payload-only without default target")
	}
	// 缺 payload
	_, errs = parseBatchImportText("DOMAIN-SUFFIX,,直连\n", "", "直连", "")
	if len(errs) == 0 {
		t.Fatal("expected error for empty payload")
	}
}
