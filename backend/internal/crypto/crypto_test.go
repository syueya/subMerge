package crypto

import (
	"encoding/hex"
	"os"
	"runtime"
	"strings"
	"testing"
)

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

func TestBoxWithSaltRoundTrip(t *testing.T) {
	salt := make([]byte, SaltLen)
	for i := range salt {
		salt[i] = byte(i)
	}
	box, err := NewBoxWithSalt("test-key", salt)
	if err != nil {
		t.Fatal(err)
	}
	enc, err := box.Encrypt("hello")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(enc, encV2Prefix) {
		t.Fatalf("expected %q prefix, got %q", encV2Prefix, enc)
	}
	plain, err := box.Decrypt(enc)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "hello" {
		t.Fatalf("got %q", plain)
	}
}

// TestBoxDecryptsLegacyCiphertext 强派生 Box 必须能解密旧 sha256 密文（无 v2: 前缀）。
func TestBoxDecryptsLegacyCiphertext(t *testing.T) {
	legacy, err := NewBox("test-key")
	if err != nil {
		t.Fatal(err)
	}
	oldEnc, err := legacy.Encrypt("secret")
	if err != nil {
		t.Fatal(err)
	}
	// 历史密文没有 v2: 前缀；去掉前缀以模拟旧版本落库的数据。
	oldEnc = strings.TrimPrefix(oldEnc, encV2Prefix)

	salt := make([]byte, SaltLen)
	strong, err := NewBoxWithSalt("test-key", salt)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := strong.Decrypt(oldEnc)
	if err != nil {
		t.Fatalf("strong box failed to decrypt legacy ciphertext: %v", err)
	}
	if plain != "secret" {
		t.Fatalf("got %q", plain)
	}
}

func TestLoadOrCreateKey(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/crypto.key"
	key1, err := LoadOrCreateKey(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(key1) != KeyLen*2 {
		t.Fatalf("key len = %d, want %d", len(key1), KeyLen*2)
	}
	if _, err := hex.DecodeString(key1); err != nil {
		t.Fatalf("generated key is not hex: %v", err)
	}
	key2, err := LoadOrCreateKey(path)
	if err != nil {
		t.Fatal(err)
	}
	if key1 != key2 {
		t.Fatal("key should be stable across loads")
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("key file mode = %o, want 600", info.Mode().Perm())
		}
	}
}

func TestLoadOrCreateKeyRejectsCorruptFile(t *testing.T) {
	path := t.TempDir() + "/crypto.key"
	if err := os.WriteFile(path, []byte("not-a-key\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOrCreateKey(path); err == nil {
		t.Fatal("expected corrupt key file to fail")
	}
}

func TestLoadOrCreateSalt(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/salt.bin"
	s1, err := LoadOrCreateSalt(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(s1) != SaltLen {
		t.Fatalf("salt len = %d", len(s1))
	}
	s2, err := LoadOrCreateSalt(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(s1) != string(s2) {
		t.Fatal("salt should be stable across loads")
	}
}
