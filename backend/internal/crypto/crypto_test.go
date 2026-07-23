package crypto

import "testing"

func TestMaskURL(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"https://example.com/path?token=secret", "https://example.com/***/***"},
		{"https://uid:pass@airport.example/sub/abc?token=xyz", "https://airport.example/***/***"},
		{"https://user:p%40ss@host:8443/a/b", "https://host:8443/***/***"},
		{"http://[2001:db8::1]:8080/x", "http://[2001:db8::1]:8080/***/***"},
		{"not a url", "***"},
		{"ftp://files.example/x", "ftp://files.example/***/***"},
	}
	for _, tc := range cases {
		if got := MaskURL(tc.in); got != tc.want {
			t.Errorf("MaskURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestMaskToken(t *testing.T) {
	if got := MaskToken("short"); got != "****" {
		t.Fatalf("short token = %q", got)
	}
	if got := MaskToken("abcdefghijklmnop"); got != "abcd****mnop" {
		t.Fatalf("long token = %q", got)
	}
}

func TestBoxRoundTrip(t *testing.T) {
	box, err := NewBox("test-key")
	if err != nil {
		t.Fatal(err)
	}
	enc, err := box.Encrypt("hello")
	if err != nil {
		t.Fatal(err)
	}
	plain, err := box.Decrypt(enc)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "hello" {
		t.Fatalf("got %q", plain)
	}
}
