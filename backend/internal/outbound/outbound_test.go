package outbound

import (
	"strings"
	"testing"
)

func TestProxyURLValidationAndMasking(t *testing.T) {
	valid := []string{
		"http://127.0.0.1:7890",
		"https://user:secret@example.test:443",
		"socks5://proxy.internal",
		"socks5h://user:secret@[::1]:1080",
	}
	for _, raw := range valid {
		if err := ValidateProxyURL(raw); err != nil {
			t.Errorf("ValidateProxyURL(%q): %v", raw, err)
		}
	}
	for _, raw := range []string{"ftp://proxy.test:21", "http://proxy.test:0", "http://proxy.test:65536", "http://:7890"} {
		if err := ValidateProxyURL(raw); err == nil {
			t.Errorf("ValidateProxyURL(%q) succeeded, want error", raw)
		}
	}
	masked := MaskURL("https://alice:secret@example.test:443/path")
	if strings.Contains(masked, "secret") || !strings.Contains(masked, "alice:***@") {
		t.Fatalf("masked URL = %q", masked)
	}
}