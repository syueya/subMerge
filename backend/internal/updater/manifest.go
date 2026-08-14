package updater

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

const ManifestSchemaVersion = 1

var (
	ErrInvalidSignature = errors.New("update manifest signature is invalid")
	ErrNoAsset          = errors.New("update has no asset for this platform")
)

// PublicKeyBase64 is injected by the release build with:
//
//	-X github.com/submerge/submerge/backend/internal/updater.PublicKeyBase64=...
//
// Empty development builds deliberately cannot check or install updates.
var PublicKeyBase64 string

// Manifest is the signed update metadata published as a GitHub Release asset.
// The detached Ed25519 signature covers the exact update-manifest.json bytes.
type Manifest struct {
	SchemaVersion int       `json:"schemaVersion"`
	Version       string    `json:"version"`
	PublishedAt   time.Time `json:"publishedAt"`
	ReleaseURL    string    `json:"releaseUrl,omitempty"`
	Notes         string    `json:"notes,omitempty"`
	Assets        []Asset   `json:"assets"`
}

type Asset struct {
	OS     string `json:"os"`
	Arch   string `json:"arch"`
	Name   string `json:"name"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type SignatureEnvelope struct {
	Algorithm string `json:"algorithm"`
	KeyID     string `json:"keyId"`
	Signature string `json:"signature"`
}

func ParseManifest(raw []byte) (Manifest, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var manifest Manifest
	if err := dec.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode update manifest: %w", err)
	}
	if err := ensureJSONEOF(dec); err != nil {
		return Manifest{}, err
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func ensureJSONEOF(dec *json.Decoder) error {
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("decode update manifest: trailing JSON value")
		}
		return fmt.Errorf("decode update manifest: %w", err)
	}
	return nil
}

func (m Manifest) Validate() error {
	if m.SchemaVersion != ManifestSchemaVersion {
		return fmt.Errorf("unsupported update manifest schema %d", m.SchemaVersion)
	}
	if _, err := ParseVersion(m.Version); err != nil {
		return fmt.Errorf("manifest version: %w", err)
	}
	if m.PublishedAt.IsZero() {
		return errors.New("manifest publishedAt is required")
	}
	if m.ReleaseURL != "" {
		if err := validateHTTPSURL(m.ReleaseURL); err != nil {
			return fmt.Errorf("manifest releaseUrl: %w", err)
		}
	}
	if len(m.Assets) == 0 {
		return errors.New("manifest assets must not be empty")
	}
	seen := make(map[string]struct{}, len(m.Assets))
	for i := range m.Assets {
		asset := &m.Assets[i]
		if asset.OS != strings.TrimSpace(asset.OS) || asset.Arch != strings.TrimSpace(asset.Arch) || asset.Name != strings.TrimSpace(asset.Name) || asset.SHA256 != strings.TrimSpace(asset.SHA256) {
			return fmt.Errorf("manifest asset %d contains surrounding whitespace", i)
		}
		if asset.OS == "" || asset.Arch == "" {
			return fmt.Errorf("manifest asset %d must specify os and arch", i)
		}
		if asset.Name == "" || filepath.Base(asset.Name) != asset.Name || asset.Name == "." || asset.Name == ".." {
			return fmt.Errorf("manifest asset %d has an invalid name", i)
		}
		if err := validateHTTPSURL(asset.URL); err != nil {
			return fmt.Errorf("manifest asset %d URL: %w", i, err)
		}
		hash, err := hex.DecodeString(asset.SHA256)
		if err != nil || len(hash) != 32 {
			return fmt.Errorf("manifest asset %d has an invalid SHA-256", i)
		}
		if asset.Size <= 0 {
			return fmt.Errorf("manifest asset %d size must be positive", i)
		}
		key := asset.OS + "/" + asset.Arch
		if _, exists := seen[key]; exists {
			return fmt.Errorf("manifest has duplicate asset for %s", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateHTTPSURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" {
		return errors.New("must be an HTTPS URL without credentials or fragment")
	}
	return nil
}

// SelectAsset returns the unique asset matching a Go operating system and
// architecture pair.
func (m Manifest) SelectAsset(goos, goarch string) (Asset, error) {
	for _, asset := range m.Assets {
		if asset.OS == goos && asset.Arch == goarch {
			return asset, nil
		}
	}
	return Asset{}, fmt.Errorf("%w: %s/%s", ErrNoAsset, goos, goarch)
}

// ParsePublicKey accepts an Ed25519 public key encoded as hex or base64.
func ParsePublicKey(raw string) (ed25519.PublicKey, error) {
	decoded, err := decodeTextBytes(raw, ed25519.PublicKeySize)
	if err != nil {
		return nil, fmt.Errorf("decode update public key: %w", err)
	}
	return ed25519.PublicKey(decoded), nil
}

// EmbeddedPublicKey returns the release key injected at link time. It fails
// closed in local/development builds where no trusted key was injected.
func EmbeddedPublicKey() (ed25519.PublicKey, error) {
	if strings.TrimSpace(PublicKeyBase64) == "" {
		return nil, errors.New("online updates are disabled: no release public key is embedded")
	}
	return ParsePublicKey(PublicKeyBase64)
}

func PublicKeyID(publicKey ed25519.PublicKey) (string, error) {
	if len(publicKey) != ed25519.PublicKeySize {
		return "", errors.New("update public key has an invalid length")
	}
	sum := sha256.Sum256(publicKey)
	return hex.EncodeToString(sum[:])[:16], nil
}

// VerifyManifestSignature verifies the release signature envelope. The
// signature covers the exact downloaded manifest bytes.
func VerifyManifestSignature(publicKey ed25519.PublicKey, manifest, signatureJSON []byte) error {
	if len(publicKey) != ed25519.PublicKeySize {
		return errors.New("update public key has an invalid length")
	}
	dec := json.NewDecoder(bytes.NewReader(signatureJSON))
	dec.DisallowUnknownFields()
	var envelope SignatureEnvelope
	if err := dec.Decode(&envelope); err != nil {
		return fmt.Errorf("%w: decode signature envelope: %v", ErrInvalidSignature, err)
	}
	if err := ensureJSONEOF(dec); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidSignature, err)
	}
	if envelope.Algorithm != "ed25519" {
		return fmt.Errorf("%w: unsupported algorithm %q", ErrInvalidSignature, envelope.Algorithm)
	}
	keyID, _ := PublicKeyID(publicKey)
	if envelope.KeyID != keyID {
		return fmt.Errorf("%w: key ID does not match embedded key", ErrInvalidSignature)
	}
	decoded, err := base64.StdEncoding.DecodeString(envelope.Signature)
	if err != nil || len(decoded) != ed25519.SignatureSize {
		return fmt.Errorf("%w: signature must be 64 bytes encoded as base64", ErrInvalidSignature)
	}
	if !ed25519.Verify(publicKey, manifest, decoded) {
		return ErrInvalidSignature
	}
	return nil
}

func decodeTextBytes(raw string, size int) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	decoders := []func(string) ([]byte, error){
		hex.DecodeString,
		base64.StdEncoding.DecodeString,
		base64.RawStdEncoding.DecodeString,
		base64.URLEncoding.DecodeString,
		base64.RawURLEncoding.DecodeString,
	}
	for _, decode := range decoders {
		value, err := decode(raw)
		if err == nil && len(value) == size {
			return value, nil
		}
	}
	return nil, fmt.Errorf("expected %d encoded bytes", size)
}
