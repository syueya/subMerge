package geo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/oschwald/maxminddb-golang"
	"google.golang.org/protobuf/encoding/protowire"
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
	dir    string
	urls   URLs
	mu     sync.RWMutex
	snap   snapshot
	client *http.Client
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

func (s *Service) Load() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snap = s.loadSnapshot()
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

func normalizeDomain(raw string) (string, error) {
	domain := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(raw), "."))
	if len(domain) == 0 || len(domain) > maxDomainLength || strings.Contains(domain, "/") || strings.ContainsAny(domain, " \t\r\n") {
		return "", errors.New("invalid domain")
	}
	return domain, nil
}

func domainMatch(entry domainRecord, domain string) bool {
	value := strings.ToLower(strings.TrimSuffix(entry.Value, "."))
	switch entry.Type {
	case "full":
		return domain == value
	case "domain":
		return domain == value || strings.HasSuffix(domain, "."+value)
	case "plain":
		return strings.Contains(domain, value)
	case "regex":
		matched, _ := regexp.MatchString(value, domain)
		return matched
	default:
		return false
	}
}

func (s *Service) Query(rawDomain string, resolve bool) (QueryResponse, error) {
	domain, err := normalizeDomain(rawDomain)
	if err != nil {
		return QueryResponse{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := QueryResponse{Domain: domain, GeoSite: []GeoSiteHit{}, GeoIP: []IPHit{}, MetaDB: []DBHit{}, ASN: []ASNHit{}}
	for _, entry := range s.snap.domains {
		if domainMatch(entry, domain) {
			result.GeoSite = append(result.GeoSite, GeoSiteHit{Category: entry.Category, Type: entry.Type, Value: entry.Value})
		}
	}
	if !resolve {
		result.ResolveSkipped = true
		return result, nil
	}
	ips, lookupErr := net.LookupIP(domain)
	if lookupErr != nil {
		result.ResolveError = lookupErr.Error()
		return result, nil
	}
	seen := map[string]bool{}
	for _, ip := range ips {
		ipText := ip.String()
		if seen[ipText] {
			continue
		}
		seen[ipText] = true
		result.IPs = append(result.IPs, ipText)
		for _, entry := range s.snap.cidrs {
			if entry.Network.Contains(ip) {
				result.GeoIP = append(result.GeoIP, IPHit{IP: ipText, Category: entry.Category, CIDR: entry.Network.String()})
			}
		}
		if s.snap.meta != nil {
			result.MetaDB = append(result.MetaDB, lookupMMDB(s.snap.meta, ip)...)
		}
		if s.snap.asn != nil {
			result.ASN = append(result.ASN, lookupASN(s.snap.asn, ip)...)
		}
	}
	return result, nil
}

func buildSearchIndex(meta, asn *maxminddb.Reader) []searchRecord {
	items := make([]searchRecord, 0)
	if asn != nil {
		it := asn.Networks(maxminddb.SkipAliasedNetworks)
		for it.Next() {
			var record struct {
				Number       uint32 `maxminddb:"autonomous_system_number"`
				Organization string `maxminddb:"autonomous_system_organization"`
			}
			network, err := it.Network(&record)
			if err != nil || network == nil {
				continue
			}
			items = append(items, searchRecord{File: ASNFile, ASN: record.Number, Organization: record.Organization, Network: network})
		}
	}
	if meta != nil {
		it := meta.Networks(maxminddb.SkipAliasedNetworks)
		for it.Next() {
			var record any
			network, err := it.Network(&record)
			if err != nil || network == nil {
				continue
			}
			codes := metaCodes(record)
			if len(codes) == 0 {
				continue
			}
			items = append(items, searchRecord{File: MetaDBFile, Codes: codes, Network: network, Detail: strings.Join(codes, ", ")})
		}
	}
	return items
}

func metaCodes(value any) []string {
	var codes []string
	var collect func(any)
	collect = func(item any) {
		switch v := item.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				codes = append(codes, strings.TrimSpace(v))
			}
		case []any:
			for _, nested := range v {
				collect(nested)
			}
		case []string:
			codes = append(codes, v...)
		}
	}
	collect(value)
	return codes
}

