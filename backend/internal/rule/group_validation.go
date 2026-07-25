package rule

import (
	"fmt"
	"net/url"
	"strings"

	common "github.com/submerge/submerge/backend/common"
)

const (
	maxGroupNameLen   = 128
	maxGroupMemberLen = 255
	maxGroupMembers   = 2000
	maxGroupJSONBytes = 512 << 10
	maxGroupURLBytes  = 255
	minGroupInterval  = 5
	maxGroupInterval  = 86400
)

func normalizeGroupRequest(req common.UpsertProxyGroupRequest) (common.UpsertProxyGroupRequest, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Type = strings.ToLower(strings.TrimSpace(req.Type))
	req.URL = strings.TrimSpace(req.URL)
	if req.Name == "" {
		return req, fmt.Errorf("group name required")
	}
	if len([]rune(req.Name)) > maxGroupNameLen {
		return req, fmt.Errorf("group name too long")
	}
	switch common.ProxyGroupType(req.Type) {
	case common.ProxyGroupTypeSelect, common.ProxyGroupTypeURLTest,
		common.ProxyGroupTypeFallback, common.ProxyGroupTypeLoadBalance:
	default:
		return req, fmt.Errorf("unsupported group type %q", req.Type)
	}
	seen := make(map[string]struct{}, len(req.Proxies))
	members := make([]string, 0, len(req.Proxies))
	for _, raw := range req.Proxies {
		member := strings.TrimSpace(raw)
		if member == "" {
			continue
		}
		if len([]rune(member)) > maxGroupMemberLen {
			return req, fmt.Errorf("group member too long")
		}
		if _, ok := seen[member]; ok {
			continue
		}
		seen[member] = struct{}{}
		members = append(members, member)
	}
	if len(members) == 0 {
		return req, fmt.Errorf("group must contain at least one member")
	}
	if len(members) > maxGroupMembers {
		return req, fmt.Errorf("too many group members; maximum is %d", maxGroupMembers)
	}
	if len([]byte(string(mustJSON(members)))) > maxGroupJSONBytes {
		return req, fmt.Errorf("group members are too large")
	}
	req.Proxies = members
	if req.Type == string(common.ProxyGroupTypeURLTest) || req.Type == string(common.ProxyGroupTypeFallback) {
		if req.URL == "" {
			return req, fmt.Errorf("group type %s requires url", req.Type)
		}
		u, err := url.ParseRequestURI(req.URL)
		if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return req, fmt.Errorf("group url must be an http or https URL")
		}
		if len(req.URL) > maxGroupURLBytes {
			return req, fmt.Errorf("group url too long")
		}
		if req.Interval == nil {
			v := 300
			req.Interval = &v
		}
		if *req.Interval < minGroupInterval || *req.Interval > maxGroupInterval {
			return req, fmt.Errorf("group interval must be between %d and %d seconds", minGroupInterval, maxGroupInterval)
		}
	} else {
		req.URL = ""
		req.Interval = nil
	}
	return req, nil
}
