package source

import (
	"fmt"
	"net/url"
	"strings"
)

func parseShadowsocksURI(raw string) (map[string]interface{}, error) {
	// SIP002: ss://BASE64(method:password)@host:port#name
	// 旧式: ss://BASE64(method:password@host:port)#name
	u := raw
	name := ""
	if i := strings.Index(u, "#"); i >= 0 {
		frag, err := url.QueryUnescape(u[i+1:])
		if err == nil {
			name = strings.TrimSpace(frag)
		} else {
			name = strings.TrimSpace(u[i+1:])
		}
		u = u[:i]
	}
	body := u
	if strings.HasPrefix(strings.ToLower(body), "ss://") {
		body = body[5:]
	}
	var method, password, host string
	var port int

	if strings.Contains(body, "@") {
		// SIP002
		at := strings.LastIndex(body, "@")
		userinfo := body[:at]
		hostport := body[at+1:]
		// userinfo 可能是 base64(method:password) 或明文 method:password
		decoded, ok := tryBase64Decode([]byte(userinfo))
		userDecoded := userinfo
		if ok {
			userDecoded = string(decoded)
		} else if unesc, err := url.QueryUnescape(userinfo); err == nil {
			userDecoded = unesc
		}
		method, password, ok = splitMethodPassword(userDecoded)
		if !ok {
			return nil, fmt.Errorf("ss invalid userinfo")
		}
		h, p, err := splitHostPort(hostport)
		if err != nil {
			return nil, err
		}
		host, port = h, p
	} else {
		decoded, ok := tryBase64Decode([]byte(body))
		if !ok {
			return nil, fmt.Errorf("ss invalid legacy body")
		}
		// method:password@host:port
		legacy := string(decoded)
		at := strings.LastIndex(legacy, "@")
		if at < 0 {
			return nil, fmt.Errorf("ss legacy missing @")
		}
		var ok2 bool
		method, password, ok2 = splitMethodPassword(legacy[:at])
		if !ok2 {
			return nil, fmt.Errorf("ss legacy userinfo")
		}
		h, p, err := splitHostPort(legacy[at+1:])
		if err != nil {
			return nil, err
		}
		host, port = h, p
	}
	if name == "" {
		name = fmt.Sprintf("ss-%s-%d", host, port)
	}
	return map[string]interface{}{
		"name":     name,
		"type":     "ss",
		"server":   host,
		"port":     port,
		"cipher":   method,
		"password": password,
		"udp":      true,
	}, nil
}
