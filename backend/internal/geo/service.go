package geo

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/oschwald/maxminddb-golang"
	"github.com/submerge/submerge/backend/internal/ipgeo"
	"github.com/submerge/submerge/backend/internal/outbound"
)

const (
	GeoIPFile         = "geoip.dat"
	GeoSiteFile       = "geosite.dat"
	MetaDBFile        = "geoip.metadb"
	ASNFile           = "GeoLite2-ASN.mmdb"
	maxDownloadBytes  = 64 << 20
	maxDomainLength   = 253
	maxCategoryLength = 128
)

var resourceNames = []string{GeoIPFile, GeoSiteFile, MetaDBFile, ASNFile}

type URLs struct {
	GeoIP   string
	GeoSite string
	MetaDB  string
	ASN     string
}

func DefaultURLs() URLs {
	return URLs{
		GeoIP:   "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geoip.dat",
		GeoSite: "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geosite.dat",
		MetaDB:  "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geoip.metadb",
		ASN:     "https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/GeoLite2-ASN.mmdb",
	}
}

type domainRecord struct {
	Category string
	Type     string
	Value    string
}

type cidrRecord struct {
	Category string
	Network  *net.IPNet
}

type searchRecord struct {
	File         string
	ASN          uint32
	Organization string
	Network      *net.IPNet
	Codes        []string
	Detail       string
}

type resourceState struct {
	status ResourceStatus
	body   []byte
}

type snapshot struct {
	resources  map[string]resourceState
	domains    []domainRecord
	cidrs      []cidrRecord
	categories map[string][]Category
	meta       *maxminddb.Reader
	asn        *maxminddb.Reader
	search     []searchRecord
}

// Service owns one consistent in-memory view of all Geo resources.
type Service struct {
	dir      string
	urls     URLs
	mu       sync.RWMutex
	snap     snapshot
	client   *http.Client
	clientMu sync.RWMutex
	ipGeoMu  sync.RWMutex
	ipGeo    *ipgeo.Client
}

func NewService(dir string, urls URLs) *Service {
	defaults := DefaultURLs()
	if urls.GeoIP == "" {
		urls.GeoIP = defaults.GeoIP
	}
	if urls.GeoSite == "" {
		urls.GeoSite = defaults.GeoSite
	}
	if urls.MetaDB == "" {
		urls.MetaDB = defaults.MetaDB
	}
	if urls.ASN == "" {
		urls.ASN = defaults.ASN
	}
	return &Service{
		dir:  dir,
		urls: urls,
		client: &http.Client{
			Timeout: 2 * time.Minute,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return errors.New("too many redirects")
				}
				return validateURL(req.URL)
			},
		},
	}
}

func (s *Service) SetIPGeoClient(client *ipgeo.Client) {
	s.ipGeoMu.Lock()
	s.ipGeo = client
	s.ipGeoMu.Unlock()
}

func (s *Service) SetURLs(urls URLs) error {
	if err := validateURLs(urls); err != nil {
		return err
	}
	s.mu.Lock()
	s.urls = urls
	s.mu.Unlock()
	return nil
}

func validateURLs(urls URLs) error {
	for _, raw := range []string{urls.GeoIP, urls.GeoSite, urls.MetaDB, urls.ASN} {
		u, err := url.Parse(raw)
		if err != nil || u.Scheme != "https" || u.Host == "" {
			return errors.New("geo URLs must use HTTPS")
		}
	}
	return nil
}
func (s *Service) SetProxy(proxyURL string) error {
	transport, err := outbound.NewTransport(proxyURL, 10*time.Second)
	if err != nil {
		return err
	}
	client := &http.Client{
		Timeout:   2 * time.Minute,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			return validateURL(req.URL)
		},
	}
	s.clientMu.Lock()
	s.client = client
	s.clientMu.Unlock()
	return nil
}

func (s *Service) Load() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snap = s.loadSnapshot()
}

// NeedsBootstrap 表示是否有任一必需 Geo 资源不可用（典型：Docker 空 volume 首次启动）。
// 为 true 时启动流程可后台调用一次 Update，失败不阻塞服务。
func (s *Service) NeedsBootstrap() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, name := range resourceNames {
		st, ok := s.snap.resources[name]
		if !ok || !st.status.Available {
			return true
		}
	}
	return false
}

