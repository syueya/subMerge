package source

import (
	"errors"
	"net"
	"net/url"
	"strings"
	"testing"
)

func TestIsBlockedIP(t *testing.T) {
	blocked := []string{
		"127.0.0.1",
		"::1",
		"10.0.0.1",
		"192.168.1.1",
		"172.16.0.1",
		"169.254.1.1",
		"100.64.0.1",
		"100.127.255.255",
		"0.0.0.0",
		"224.0.0.1",
		"ff02::1",
	}
	for _, s := range blocked {
		ip := net.ParseIP(s)
		if !isBlockedIP(ip) {
			t.Errorf("expected blocked: %s", s)
		}
	}
	allowed := []string{
		"8.8.8.8",
		"1.1.1.1",
		"2001:4860:4860::8888",
	}
	for _, s := range allowed {
		ip := net.ParseIP(s)
		if isBlockedIP(ip) {
			t.Errorf("expected allowed: %s", s)
		}
	}
	if !isBlockedIP(nil) {
		t.Error("nil should be blocked")
	}
}

func TestCleanFetchErrStripsURL(t *testing.T) {
	raw := "https://uid:secret@airport.example/sub?token=abc"
	inner := &url.Error{Op: "Get", URL: raw, Err: errors.New("connection refused")}
	got := cleanFetchErr(inner, raw)
	if strings.Contains(got, "secret") || strings.Contains(got, "token=abc") || strings.Contains(got, "uid:") {
		t.Fatalf("leaked credentials in %q", got)
	}
	if !strings.Contains(got, "connection refused") {
		t.Fatalf("lost root cause: %q", got)
	}
}
