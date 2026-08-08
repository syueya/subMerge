package geo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/oschwald/maxminddb-golang"

		"github.com/submerge/submerge/backend/internal/applog"
)

func (s *Service) Update(ctx context.Context) UpdateResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := UpdateResponse{Items: []UpdateItem{}}
	entries := []struct{ name, rawURL string }{
		{GeoIPFile, s.urls.GeoIP}, {GeoSiteFile, s.urls.GeoSite}, {MetaDBFile, s.urls.MetaDB}, {ASNFile, s.urls.ASN},
	}
	for _, item := range entries {
		body, err := s.download(ctx, item.rawURL)
		if err == nil {
			err = validateResource(item.name, body)
		}
		if err == nil {
			err = replaceResource(s.dir, item.name, body)
		}
out := UpdateItem{Name: item.name, Updated: err == nil}
			if err != nil {
				out.Error = err.Error()
				applog.Error("[geo] %s 更新失败: %v", item.name, err)
			}
		result.Items = append(result.Items, out)
	}
	s.snap = s.loadSnapshot()
	ok := 0
	for _, item := range result.Items {
		if item.Updated {
			ok++
		}
	}
	applog.Info("[geo] 更新完成 成功=%d/%d", ok, len(result.Items))
	return result
}

func (s *Service) download(ctx context.Context, raw string) ([]byte, error) {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return nil, errors.New("invalid geo download URL")
	}
	if err = validateURL(u); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "submerge-geo-updater/1")
	s.clientMu.RLock()
	client := s.client
	s.clientMu.RUnlock()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("upstream status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDownloadBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) == 0 || len(body) > maxDownloadBytes {
		return nil, errors.New("geo response size is invalid")
	}
	return body, nil
}

func validateResource(name string, body []byte) error {
	if len(body) == 0 {
		return errors.New("empty geo file")
	}
	switch name {
	case GeoSiteFile:
		_, err := parseGeoSite(body)
		return err
	case GeoIPFile:
		_, err := parseGeoIP(body)
		return err
	case MetaDBFile, ASNFile:
		_, err := maxminddb.FromBytes(body)
		if err != nil {
			return fmt.Errorf("invalid mmdb: %w", err)
		}
	}
	return nil
}

func replaceResource(dir, name string, body []byte) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+name+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err = tmp.Write(body); err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return atomicReplace(tmpName, filepath.Join(dir, name))
}

func atomicReplace(tmp, target string) error {
	backup := target + ".backup"
	_ = os.Remove(backup)
	if err := os.Rename(target, backup); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Rename(backup, target)
		return err
	}
	_ = os.Remove(backup)
	return nil
}

func validateURL(u *url.URL) error {
	if u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return errors.New("unsupported geo URL")
	}
	ips, err := net.LookupIP(u.Hostname())
	if err != nil {
		return err
	}
	for _, ip := range ips {
		if blockedIP(ip) {
			return errors.New("geo URL resolves to a blocked address")
		}
	}
	return nil
}

func blockedIP(ip net.IP) bool {
	return ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified()
}
