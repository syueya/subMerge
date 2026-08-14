// Command release-tool creates and signs the update metadata published with a release.
package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/submerge/submerge/backend/internal/updater"
)

const privateKeyEnv = "UPDATE_SIGNING_PRIVATE_KEY"

type manifest struct {
	SchemaVersion int             `json:"schemaVersion"`
	Version       string          `json:"version"`
	PublishedAt   string          `json:"publishedAt"`
	ReleaseURL    string          `json:"releaseUrl"`
	Assets        []manifestAsset `json:"assets"`
}

type manifestAsset struct {
	OS     string `json:"os"`
	Arch   string `json:"arch"`
	Name   string `json:"name"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type signatureFile struct {
	Algorithm string `json:"algorithm"`
	KeyID     string `json:"keyId"`
	Signature string `json:"signature"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "release-tool:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("expected public-key or manifest command")
	}
	privateKey, err := privateKeyFromEnv()
	if err != nil {
		return err
	}

	switch args[0] {
	case "public-key":
		if len(args) != 1 {
			return errors.New("public-key does not accept arguments")
		}
		_, err = fmt.Fprintln(stdout, base64.StdEncoding.EncodeToString(privateKey.Public().(ed25519.PublicKey)))
		return err
	case "manifest":
		return runManifest(args[1:], privateKey, stdout)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runManifest(args []string, privateKey ed25519.PrivateKey, stdout io.Writer) error {
	flags := flag.NewFlagSet("manifest", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	assetsDir := flags.String("assets", "", "directory containing release binaries")
	repository := flags.String("repository", "", "GitHub owner/repository")
	version := flags.String("version", "", "release version without v prefix")
	publishedAt := flags.String("published-at", "", "RFC3339 publication time")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *assetsDir == "" || *repository == "" || strings.HasPrefix(*version, "v") {
		return errors.New("manifest requires --assets, --repository, and a SemVer --version without v prefix")
	}
	if _, err := updater.ParseVersion(*version); err != nil {
		return fmt.Errorf("invalid --version: %w", err)
	}
	publicationTime := time.Now().UTC()
	if *publishedAt != "" {
		parsed, err := time.Parse(time.RFC3339, *publishedAt)
		if err != nil {
			return fmt.Errorf("parse --published-at: %w", err)
		}
		publicationTime = parsed.UTC()
	}

	m := manifest{
		SchemaVersion: 1,
		Version:       *version,
		PublishedAt:   publicationTime.Format(time.RFC3339),
		ReleaseURL:    fmt.Sprintf("https://github.com/%s/releases/tag/v%s", *repository, *version),
	}
	for _, arch := range []string{"amd64", "arm64"} {
		name := "submerge-linux-" + arch
		path := filepath.Join(*assetsDir, name)
		asset, err := hashAsset(path, *repository, *version, name, arch)
		if err != nil {
			return err
		}
		m.Assets = append(m.Assets, asset)
	}

	manifestBytes, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	manifestBytes = append(manifestBytes, '\n')
	if _, err := updater.ParseManifest(manifestBytes); err != nil {
		return fmt.Errorf("generated manifest violates updater contract: %w", err)
	}
	publicKey := privateKey.Public().(ed25519.PublicKey)
	keyHash := sha256.Sum256(publicKey)
	sig := signatureFile{
		Algorithm: "ed25519",
		KeyID:     hex.EncodeToString(keyHash[:8]),
		Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, manifestBytes)),
	}
	signatureBytes, err := json.MarshalIndent(sig, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal signature: %w", err)
	}
	signatureBytes = append(signatureBytes, '\n')

	manifestPath := filepath.Join(*assetsDir, "update-manifest.json")
	if err := os.WriteFile(manifestPath, manifestBytes, 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	if err := os.WriteFile(manifestPath+".sig", signatureBytes, 0o644); err != nil {
		return fmt.Errorf("write signature: %w", err)
	}
	_, err = fmt.Fprintf(stdout, "wrote %s and %s\n", manifestPath, manifestPath+".sig")
	return err
}

func hashAsset(path, repository, version, name, arch string) (manifestAsset, error) {
	f, err := os.Open(path)
	if err != nil {
		return manifestAsset{}, fmt.Errorf("open %s: %w", name, err)
	}
	defer f.Close()
	h := sha256.New()
	size, err := io.Copy(h, f)
	if err != nil {
		return manifestAsset{}, fmt.Errorf("hash %s: %w", name, err)
	}
	return manifestAsset{
		OS:     "linux",
		Arch:   arch,
		Name:   name,
		URL:    fmt.Sprintf("https://github.com/%s/releases/download/v%s/%s", repository, version, name),
		SHA256: hex.EncodeToString(h.Sum(nil)),
		Size:   size,
	}, nil
}

func privateKeyFromEnv() (ed25519.PrivateKey, error) {
	raw := strings.TrimSpace(os.Getenv(privateKeyEnv))
	if raw == "" {
		return nil, fmt.Errorf("%s is required", privateKeyEnv)
	}
	if block, _ := pem.Decode([]byte(raw)); block != nil {
		parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse %s PKCS#8 PEM: %w", privateKeyEnv, err)
		}
		key, ok := parsed.(ed25519.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("%s PEM is not an Ed25519 private key", privateKeyEnv)
		}
		return key, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("%s must be PKCS#8 PEM or base64 raw Ed25519 key: %w", privateKeyEnv, err)
	}
	switch len(decoded) {
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(decoded), nil
	case ed25519.PrivateKeySize:
		derived := ed25519.NewKeyFromSeed(decoded[:ed25519.SeedSize])
		if subtle.ConstantTimeCompare(derived, decoded) != 1 {
			return nil, fmt.Errorf("%s private key public half does not match its seed", privateKeyEnv)
		}
		return derived, nil
	default:
		return nil, fmt.Errorf("%s decoded length is %d, want 32-byte seed or 64-byte private key", privateKeyEnv, len(decoded))
	}
}
