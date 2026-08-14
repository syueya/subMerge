package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestDownloaderStreamsVerifiesAndReportsProgress(t *testing.T) {
	body := []byte("release-binary")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "14")
		_, _ = w.Write(body[:5])
		_, _ = w.Write(body[5:])
	}))
	defer server.Close()
	sum := sha256.Sum256(body)
	asset := Asset{Name: "submerge-linux-amd64", URL: "https://updates.test/binary", SHA256: hex.EncodeToString(sum[:]), Size: int64(len(body))}
	var progress []Progress
	result, err := (Downloader{HTTPClient: rewriteClient(t, server.URL)}).Download(context.Background(), asset, t.TempDir(), func(item Progress) {
		progress = append(progress, item)
	})
	if err != nil {
		t.Fatal(err)
	}
	downloaded, err := os.ReadFile(result.Path)
	if err != nil || string(downloaded) != string(body) {
		t.Fatalf("downloaded=%q err=%v", downloaded, err)
	}
	if len(progress) < 2 || progress[0].Downloaded != 0 || progress[len(progress)-1].Downloaded != int64(len(body)) {
		t.Fatalf("progress = %+v", progress)
	}
}

func TestDownloaderRemovesInvalidDownloads(t *testing.T) {
	body := []byte("release-binary")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(body) }))
	defer server.Close()
	tests := []Asset{
		{Name: "submerge", URL: "https://updates.test/binary", SHA256: strings.Repeat("0", 64), Size: int64(len(body))},
		{Name: "submerge", URL: "https://updates.test/binary", SHA256: strings.Repeat("a", 64), Size: int64(len(body) + 1)},
	}
	for _, asset := range tests {
		dir := t.TempDir()
		_, err := (Downloader{HTTPClient: rewriteClient(t, server.URL)}).Download(context.Background(), asset, dir, nil)
		if err == nil {
			t.Fatal("invalid download succeeded")
		}
		entries, readErr := os.ReadDir(dir)
		if readErr != nil || len(entries) != 0 {
			t.Fatalf("failed download left files: entries=%v err=%v", entries, readErr)
		}
	}
}

func TestDownloaderRejectsLimitsAndCancellation(t *testing.T) {
	asset := Asset{Name: "submerge", URL: "https://updates.test/binary", SHA256: strings.Repeat("a", 64), Size: 20}
	if _, err := (Downloader{MaxBytes: 10}).Download(context.Background(), asset, t.TempDir(), nil); err == nil {
		t.Fatal("oversized asset was accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := (Downloader{}).Download(ctx, asset, t.TempDir(), nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel error = %v", err)
	}
}