func (s *Service) Search(file, field, keyword string, limit, offset int) (SearchResponse, error) {
	file = strings.ToLower(strings.TrimSpace(file))
	field = strings.ToLower(strings.TrimSpace(field))
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if file == "asn" {
		file = ASNFile
	}
	if file == "metadb" {
		file = MetaDBFile
	}
	if file != ASNFile && file != MetaDBFile {
		return SearchResponse{}, errors.New("unsupported geo search file")
	}
	if file == ASNFile && field != "asn" && field != "organization" {
		return SearchResponse{}, errors.New("unsupported ASN search field")
	}
	if file == MetaDBFile && field != "code" && field != "country" && field != "region" {
		return SearchResponse{}, errors.New("unsupported MetaDB search field")
	}
	if keyword == "" || len(keyword) > maxCategoryLength {
		return SearchResponse{}, errors.New("invalid search keyword")
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	result := SearchResponse{File: file, Field: field, Keyword: keyword, Limit: limit, Offset: offset, Items: []SearchItem{}}
	all := make([]SearchItem, 0)
	for _, item := range s.snap.search {
		if item.File != file {
			continue
		}
		matched := false
		if file == ASNFile {
			if field == "asn" {
				matched = strings.Contains(strconv.FormatUint(uint64(item.ASN), 10), keyword)
			} else {
				matched = strings.Contains(strings.ToLower(item.Organization), keyword)
			}
		} else {
			for _, code := range item.Codes {
				if strings.Contains(strings.ToLower(code), keyword) {
					matched = true
					break
				}
			}
		}
		if !matched {
			continue
		}
		if file == ASNFile {
			all = append(all, SearchItem{Type: fmt.Sprintf("AS%d", item.ASN), Value: item.Network.String(), Detail: item.Organization})
		} else {
			all = append(all, SearchItem{Type: "code", Value: item.Network.String(), Detail: item.Detail})
		}
	}
	result.Total = len(all)
	if offset < len(all) {
		end := offset + limit
		if end > len(all) {
			end = len(all)
		}
		result.Items = all[offset:end]
	}
	return result, nil
}

func lookupMMDB(reader *maxminddb.Reader, ip net.IP) []DBHit {
	var record any
	network, ok, err := reader.LookupNetwork(ip, &record)
	if err != nil || !ok {
		return nil
	}
	body, _ := json.Marshal(record)
	if len(body) == 0 || string(body) == "null" {
		codes := metaCodes(record)
		body, _ = json.Marshal(codes)
	}
	return []DBHit{{IP: ip.String(), CIDR: network.String(), Record: string(body)}}
}
func lookupASN(reader *maxminddb.Reader, ip net.IP) []ASNHit {
	var record struct {
		Number       uint32 `maxminddb:"autonomous_system_number"`
		Organization string `maxminddb:"autonomous_system_organization"`
	}
	network, ok, err := reader.LookupNetwork(ip, &record)
	if err != nil || !ok {
		return nil
	}
	return []ASNHit{{IP: ip.String(), CIDR: network.String(), ASN: record.Number, Organization: record.Organization}}
}

func (s *Service) Reverse(file, category string, limit, offset int) (ReverseResponse, error) {
	file = strings.ToLower(strings.TrimSpace(file))
	category = strings.ToLower(strings.TrimSpace(category))
	if len(category) > maxCategoryLength {
		return ReverseResponse{}, errors.New("invalid category")
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := ReverseResponse{File: file, Category: category, Limit: limit, Offset: offset}
	if file == "geosite" {
		file = GeoSiteFile
	}
	if file == "geoip" {
		file = GeoIPFile
	}
	if file == "metadb" {
		file = MetaDBFile
	}
	if file == "asn" {
		file = ASNFile
	}
	switch file {
	case GeoSiteFile:
		items := make([]ReverseItem, 0)
		for _, entry := range s.snap.domains {
			if entry.Category == category {
				items = append(items, ReverseItem{Value: entry.Value, Type: entry.Type})
			}
		}
		return paginateReverse(result, items), nil
	case GeoIPFile:
		items := make([]ReverseItem, 0)
		for _, entry := range s.snap.cidrs {
			if entry.Category == category {
				items = append(items, ReverseItem{Value: entry.Network.String(), Type: "cidr"})
			}
		}
		return paginateReverse(result, items), nil
	case MetaDBFile, ASNFile:
		result.Message = "该文件保存 IP/ASN 数据，不保存域名，无法按分类反查域名"
		return result, nil
	default:
		return ReverseResponse{}, errors.New("unknown geo file")
	}
}

func paginateReverse(result ReverseResponse, items []ReverseItem) ReverseResponse {
	result.Total = len(items)
	if result.Offset >= len(items) {
		return result
	}
	end := result.Offset + result.Limit
	if end > len(items) {
		end = len(items)
	}
	result.Items = items[result.Offset:end]
	return result
}

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
		}
		result.Items = append(result.Items, out)
	}
	s.snap = s.loadSnapshot()
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
	resp, err := s.client.Do(req)
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

func consumeField(body []byte) (protowire.Number, protowire.Type, []byte, []byte, error) {
	num, typ, tagLen := protowire.ConsumeTag(body)
	if tagLen < 0 {
		return 0, 0, nil, nil, protowire.ParseError(tagLen)
	}
	valueLen := protowire.ConsumeFieldValue(num, typ, body[tagLen:])
	if valueLen < 0 {
		return 0, 0, nil, nil, protowire.ParseError(valueLen)
	}
	valueBytes := body[tagLen : tagLen+valueLen]
	var value []byte
	if typ == protowire.BytesType {
		value, valueLen = protowire.ConsumeBytes(valueBytes)
		if valueLen < 0 {
			return 0, 0, nil, nil, protowire.ParseError(valueLen)
		}
	} else {
		value = valueBytes
	}
	return num, typ, value, body[tagLen+valueLen:], nil
}

