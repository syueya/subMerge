package updater

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestReleaseSourceCheckVerifiesAndSelectsPlatform(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	manifest := validManifestJSON("1.2.0", strings.Repeat("a", 64), 12)
	keyID, _ := PublicKeyID(publicKey)
	envelope := []byte(fmt.Sprintf(`{"algorithm":"ed25519","keyId":"%s","signature":"%s"}`, keyID, base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, manifest))))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.UserAgent() != DefaultUserAgent {
			t.Errorf("User-Agent = %q", r.UserAgent())
		}
		switch r.URL.Path {
		case "/manifest":
			_, _ = w.Write(manifest)
		case "/manifest.sig":
			_, _ = w.Write(envelope)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	source := ReleaseSource{
		ManifestURL: "https://updates.test/manifest",
		PublicKey:   publicKey,
		HTTPClient:  rewriteClient(t, server.URL),
	}
	result, err := source.CheckPlatform(context.Background(), "1.1.9", "linux", "arm64")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Available || result.LatestVersion != "1.2.0" || result.Asset.Arch != "arm64" {
		t.Fatalf("check result = %+v", result)
	}
	result, err = source.CheckPlatform(context.Background(), "1.3.0", "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if result.Available {
		t.Fatal("older release was reported as available")
	}
}

func TestReleaseSourceFailsClosed(t *testing.T) {
	if _, err := (ReleaseSource{ManifestURL: "https://updates.test/manifest"}).CheckPlatform(context.Background(), "1.0.0", "linux", "amd64"); err == nil {
		t.Fatal("missing key was accepted")
	}
	if _, err := GitHubLatestManifestURL("bad/owner", "repo"); err == nil {
		t.Fatal("invalid GitHub path was accepted")
	}
	url, err := GitHubLatestManifestURL("submerge", "submerge")
	if err != nil || url != "https://github.com/submerge/submerge/releases/latest/download/update-manifest.json" {
		t.Fatalf("latest URL = %q, err=%v", url, err)
	}
}

func TestReleaseSourceRejectsOversizedOrBadStatus(t *testing.T) {
	publicKey, _, _ := ed25519.GenerateKey(rand.Reader)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/status" {
			http.Error(w, "no", http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte("too large"))
	}))
	defer server.Close()
	for _, path := range []string{"status", "large"} {
		source := ReleaseSource{
			ManifestURL:     "https://updates.test/" + path,
			PublicKey:       publicKey,
			HTTPClient:      rewriteClient(t, server.URL),
			MaxManifestSize: 2,
		}
		if _, err := source.CheckPlatform(context.Background(), "1.0.0", "linux", "amd64"); err == nil {
			t.Errorf("%s response was accepted", path)
		}
	}
}

func rewriteClient(t *testing.T, serverURL string) *http.Client {
	t.Helper()
	target, err := url.Parse(serverURL)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		clone := req.Clone(req.Context())
		clone.URL.Scheme = target.Scheme
		clone.URL.Host = target.Host
		return http.DefaultTransport.RoundTrip(clone)
	})}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }
