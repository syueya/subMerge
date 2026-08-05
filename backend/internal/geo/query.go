package geo

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/oschwald/maxminddb-golang"
)

func normalizeDomain(raw string) (string, error) {
	domain := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(raw), "."))
	if len(domain) == 0 || len(domain) > maxDomainLength || strings.Contains(domain, "/") || strings.ContainsAny(domain, " \t\r\n") {
		return "", errors.New("invalid domain")
	}
	return domain, nil
}

// normalizeQuery accepts either an IP address or a domain name.
// IP is preferred when net.ParseIP succeeds after trimming.
func normalizeQuery(raw string) (inputType string, value string, ip net.IP, err error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", "", nil, errors.New("invalid domain or IP")
	}
	if parsed := net.ParseIP(trimmed); parsed != nil {
		return "ip", parsed.String(), parsed, nil
	}
	domain, err := normalizeDomain(trimmed)
	if err != nil {
		return "", "", nil, errors.New("invalid domain or IP")
	}
	return "domain", domain, nil, nil
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

func (s *Service) appendIPLookups(result *QueryResponse, ip net.IP) {
	ipText := ip.String()
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

func (s *Service) Query(rawDomain string, resolve bool) (QueryResponse, error) {
	inputType, value, parsedIP, err := normalizeQuery(rawDomain)
	if err != nil {
		return QueryResponse{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := QueryResponse{
		Domain:    value,
		InputType: inputType,
		GeoSite:   []GeoSiteHit{},
		GeoIP:     []IPHit{},
		MetaDB:    []DBHit{},
		ASN:       []ASNHit{},
	}

	if inputType == "ip" {
		// Direct IP lookup: no GeoSite, no DNS.
		result.ResolveSkipped = true
		s.appendIPLookups(&result, parsedIP)
		return result, nil
	}

	for _, entry := range s.snap.domains {
		if domainMatch(entry, value) {
			result.GeoSite = append(result.GeoSite, GeoSiteHit{Category: entry.Category, Type: entry.Type, Value: entry.Value})
		}
	}
	if !resolve {
		result.ResolveSkipped = true
		return result, nil
	}
	ips, lookupErr := net.LookupIP(value)
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
		s.appendIPLookups(&result, ip)
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

