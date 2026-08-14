package updater

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
)

const (
	DefaultManifestName     = "update-manifest.json"
	DefaultMaxManifestBytes = 1 << 20
	DefaultMaxSignatureSize = 4 << 10
	DefaultUserAgent        = "submerge-updater/1"
)

type ReleaseSource struct {
	ManifestURL     string
	SignatureURL    string
	PublicKey       ed25519.PublicKey
	HTTPClient      *http.Client
	UserAgent       string
	MaxManifestSize int64
}

type CheckResult struct {
	CurrentVersion string   `json:"currentVersion"`
	LatestVersion  string   `json:"latestVersion"`
	Available      bool     `json:"available"`
	Manifest       Manifest `json:"manifest"`
	Asset          Asset    `json:"asset"`
}

// GitHubLatestManifestURL returns the stable public URL for a manifest
// attached to the latest GitHub Release.
func GitHubLatestManifestURL(owner, repository string) (string, error) {
	if !validGitHubPart(owner) || !validGitHubPart(repository) {
		return "", errors.New("invalid GitHub owner or repository")
	}
	return fmt.Sprintf("https://github.com/%s/%s/releases/latest/download/%s", owner, repository, DefaultManifestName), nil
}

func validGitHubPart(raw string) bool {
	if raw == "" || raw == "." || raw == ".." {
		return false
	}
	for _, ch := range raw {
		if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') && ch != '-' && ch != '_' && ch != '.' {
			return false
		}
	}
	return true
}

func (s ReleaseSource) Check(ctx context.Context, currentVersion string) (CheckResult, error) {
	return s.CheckPlatform(ctx, currentVersion, runtime.GOOS, runtime.GOARCH)
}

func (s ReleaseSource) CheckPlatform(ctx context.Context, currentVersion, goos, goarch string) (CheckResult, error) {
	if _, err := ParseVersion(currentVersion); err != nil {
		return CheckResult{}, fmt.Errorf("current version: %w", err)
	}
	if len(s.PublicKey) != ed25519.PublicKeySize {
		return CheckResult{}, errors.New("update public key is not configured")
	}
	if err := validateHTTPSURL(s.ManifestURL); err != nil {
		return CheckResult{}, fmt.Errorf("manifest URL: %w", err)
	}
	sigURL := strings.TrimSpace(s.SignatureURL)
	if sigURL == "" {
		sigURL = s.ManifestURL + ".sig"
	}
	if err := validateHTTPSURL(sigURL); err != nil {
		return CheckResult{}, fmt.Errorf("signature URL: %w", err)
	}

	manifestRaw, err := s.fetch(ctx, s.ManifestURL, s.maxManifestSize())
	if err != nil {
		return CheckResult{}, fmt.Errorf("fetch update manifest: %w", err)
	}
	signature, err := s.fetch(ctx, sigURL, DefaultMaxSignatureSize)
	if err != nil {
		return CheckResult{}, fmt.Errorf("fetch update signature: %w", err)
	}
	if err := VerifyManifestSignature(s.PublicKey, manifestRaw, signature); err != nil {
		return CheckResult{}, err
	}
	manifest, err := ParseManifest(manifestRaw)
	if err != nil {
		return CheckResult{}, err
	}
	asset, err := manifest.SelectAsset(goos, goarch)
	if err != nil {
		return CheckResult{}, err
	}
	comparison, err := CompareVersions(manifest.Version, currentVersion)
	if err != nil {
		return CheckResult{}, err
	}
	return CheckResult{
		CurrentVersion: currentVersion,
		LatestVersion:  manifest.Version,
		Available:      comparison > 0,
		Manifest:       manifest,
		Asset:          asset,
	}, nil
}

func (s ReleaseSource) fetch(ctx context.Context, rawURL string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/octet-stream, application/json")
	req.Header.Set("User-Agent", s.userAgent())
	client := s.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := secureHTTPClient(client).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("upstream status %d", resp.StatusCode)
	}
	if resp.ContentLength > limit {
		return nil, fmt.Errorf("response exceeds %d bytes", limit)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return nil, errors.New("empty response")
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("response exceeds %d bytes", limit)
	}
	return body, nil
}

// NewReleaseSource creates a source using the trusted public key injected by
// the release build. It is the production constructor used by API integration.
func NewReleaseSource(manifestURL string, client *http.Client) (ReleaseSource, error) {
	publicKey, err := EmbeddedPublicKey()
	if err != nil {
		return ReleaseSource{}, err
	}
	return ReleaseSource{ManifestURL: manifestURL, PublicKey: publicKey, HTTPClient: client}, nil
}

func UpdatesEnabled() bool {
	_, err := EmbeddedPublicKey()
	return err == nil
}

func secureHTTPClient(client *http.Client) *http.Client {
	clone := *client
	previous := clone.CheckRedirect
	clone.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if err := validateHTTPSURL(req.URL.String()); err != nil {
			return fmt.Errorf("insecure update redirect: %w", err)
		}
		if previous != nil {
			return previous(req, via)
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}
	return &clone
}

func (s ReleaseSource) maxManifestSize() int64 {
	if s.MaxManifestSize > 0 {
		return s.MaxManifestSize
	}
	return DefaultMaxManifestBytes
}

func (s ReleaseSource) userAgent() string {
	if strings.TrimSpace(s.UserAgent) != "" {
		return s.UserAgent
	}
	return DefaultUserAgent
}
