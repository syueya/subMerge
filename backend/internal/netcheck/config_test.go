package netcheck

import "testing"

func TestNormalizeTargetURLRejectsPrivateAddresses(t *testing.T) {
	for _, raw := range []string{
		"http://127.0.0.1/",
		"http://10.0.0.1/",
		"http://169.254.169.254/latest/meta-data/",
		"http://100.64.0.1/",
		"http://[::1]/",
	} {
		if _, err := normalizeTargetURL(raw); err == nil {
			t.Fatalf("normalizeTargetURL(%q) succeeded", raw)
		}
	}
}

func TestNormalizeTargetURLAcceptsPublicAddress(t *testing.T) {
	if _, err := normalizeTargetURL("https://8.8.8.8/"); err != nil {
		t.Fatalf("public target rejected: %v", err)
	}
}

func TestNormalizeTargetURLRejectsUnsupportedScheme(t *testing.T) {
	if _, err := normalizeTargetURL("file:///etc/passwd"); err == nil {
		t.Fatal("unsupported scheme accepted")
	}
}

func TestDoHTTPRejectsPrivateAddressBeforeTransport(t *testing.T) {
	result := doHTTP(nil, "http://127.0.0.1/", "HEAD", 1)
	if result.OK {
		t.Fatal("private target was sent to transport")
	}
	if result.Error != "目标地址禁止访问内网或保留地址" {
		t.Fatalf("unexpected error: %q", result.Error)
	}
}
