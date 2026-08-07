package ipgeo

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLookupMapsIPWhoIsResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/1.1.1.1" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"ip":"1.1.1.1","continent":"North America","continent_code":"NA","country":"United States","country_code":"US","region":"California","region_code":"CA","city":"Los Angeles","postal":"90001","flag":{"img":"https://cdn.ipwhois.io/flags/us.svg","emoji":"🇺🇸","emoji_unicode":"U+1F1FA U+1F1F8"},"latitude":34.05,"longitude":-118.24,"connection":{"asn":13335,"org":"Cloudflare, Inc.","isp":"Cloudflare"}}`))
	}))
	defer srv.Close()
	client, err := NewClient(srv.URL+"/{ip}", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.Lookup(context.Background(), net.ParseIP("1.1.1.1"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Continent != "North America" || got.ContinentCode != "NA" || got.CountryCode != "US" || got.RegionCode != "CA" || got.City != "Los Angeles" || got.Postal != "90001" || got.Flag.Img != "https://cdn.ipwhois.io/flags/us.svg" || got.Flag.Emoji != "🇺🇸" || got.Flag.EmojiUnicode != "U+1F1FA U+1F1F8" || got.ASN != "13335" || got.ISP != "Cloudflare" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestLookupRejectsProviderFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":false,"message":"invalid ip"}`))
	}))
	defer srv.Close()
	client, err := NewClient(srv.URL+"/{ip}", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Lookup(context.Background(), net.ParseIP("1.1.1.1")); err == nil {
		t.Fatal("expected provider error")
	}
}

func TestNewClientRejectsRemoteHTTP(t *testing.T) {
	if _, err := NewClient("http://example.com/{ip}", time.Second); err == nil {
		t.Fatal("expected HTTPS validation error")
	}
}
