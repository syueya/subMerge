package source

import "testing"

func TestRegionDictLoads(t *testing.T) {
	if err := RegionDictError(); err != nil {
		t.Fatal(err)
	}
	if len(flagToRegion) == 0 || len(keywordToRegion) == 0 {
		t.Fatal("region dict empty after load")
	}
}

func TestDetectRegionEmojiAndKeywords(t *testing.T) {
	cases := map[string]string{
		"🇯🇵日本高速01|CTCU|0.5x": "JP",
		"🇭🇰香港高速01|BGP|CMCU":  "HK",
		"美国洛杉矶01":            "US",
		"Japan-Tokyo-01":     "JP",
		"Hong Kong BGP":      "HK",
		"SG-Singapore-1":     "SG",
		"random-node":        "",
	}
	for name, want := range cases {
		got := DetectRegion(name)
		if got != want {
			t.Fatalf("%q: got %q want %q", name, got, want)
		}
	}
}

func TestResolveRegionModes(t *testing.T) {
	if got := ResolveRegion("🇭🇰香港01", "fixed", "JP"); got != "JP" {
		t.Fatalf("fixed mode: got %s", got)
	}
	if got := ResolveRegion("🇭🇰香港01", "auto", "JP"); got != "HK" {
		t.Fatalf("auto mode: got %s", got)
	}
	if got := ResolveRegion("mystery", "auto", "US"); got != "US" {
		t.Fatalf("fallback: got %s", got)
	}
}

func TestDetectRegionDetailedMethods(t *testing.T) {
	d := DetectRegionDetailed("🇯🇵日本高速01")
	if d.Region != "JP" || d.Method != "flag" || d.Matched == "" {
		t.Fatalf("flag: %+v", d)
	}
	d = DetectRegionDetailed("Hong Kong BGP")
	if d.Region != "HK" || d.Method != "keyword" {
		t.Fatalf("keyword: %+v", d)
	}
	d = DetectRegionDetailed("SG-Singapore-1")
	// 可能被 keyword Singapore 或 prefix SG 命中，只要是 SG 即可
	if d.Region != "SG" || (d.Method != "keyword" && d.Method != "prefix") {
		t.Fatalf("sg: %+v", d)
	}
	d = DetectRegionDetailed("random-node")
	if d.Region != "" || d.Method != "none" {
		t.Fatalf("none: %+v", d)
	}

	r := ResolveRegionDetailed("mystery", "auto", "US")
	if !r.UsedFallback || r.Region != "US" || r.Detect.Method != "none" {
		t.Fatalf("fallback detail: %+v", r)
	}
	r = ResolveRegionDetailed("🇭🇰香港01", "fixed", "JP")
	if r.Region != "JP" || r.Detect.Method != "fixed" || r.UsedFallback {
		t.Fatalf("fixed detail: %+v", r)
	}
}

func TestRegionConflictPrefersKeyword(t *testing.T) {
	d := DetectRegionDetailed("🇨🇳台湾高速01")
	if d.Region != "TW" || d.Method != "conflict" || !d.Conflict {
		t.Fatalf("conflict resolution: %+v", d)
	}
	if d.FlagRegion != "CN" || d.KeywordRegion != "TW" {
		t.Fatalf("conflict regions: %+v", d)
	}
	// 多国旗时匹配顺序应稳定（按 flag 排序后取第一个）
	first := DetectRegion("🇯🇵🇭🇰node")
	for i := 0; i < 10; i++ {
		if got := DetectRegion("🇯🇵🇭🇰node"); got != first {
			t.Fatalf("non-deterministic flag match: first=%q got=%q", first, got)
		}
	}
}

func TestProxyFingerprintStable(t *testing.T) {
	a := ParsedProxy{
		Name: "rename-me", Type: "vless", Server: "a.com", Port: 443,
		Raw: map[string]interface{}{"uuid": "abc-123"},
	}
	b := ParsedProxy{
		Name: "other-name", Type: "vless", Server: "a.com", Port: 443,
		Raw: map[string]interface{}{"uuid": "abc-123"},
	}
	if ProxyFingerprint(a) != ProxyFingerprint(b) {
		t.Fatal("fingerprint should ignore display name when uuid present")
	}
	c := ParsedProxy{
		Name: "other-name", Type: "vless", Server: "a.com", Port: 443,
		Raw: map[string]interface{}{"uuid": "zzz"},
	}
	if ProxyFingerprint(a) == ProxyFingerprint(c) {
		t.Fatal("different uuid should differ")
	}
}