func parseGeoSite(body []byte) ([]domainRecord, error) {
	var result []domainRecord
	for len(body) > 0 {
		num, typ, value, rest, err := consumeField(body)
		if err != nil {
			return nil, errors.New("invalid geosite protobuf")
		}
		body = rest
		if typ != protowire.BytesType || num != 1 {
			continue
		}
		category, domains, err := parseGeoSiteEntry(value)
		if err != nil {
			return nil, err
		}
		for _, domain := range domains {
			result = append(result, domainRecord{Category: strings.ToLower(category), Type: domain.Type, Value: domain.Value})
		}
	}
	if len(result) == 0 {
		return nil, errors.New("empty geosite database")
	}
	return result, nil
}

func parseGeoSiteEntry(body []byte) (string, []domainRecord, error) {
	var category string
	var result []domainRecord
	for len(body) > 0 {
		num, typ, value, rest, err := consumeField(body)
		if err != nil {
			return "", nil, errors.New("invalid geosite entry")
		}
		body = rest
		switch {
		case num == 1 && typ == protowire.BytesType:
			category = string(value)
		case num == 2 && typ == protowire.BytesType:
			domain, err := parseDomain(value)
			if err != nil {
				continue
			}
			result = append(result, domain)
		}
	}
	if category == "" {
		return "", nil, errors.New("geosite entry has no category")
	}
	return category, result, nil
}

func parseDomain(body []byte) (domainRecord, error) {
	var result domainRecord
	for len(body) > 0 {
		num, typ, value, rest, err := consumeField(body)
		if err != nil {
			return result, errors.New("invalid geosite domain")
		}
		body = rest
		switch {
		case num == 1 && typ == protowire.VarintType:
			code, n := protowire.ConsumeVarint(value)
			if n < 0 {
				return result, errors.New("invalid geosite domain type")
			}
			switch code {
			case 0:
				result.Type = "plain"
			case 1:
				result.Type = "regex"
			case 2:
				result.Type = "domain"
			case 3:
				result.Type = "full"
			default:
				return result, errors.New("unsupported geosite domain type")
			}
		case num == 2 && typ == protowire.BytesType:
			result.Value = string(value)
		}
	}
	if result.Type == "" || result.Value == "" {
		return result, errors.New("empty geosite domain")
	}
	return result, nil
}

func parseGeoIP(body []byte) ([]cidrRecord, error) {
	var result []cidrRecord
	for len(body) > 0 {
		num, typ, value, rest, err := consumeField(body)
		if err != nil {
			return nil, errors.New("invalid geoip protobuf")
		}
		body = rest
		if typ != protowire.BytesType || num != 1 {
			continue
		}
		category, cidrs, err := parseGeoIPEntry(value)
		if err != nil {
			return nil, err
		}
		for _, network := range cidrs {
			result = append(result, cidrRecord{Category: strings.ToLower(category), Network: network})
		}
	}
	if len(result) == 0 {
		return nil, errors.New("empty geoip database")
	}
	return result, nil
}

func parseGeoIPEntry(body []byte) (string, []*net.IPNet, error) {
	var category string
	var result []*net.IPNet
	for len(body) > 0 {
		num, typ, value, rest, err := consumeField(body)
		if err != nil {
			return "", nil, errors.New("invalid geoip entry")
		}
		body = rest
		switch {
		case num == 1 && typ == protowire.BytesType:
			category = string(value)
		case num == 2 && typ == protowire.BytesType:
			network, err := parseCIDR(value)
			if err != nil {
				return "", nil, err
			}
			result = append(result, network)
		}
	}
	if category == "" {
		return "", nil, errors.New("geoip entry has no category")
	}
	return category, result, nil
}

func parseCIDR(body []byte) (*net.IPNet, error) {
	var ip net.IP
	var prefix uint64
	for len(body) > 0 {
		num, typ, value, rest, err := consumeField(body)
		if err != nil {
			return nil, errors.New("invalid geoip cidr")
		}
		body = rest
		switch {
		case num == 1 && typ == protowire.BytesType:
			ip = net.IP(append([]byte(nil), value...))
		case num == 2 && typ == protowire.VarintType:
			var n int
			prefix, n = protowire.ConsumeVarint(value)
			if n < 0 {
				return nil, errors.New("invalid geoip prefix")
			}
		}
	}
	bits := len(ip) * 8
	if (bits != 32 && bits != 128) || prefix > uint64(bits) {
		return nil, errors.New("invalid geoip network")
	}
	mask := net.CIDRMask(int(prefix), bits)
	return &net.IPNet{IP: ip.Mask(mask), Mask: mask}, nil
}
