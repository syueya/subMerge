package updater

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestManifestParseValidateAndSelect(t *testing.T) {
	raw := validManifestJSON("1.2.0", strings.Repeat("a", 64), 12)
	manifest, err := ParseManifest(raw)
	if err != nil {
		t.Fatal(err)
	}
	asset, err := manifest.SelectAsset("linux", "arm64")
	if err != nil {
		t.Fatal(err)
	}
	if asset.Name != "submerge-linux-arm64" || asset.Size != 12 {
		t.Fatalf("selected asset = %+v", asset)
	}
	if _, err := manifest.SelectAsset("darwin", "arm64"); !errors.Is(err, ErrNoAsset) {
		t.Fatalf("missing asset error = %v", err)
	}
}

func TestManifestRejectsInvalidSchemasAndAssets(t *testing.T) {
	tests := []string{
		`{"schemaVersion":2,"version":"1.2.0","publishedAt":"2026-01-02T03:04:05Z","assets":[]}`,
		`{"schemaVersion":1,"version":"bad","publishedAt":"2026-01-02T03:04:05Z","assets":[]}`,
		`{"schemaVersion":1,"version":"1.2.0","publishedAt":"0001-01-01T00:00:00Z","assets":[]}`,
		`{"schemaVersion":1,"version":"1.2.0","publishedAt":"2026-01-02T03:04:05Z","unknown":true,"assets":[]}`,
		`{"schemaVersion":1,"version":"1.2.0","publishedAt":"2026-01-02T03:04:05Z","assets":[]} {}`,
		`{"schemaVersion":1,"version":"1.2.0","publishedAt":"2026-01-02T03:04:05Z","assets":[{"os":"linux","arch":"amd64","name":"../bad","url":"https://example.com/a","sha256":"aa","size":1}]}`,
		`{"schemaVersion":1,"version":"1.2.0","publishedAt":"2026-01-02T03:04:05Z","assets":[{"os":"linux","arch":"amd64","name":"a","url":"http://example.com/a","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":1}]}`,
	}
	for _, raw := range tests {
		if _, err := ParseManifest([]byte(raw)); err == nil {
			t.Errorf("invalid manifest accepted: %s", raw)
		}
	}
}

func TestManifestRejectsDuplicatePlatform(t *testing.T) {
	raw := fmt.Sprintf(`{"schemaVersion":1,"version":"1.2.0","publishedAt":"2026-01-02T03:04:05Z","assets":[%s,%s]}`,
		validAssetJSON("amd64", strings.Repeat("a", 64), 1), validAssetJSON("amd64", strings.Repeat("b", 64), 2))
	if _, err := ParseManifest([]byte(raw)); err == nil {
		t.Fatal("duplicate platform was accepted")
	}
}

func TestVerifyManifestSignatureEnvelope(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	manifest := validManifestJSON("1.2.0", strings.Repeat("a", 64), 12)
	keyID, err := PublicKeyID(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	sig := ed25519.Sign(privateKey, manifest)
	envelope := []byte(fmt.Sprintf(`{"algorithm":"ed25519","keyId":"%s","signature":"%s"}`, keyID, base64.StdEncoding.EncodeToString(sig)))
	if err := VerifyManifestSignature(publicKey, manifest, envelope); err != nil {
		t.Fatal(err)
	}

	mutated := append([]byte(nil), manifest...)
	mutated[len(mutated)-1] ^= 1
	if err := VerifyManifestSignature(publicKey, mutated, envelope); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("mutated manifest error = %v", err)
	}
	for _, invalid := range [][]byte{
		[]byte(`{"algorithm":"rsa","keyId":"` + keyID + `","signature":"x"}`),
		[]byte(`{"algorithm":"ed25519","keyId":"wrong","signature":"x"}`),
		[]byte(`{"algorithm":"ed25519","keyId":"` + keyID + `","signature":"x","extra":true}`),
	} {
		if err := VerifyManifestSignature(publicKey, manifest, invalid); !errors.Is(err, ErrInvalidSignature) {
			t.Errorf("invalid envelope error = %v", err)
		}
	}
}

func TestPublicKeyParsingAndEmbeddedConfiguration(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	for _, encoded := range []string{base64.StdEncoding.EncodeToString(publicKey), hex.EncodeToString(publicKey)} {
		parsed, err := ParsePublicKey(encoded)
		if err != nil || !publicKey.Equal(parsed) {
			t.Fatalf("ParsePublicKey: equal=%v err=%v", publicKey.Equal(parsed), err)
		}
	}
	old := PublicKeyBase64
	t.Cleanup(func() { PublicKeyBase64 = old })
	PublicKeyBase64 = ""
	if _, err := EmbeddedPublicKey(); err == nil {
		t.Fatal("empty embedded key was accepted")
	}
	PublicKeyBase64 = base64.StdEncoding.EncodeToString(publicKey)
	if _, err := EmbeddedPublicKey(); err != nil {
		t.Fatal(err)
	}
}

func validManifestJSON(version, hash string, size int64) []byte {
	return []byte(fmt.Sprintf(`{"schemaVersion":1,"version":%q,"publishedAt":%q,"releaseUrl":"https://github.com/submerge/submerge/releases/tag/v%s","assets":[%s,%s]}`,
		version, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC).Format(time.RFC3339), version,
		validAssetJSON("amd64", hash, size), validAssetJSON("arm64", hash, size)))
}

func validAssetJSON(arch, hash string, size int64) string {
	return fmt.Sprintf(`{"os":"linux","arch":%q,"name":"submerge-linux-%s","url":"https://github.com/submerge/submerge/releases/download/v1.2.0/submerge-linux-%s","sha256":%q,"size":%d}`,
		arch, arch, arch, hash, size)
}
