package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManifestIsSignedOverExactBytes(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(privateKeyEnv, base64.StdEncoding.EncodeToString(privateKey))
	dir := t.TempDir()
	for _, arch := range []string{"amd64", "arm64"} {
		if err := os.WriteFile(filepath.Join(dir, "submerge-linux-"+arch), []byte("binary-"+arch), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	if err := run([]string{"manifest", "--assets", dir, "--repository", "acme/submerge", "--version", "1.2.3", "--published-at", "2026-08-13T01:02:03Z"}, os.Stdout); err != nil {
		t.Fatal(err)
	}
	manifestBytes, err := os.ReadFile(filepath.Join(dir, "update-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got manifest
	if err := json.Unmarshal(manifestBytes, &got); err != nil {
		t.Fatal(err)
	}
	if got.Version != "1.2.3" || got.PublishedAt != "2026-08-13T01:02:03Z" || len(got.Assets) != 2 {
		t.Fatalf("unexpected manifest: %+v", got)
	}
	wantHash := sha256.Sum256([]byte("binary-amd64"))
	if got.Assets[0].SHA256 != hex.EncodeToString(wantHash[:]) || got.Assets[0].Size != int64(len("binary-amd64")) {
		t.Fatalf("unexpected amd64 asset: %+v", got.Assets[0])
	}

	signatureBytes, err := os.ReadFile(filepath.Join(dir, "update-manifest.json.sig"))
	if err != nil {
		t.Fatal(err)
	}
	var sig signatureFile
	if err := json.Unmarshal(signatureBytes, &sig); err != nil {
		t.Fatal(err)
	}
	decodedSignature, err := base64.StdEncoding.DecodeString(sig.Signature)
	if err != nil {
		t.Fatal(err)
	}
	if !ed25519.Verify(publicKey, manifestBytes, decodedSignature) {
		t.Fatal("signature does not verify over the exact manifest bytes")
	}
}

func TestPublicKeyAndMissingSecret(t *testing.T) {
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i)
	}
	t.Setenv(privateKeyEnv, base64.StdEncoding.EncodeToString(seed))
	var out strings.Builder
	if err := run([]string{"public-key"}, &out); err != nil {
		t.Fatal(err)
	}
	want := base64.StdEncoding.EncodeToString(ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey))
	if strings.TrimSpace(out.String()) != want {
		t.Fatalf("public key = %q, want %q", strings.TrimSpace(out.String()), want)
	}

	t.Setenv(privateKeyEnv, "")
	if err := run([]string{"public-key"}, &out); err == nil {
		t.Fatal("expected missing signing secret to fail")
	}

	malformed := ed25519.NewKeyFromSeed(seed)
	malformed[len(malformed)-1] ^= 0xff
	t.Setenv(privateKeyEnv, base64.StdEncoding.EncodeToString(malformed))
	if err := run([]string{"public-key"}, &out); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("malformed 64-byte private key error = %v", err)
	}
}

func TestManifestRejectsInvalidVersionBeforeReadingAssets(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(privateKeyEnv, base64.StdEncoding.EncodeToString(privateKey))
	for _, version := range []string{"v1.2.3", "01.2.3", "1.2.3-01"} {
		err := run([]string{"manifest", "--assets", t.TempDir(), "--repository", "acme/submerge", "--version", version}, os.Stdout)
		if err == nil {
			t.Fatalf("version %q unexpectedly accepted", version)
		}
	}
}

func TestManifestRejectsRepositoryThatCreatesInvalidURLs(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(privateKeyEnv, base64.StdEncoding.EncodeToString(privateKey))
	dir := t.TempDir()
	for _, arch := range []string{"amd64", "arm64"} {
		if err := os.WriteFile(filepath.Join(dir, "submerge-linux-"+arch), []byte(arch), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	err = run([]string{"manifest", "--assets", dir, "--repository", "acme/submerge#fragment", "--version", "1.2.3"}, os.Stdout)
	if err == nil || !strings.Contains(err.Error(), "updater contract") {
		t.Fatalf("unexpected error: %v", err)
	}
}
