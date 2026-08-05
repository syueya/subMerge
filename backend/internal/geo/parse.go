package geo

import (
	"errors"
	"net"
	"strings"

	"google.golang.org/protobuf/encoding/protowire"
)

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
