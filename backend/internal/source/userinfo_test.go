package source

import (
	"net/http"
	"testing"
)

func TestParseSubscriptionUserInfo(t *testing.T) {
	info, ok := ParseSubscriptionUserInfo("upload=100; download=200; total=1073741824; expire=1893456000")
	if !ok {
		t.Fatal("expected ok")
	}
	if info.Upload != 100 || info.Download != 200 || info.Total != 1073741824 || info.Expire != 1893456000 {
		t.Fatalf("got %+v", info)
	}
	if info.Used() != 300 {
		t.Fatalf("used=%d", info.Used())
	}
}

func TestParseSubscriptionUserInfoFromHeadersMeta(t *testing.T) {
	h := http.Header{}
	h.Set("x-amz-meta-subscription-userinfo", "upload=1; download=2; total=3; expire=4")
	info, ok := ParseSubscriptionUserInfoFromHeaders(h)
	if !ok || info.Upload != 1 || info.Download != 2 || info.Total != 3 || info.Expire != 4 {
		t.Fatalf("meta header: ok=%v info=%+v", ok, info)
	}
}

func TestMergeSubscriptionUserInfo(t *testing.T) {
	a := SubscriptionUserInfo{Upload: 10, Download: 20, Total: 1000, Expire: 2000}
	b := SubscriptionUserInfo{Upload: 5, Download: 15, Total: 500, Expire: 1500}
	c := SubscriptionUserInfo{} // empty ignored for expire
	m := MergeSubscriptionUserInfo([]SubscriptionUserInfo{a, b, c})
	if m.Upload != 15 || m.Download != 35 || m.Total != 1500 {
		t.Fatalf("sum fields: %+v", m)
	}
	if m.Expire != 1500 {
		t.Fatalf("expire should be min non-zero, got %d", m.Expire)
	}
	header := FormatSubscriptionUserInfoHeader(m)
	if header != "upload=15; download=35; total=1500; expire=1500" {
		t.Fatalf("header=%q", header)
	}
}
