package rule

import (
	"testing"

	common "github.com/submerge/submerge/backend/common"
)

func TestParseBatchImportText(t *testing.T) {
	text := `
# comment
个人,DOMAIN-SUFFIX,xiaxiazi.ccwu.cc,直连,域名
海外AI,DOMAIN,zcode.z.ai,直连,个人-工具
gpt-api.xxww.online
其它,DOMAIN-KEYWORD,hybgzs,直连
系统分类,MATCH,日本JP
其它,IP-CIDR,156.225.79.36/32,直连,小鸡
`
	ok, errs := parseBatchImportText(text, "DOMAIN", "直连", "个人", "其它")
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %v", errs)
	}
	if len(ok) != 6 {
		t.Fatalf("want 6 rules, got %d: %+v", len(ok), ok)
	}
	if ok[0].Category != "个人" || ok[0].Type != "DOMAIN-SUFFIX" || ok[0].Payload != "xiaxiazi.ccwu.cc" || ok[0].Target != "直连" || ok[0].Note != "域名" {
		t.Fatalf("row0: %+v", ok[0])
	}
	if ok[1].Category != "海外AI" || ok[1].Type != "DOMAIN" || ok[1].Payload != "zcode.z.ai" {
		t.Fatalf("row1: %+v", ok[1])
	}
	if ok[2].Type != "DOMAIN" || ok[2].Payload != "gpt-api.xxww.online" || ok[2].Note != "个人" || ok[2].Category != "其它" {
		t.Fatalf("payload-only row: %+v", ok[2])
	}
	if ok[3].Category != "其它" || ok[3].Type != "DOMAIN-KEYWORD" || ok[3].Payload != "hybgzs" || ok[3].Target != "直连" {
		t.Fatalf("row3 short form: %+v", ok[3])
	}
	if ok[4].Type != string(common.RuleTypeMatch) || ok[4].Payload != "" || ok[4].Target != "日本JP" || ok[4].Category != "系统分类" {
		t.Fatalf("match row: %+v", ok[4])
	}
	if ok[5].Type != "IP-CIDR" || ok[5].Payload != "156.225.79.36/32" || ok[5].Note != "小鸡" {
		t.Fatalf("row5: %+v", ok[5])
	}
}

func TestParseBatchImportText_CategoryFirst(t *testing.T) {
	text := "海外AI,DOMAIN-SUFFIX,openai.com,美国US,AI-OpenAI\nPT站,DOMAIN,a.com,直连,备注\n"
	ok, errs := parseBatchImportText(text, "DOMAIN", "直连", "", "")
	if len(errs) != 0 {
		t.Fatalf("errs: %v", errs)
	}
	if len(ok) != 2 {
		t.Fatalf("got %d", len(ok))
	}
	if ok[0].Category != "海外AI" || ok[0].Note != "AI-OpenAI" || ok[0].Target != "美国US" {
		t.Fatalf("row0: %+v", ok[0])
	}
	if ok[1].Category != "PT站" || ok[1].Note != "备注" || ok[1].Payload != "a.com" {
		t.Fatalf("row1: %+v", ok[1])
	}
}

func TestParseBatchImportText_EmptyCategoryUsesDefault(t *testing.T) {
	text := ",DOMAIN-SUFFIX,example.com,直连,备注\n"
	ok, errs := parseBatchImportText(text, "DOMAIN", "直连", "", "个人")
	if len(errs) != 0 {
		t.Fatalf("errs: %v", errs)
	}
	if len(ok) != 1 {
		t.Fatalf("got %d", len(ok))
	}
	if ok[0].Category != "个人" || ok[0].Payload != "example.com" {
		t.Fatalf("row: %+v", ok[0])
	}
}

func TestParseBatchImportText_Errors(t *testing.T) {
	// 仅 payload 但无默认出口
	_, errs := parseBatchImportText("only.domain.com\n", "DOMAIN", "", "", "")
	if len(errs) == 0 {
		t.Fatal("expected error for payload-only without default target")
	}
	// 缺 payload
	_, errs = parseBatchImportText("个人,DOMAIN-SUFFIX,,直连\n", "", "直连", "", "")
	if len(errs) == 0 {
		t.Fatal("expected error for empty payload")
	}
	// 旧格式 TYPE 开头应失败
	_, errs = parseBatchImportText("DOMAIN-SUFFIX,xiaxiazi.ccwu.cc,直连,域名\n", "", "直连", "", "")
	if len(errs) == 0 {
		t.Fatal("expected error for old TYPE-first format")
	}
}
