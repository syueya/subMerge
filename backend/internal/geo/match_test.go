package geo

import (
	"os"
	"path/filepath"
	"testing"
)

func boolPtr(v bool) *bool { return &v }

func TestMatchRulesGeoSiteAndMatch(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, GeoSiteFile), fixtureGeoSite(), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, GeoIPFile), fixtureGeoIP(), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := NewService(dir, URLs{})
	svc.Load()

	rules := []MatchRule{
		{Type: "GEOSITE", Payload: "test", Target: "美国US"},
		{Type: "MATCH", Target: "美国US"},
	}
	res, err := svc.MatchRules("www.example.com", rules, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Matched || res.FallbackMatch || res.Rule == nil || res.Rule.Type != "GEOSITE" {
		t.Fatalf("expected GEOSITE hit, got %+v", res)
	}
	if res.GeoHit == nil || res.GeoHit.Category != "test" || res.GeoHit.Value != "example.com" {
		t.Fatalf("expected geoHit for test/example.com, got %+v", res.GeoHit)
	}
	if res.Note != "命中 GeoSite 分类" {
		t.Fatalf("note=%q", res.Note)
	}
}

func TestMatchRulesFallsToMATCH(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, GeoSiteFile), fixtureGeoSite(), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := NewService(dir, URLs{})
	svc.Load()

	rules := []MatchRule{
		{Type: "GEOSITE", Payload: "test", Target: "美国US"},
		{Type: "MATCH", Target: "美国US"},
	}
	res, err := svc.MatchRules("not-in-list.example.org", rules, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Matched || !res.FallbackMatch || res.Rule == nil || res.Rule.Type != "MATCH" {
		t.Fatalf("expected MATCH fallback, got %+v", res)
	}
}

func TestMatchRulesSkipsDisabled(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, GeoSiteFile), fixtureGeoSite(), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := NewService(dir, URLs{})
	svc.Load()

	rules := []MatchRule{
		{Type: "GEOSITE", Payload: "test", Target: "美国US", Enabled: boolPtr(false)},
		{Type: "MATCH", Target: "直连"},
	}
	res, err := svc.MatchRules("www.example.com", rules, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Matched || !res.FallbackMatch || res.Rule == nil || res.Rule.Target != "直连" {
		t.Fatalf("disabled GEOSITE should be skipped, got %+v", res)
	}
}

func TestMatchRulesGeoIP(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, GeoIPFile), fixtureGeoIP(), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := NewService(dir, URLs{})
	svc.Load()

	rules := []MatchRule{
		{Type: "GEOIP", Payload: "test", Target: "直连"},
		{Type: "MATCH", Target: "美国US"},
	}
	res, err := svc.MatchRules("192.0.2.8", rules, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Matched || res.Rule == nil || res.Rule.Type != "GEOIP" {
		t.Fatalf("expected GEOIP hit, got %+v", res)
	}
	if res.GeoHit == nil || res.GeoHit.Category != "test" {
		t.Fatalf("expected geoHit, got %+v", res.GeoHit)
	}
}

func TestMatchRulesDomainSuffixAndEmpty(t *testing.T) {
	svc := NewService(t.TempDir(), URLs{})
	svc.Load()

	rules := []MatchRule{
		{Type: "DOMAIN-SUFFIX", Payload: "openai.com", Target: "美国US"},
		{Type: "MATCH", Target: "直连"},
	}
	res, err := svc.MatchRules("chat.openai.com", rules, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Matched || res.Rule == nil || res.Rule.Type != "DOMAIN-SUFFIX" {
		t.Fatalf("expected DOMAIN-SUFFIX, got %+v", res)
	}

	empty, err := svc.MatchRules("  ", rules, false)
	if err != nil {
		t.Fatal(err)
	}
	if empty.Kind != "empty" || empty.Matched {
		t.Fatalf("expected empty, got %+v", empty)
	}
}

func TestMatchRulesTooMany(t *testing.T) {
	svc := NewService(t.TempDir(), URLs{})
	rules := make([]MatchRule, maxMatchRules+1)
	for i := range rules {
		rules[i] = MatchRule{Type: "MATCH", Target: "x"}
	}
	if _, err := svc.MatchRules("a.com", rules, false); err == nil {
		t.Fatal("expected too many rules error")
	}
}

func TestMatchRulesRealGeoSiteOpenAI(t *testing.T) {
	dir := filepath.Join("..", "..", "defaults", "geo")
	if _, err := os.Stat(filepath.Join(dir, GeoSiteFile)); err != nil {
		t.Skip("bundled geosite.dat not present")
	}
	svc := NewService(dir, URLs{})
	svc.Load()
	rules := []MatchRule{
		{Type: "GEOSITE", Payload: "category-ai-!cn", Target: "美国US"},
		{Type: "MATCH", Target: "美国US"},
	}
	res, err := svc.MatchRules("chat.openai.com", rules, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Matched || res.FallbackMatch || res.Rule == nil || res.Rule.Type != "GEOSITE" {
		t.Fatalf("expected GEOSITE category-ai-!cn, got %+v", res)
	}
	if res.Rule.Payload != "category-ai-!cn" {
		t.Fatalf("payload=%s", res.Rule.Payload)
	}
}