func (s *Service) loadSnapshot() snapshot {
	next := snapshot{
		resources:  make(map[string]resourceState),
		categories: map[string][]Category{GeoSiteFile: {}, GeoIPFile: {}},
	}
	for _, name := range resourceNames {
		path := filepath.Join(s.dir, name)
		body, err := os.ReadFile(path)
		state := resourceState{status: ResourceStatus{Name: name}}
		if err != nil {
			state.status.Error = err.Error()
			next.resources[name] = state
			continue
		}
		state.body = body
		state.status = fileStatus(name, path, body)
		switch name {
		case GeoSiteFile:
			entries, parseErr := parseGeoSite(body)
			if parseErr != nil || len(entries) == 0 {
				err = parseErr
				if err == nil {
					err = errors.New("empty geosite database")
				}
			} else {
				next.domains = append(next.domains, entries...)
				counts := map[string]int{}
				for _, entry := range entries {
					counts[entry.Category]++
				}
				for category, count := range counts {
					next.categories[GeoSiteFile] = append(next.categories[GeoSiteFile], Category{Name: category, Count: count})
				}
			}
		case GeoIPFile:
			entries, parseErr := parseGeoIP(body)
			if parseErr != nil || len(entries) == 0 {
				err = parseErr
				if err == nil {
					err = errors.New("empty geoip database")
				}
			} else {
				next.cidrs = append(next.cidrs, entries...)
				counts := map[string]int{}
				for _, entry := range entries {
					counts[entry.Category]++
				}
				for category, count := range counts {
					next.categories[GeoIPFile] = append(next.categories[GeoIPFile], Category{Name: category, Count: count})
				}
			}
		case MetaDBFile:
			next.meta, err = maxminddb.FromBytes(body)
		case ASNFile:
			next.asn, err = maxminddb.FromBytes(body)
		}
		if err != nil {
			state.status.Error = err.Error()
		} else {
			state.status.Available = true
			if name == MetaDBFile && next.meta != nil {
				applyMMDBStatus(&state.status, next.meta)
			}
			if name == ASNFile && next.asn != nil {
				applyMMDBStatus(&state.status, next.asn)
			}
		}
		next.resources[name] = state
	}
	for _, key := range []string{GeoSiteFile, GeoIPFile} {
		sort.Slice(next.categories[key], func(i, j int) bool { return next.categories[key][i].Name < next.categories[key][j].Name })
	}
	next.search = buildSearchIndex(next.meta, next.asn)
	return next
}

func fileStatus(name, path string, body []byte) ResourceStatus {
	hash := sha256Hex(body)
	status := ResourceStatus{Name: name, Available: false, Size: int64(len(body)), SHA256: hash, Version: "sha256:" + hash[:16]}
	if st, err := os.Stat(path); err == nil {
		status.ModifiedAt = st.ModTime().UTC().Format(time.RFC3339)
	}
	return status
}

func applyMMDBStatus(status *ResourceStatus, reader *maxminddb.Reader) {
	status.DatabaseType = reader.Metadata.DatabaseType
	status.BuildEpoch = reader.Metadata.BuildEpoch
	if status.DatabaseType != "" {
		status.Version = fmt.Sprintf("build:%d", status.BuildEpoch)
	}
}

func sha256Hex(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func (s *Service) Status() []ResourceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]ResourceStatus, 0, len(resourceNames))
	for _, name := range resourceNames {
		items = append(items, s.snap.resources[name].status)
	}
	return items
}

func (s *Service) Categories() CategoriesResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return CategoriesResponse{
		GeoSite: append([]Category(nil), s.snap.categories[GeoSiteFile]...),
		GeoIP:   append([]Category(nil), s.snap.categories[GeoIPFile]...),
		MetaDB:  CategoryInfo{File: MetaDBFile, SupportsReverse: false},
		ASN:     CategoryInfo{File: ASNFile, SupportsReverse: false},
	}
}
