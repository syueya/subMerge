package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const DefaultMaxAssetBytes int64 = 512 << 20

var (
	ErrChecksumMismatch = errors.New("downloaded update checksum does not match manifest")
	ErrSizeMismatch     = errors.New("downloaded update size does not match manifest")
)

type Progress struct {
	Downloaded int64 `json:"downloaded"`
	Total      int64 `json:"total"`
}

type ProgressFunc func(Progress)

type DownloadResult struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type Downloader struct {
	HTTPClient *http.Client
	UserAgent  string
	MaxBytes   int64
}

// Download streams a signed-manifest asset to a unique staging file. The
// incomplete file is removed on every failure, including context cancellation.
func (d Downloader) Download(ctx context.Context, asset Asset, stagingDir string, progress ProgressFunc) (DownloadResult, error) {
	if err := validateDownloadAsset(asset); err != nil {
		return DownloadResult{}, err
	}
	maxBytes := d.MaxBytes
	if maxBytes <= 0 {
		maxBytes = DefaultMaxAssetBytes
	}
	if asset.Size > maxBytes {
		return DownloadResult{}, fmt.Errorf("update asset size %d exceeds limit %d", asset.Size, maxBytes)
	}
	if err := os.MkdirAll(stagingDir, 0o700); err != nil {
		return DownloadResult{}, fmt.Errorf("create update staging directory: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.URL, nil)
	if err != nil {
		return DownloadResult{}, err
	}
	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("Accept-Encoding", "identity")
	userAgent := strings.TrimSpace(d.UserAgent)
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}
	req.Header.Set("User-Agent", userAgent)
	client := d.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := secureHTTPClient(client).Do(req)
	if err != nil {
		return DownloadResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return DownloadResult{}, fmt.Errorf("download update: upstream status %d", resp.StatusCode)
	}
	if resp.ContentLength >= 0 && resp.ContentLength != asset.Size {
		return DownloadResult{}, fmt.Errorf("%w: got content length %d, want %d", ErrSizeMismatch, resp.ContentLength, asset.Size)
	}

	file, err := os.CreateTemp(stagingDir, "."+asset.Name+"-*.download")
	if err != nil {
		return DownloadResult{}, fmt.Errorf("create update staging file: %w", err)
	}
	path := file.Name()
	ok := false
	defer func() {
		_ = file.Close()
		if !ok {
			_ = os.Remove(path)
		}
	}()

	hash := sha256.New()
	reader := io.LimitReader(resp.Body, asset.Size+1)
	counter := &progressWriter{total: asset.Size, callback: progress}
	if progress != nil {
		progress(Progress{Total: asset.Size})
	}
	written, err := io.Copy(io.MultiWriter(file, hash, counter), reader)
	if err != nil {
		return DownloadResult{}, fmt.Errorf("download update: %w", err)
	}
	if written != asset.Size {
		return DownloadResult{}, fmt.Errorf("%w: got %d, want %d", ErrSizeMismatch, written, asset.Size)
	}
	actualHash := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actualHash, asset.SHA256) {
		return DownloadResult{}, fmt.Errorf("%w: got %s", ErrChecksumMismatch, actualHash)
	}
	if err := file.Sync(); err != nil {
		return DownloadResult{}, fmt.Errorf("sync update staging file: %w", err)
	}
	if err := file.Close(); err != nil {
		return DownloadResult{}, fmt.Errorf("close update staging file: %w", err)
	}
	ok = true
	return DownloadResult{Path: filepath.Clean(path), Size: written, SHA256: actualHash}, nil
}

func validateDownloadAsset(asset Asset) error {
	if asset.Name == "" || filepath.Base(asset.Name) != asset.Name || asset.Name == "." || asset.Name == ".." {
		return errors.New("update asset has an invalid name")
	}
	if err := validateHTTPSURL(asset.URL); err != nil {
		return fmt.Errorf("update asset URL: %w", err)
	}
	decoded, err := hex.DecodeString(strings.TrimSpace(asset.SHA256))
	if err != nil || len(decoded) != sha256.Size {
		return errors.New("update asset has an invalid SHA-256")
	}
	if asset.Size <= 0 {
		return errors.New("update asset size must be positive")
	}
	return nil
}

type progressWriter struct {
	downloaded int64
	total      int64
	callback   ProgressFunc
}

func (w *progressWriter) Write(p []byte) (int, error) {
	w.downloaded += int64(len(p))
	if w.callback != nil {
		w.callback(Progress{Downloaded: w.downloaded, Total: w.total})
	}
	return len(p), nil
}
