package geo

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
)

func fieldBytes(number protowire.Number, value []byte) []byte {
	result := protowire.AppendTag(nil, number, protowire.BytesType)
	return protowire.AppendBytes(result, value)
}

func fieldString(number protowire.Number, value string) []byte {
	return fieldBytes(number, []byte(value))
}

func fieldVarint(number protowire.Number, value uint64) []byte {
	result := protowire.AppendTag(nil, number, protowire.VarintType)
	return protowire.AppendVarint(result, value)
}

func fixtureGeoSite() []byte {
	domain := append(fieldVarint(1, 2), fieldString(2, "example.com")...)
	entry := append(fieldString(1, "TEST"), fieldBytes(2, domain)...)
	return fieldBytes(1, entry)
}

func fixtureGeoIP() []byte {
	cidr := append(fieldBytes(1, net.ParseIP("192.0.2.0").To4()), fieldVarint(2, 24)...)
	entry := append(fieldString(1, "TEST"), fieldBytes(2, cidr)...)
	return fieldBytes(1, entry)
}

func TestParseGeoFixtures(t *testing.T) {
	site, err := parseGeoSite(fixtureGeoSite())
	if err != nil || len(site) != 1 {
		t.Fatalf("parse geosite: entries=%d err=%v", len(site), err)
	}
	if !domainMatch(site[0], "www.example.com") {
		t.Fatal("domain matcher did not match suffix")
	}
	ip, err := parseGeoIP(fixtureGeoIP())
	if err != nil || len(ip) != 1 {
		t.Fatalf("parse geoip: entries=%d err=%v", len(ip), err)
	}
	if !ip[0].Network.Contains(net.ParseIP("192.0.2.8")) {
		t.Fatal("CIDR matcher did not match address")
	}
}

func TestNeedsBootstrap(t *testing.T) {
	empty := NewService(t.TempDir(), URLs{})
	empty.Load()
	if !empty.NeedsBootstrap() {
		t.Fatal("empty geo dir should need bootstrap")
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, GeoSiteFile), fixtureGeoSite(), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, GeoIPFile), fixtureGeoIP(), 0o644); err != nil {
		t.Fatal(err)
	}
	partial := NewService(dir, URLs{})
	partial.Load()
	// geosite/geoip 可用，metadb/asn 仍缺失 → 仍需 bootstrap
	if !partial.NeedsBootstrap() {
		t.Fatal("partial geo dir should still need bootstrap")
	}
}

func TestServiceQueryReverseAndInvalidResources(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, GeoSiteFile), fixtureGeoSite(), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, GeoIPFile), fixtureGeoIP(), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, MetaDBFile), []byte("invalid"), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := NewService(dir, URLs{})
	svc.Load()
	statuses := svc.Status()
	if len(statuses) != 4 || !statuses[0].Available || !statuses[1].Available || statuses[2].Available {
		t.Fatalf("unexpected statuses: %+v", statuses)
	}
result, err := svc.Query("WWW.Example.com.", false)
		if err != nil {
			t.Fatal(err)
		}
		if result.Domain != "www.example.com" || result.InputType != "domain" || len(result.GeoSite) != 1 || !result.ResolveSkipped {
			t.Fatalf("unexpected query result: %+v", result)
		}

		ipResult, err := svc.Query("192.0.2.8", false)
		if err != nil {
			t.Fatal(err)
		}
		if ipResult.InputType != "ip" || ipResult.Domain != "192.0.2.8" || len(ipResult.IPs) != 1 || ipResult.IPs[0] != "192.0.2.8" {
			t.Fatalf("unexpected IP query result: %+v", ipResult)
		}
		if !ipResult.ResolveSkipped || len(ipResult.GeoSite) != 0 || len(ipResult.GeoIP) != 1 || ipResult.GeoIP[0].Category != "test" {
			t.Fatalf("unexpected IP geo lookup: %+v", ipResult)
		}

		if _, err := svc.Query("not a valid host", false); err == nil {
			t.Fatal("expected invalid query error")
		}

		reverse, err := svc.Reverse("geosite", "test", 10, 0)
		if err != nil || reverse.Total != 1 || reverse.Items[0].Value != "example.com" {
			t.Fatalf("unexpected reverse result: %+v err=%v", reverse, err)
		}
		unsupported, err := svc.Reverse("asn", "", 10, 0)
		if err != nil || unsupported.Message == "" {
			t.Fatalf("expected unsupported reverse response: %+v err=%v", unsupported, err)
		}
	}

func TestSearchASNAndMetaDBValidation(t *testing.T) {
	if got := metaCodes("CN"); len(got) != 1 || got[0] != "CN" {
		t.Fatalf("unexpected meta code: %#v", got)
	}
	if got := metaCodes([]any{"CN", "AS"}); len(got) != 2 {
		t.Fatalf("unexpected meta codes: %#v", got)
	}
	svc := NewService(t.TempDir(), URLs{})
	if _, err := svc.Search("metadb", "organization", "cn", 10, 0); err == nil {
		t.Fatal("expected unsupported MetaDB field")
	}
	if _, err := svc.Search("asn", "asn", "", 10, 0); err == nil {
		t.Fatal("expected empty search keyword error")
	}
}

func TestAtomicReplace(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "resource")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := replaceResource(dir, "resource", []byte("new")); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(target)
	if err != nil || string(body) != "new" {
		t.Fatalf("unexpected replacement: %q err=%v", body, err)
	}
}
