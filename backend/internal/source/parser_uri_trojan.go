package source

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func parseTrojanURI(raw string) (map[string]interface{}, error) {
	// trojan://password@host:port?allowInsecure=1&peer=sni&sni=...#name
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	password, _ := url.QueryUnescape(u.User.String())
	if password == "" {
		password = u.User.Username()
	}
	host := u.Hostname()
	port := 443
	if p := u.Port(); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 65535 {
			return nil, fmt.Errorf("trojan invalid port")
		}
		port = n
	}
	if host == "" || password == "" {
		return nil, fmt.Errorf("trojan missing host/password")
	}
	name := strings.TrimSpace(u.Fragment)
	if name == "" {
		name = fmt.Sprintf("trojan-%s-%d", host, port)
	} else if unesc, err := url.QueryUnescape(name); err == nil {
		name = strings.TrimSpace(unesc)
	}
	q := u.Query()
	m := map[string]interface{}{
		"name":     name,
		"type":     "trojan",
		"server":   host,
		"port":     port,
		"password": password,
		"udp":      true,
	}
	sni := firstNonEmpty(q.Get("sni"), q.Get("peer"))
	if sni != "" {
		m["sni"] = sni
	}
	if network := q.Get("type"); network != "" && network != "tcp" {
		m["network"] = network
		if network == "ws" {
			opts := map[string]interface{}{}
			if path := q.Get("path"); path != "" {
				opts["path"] = path
			}
			if h := firstNonEmpty(q.Get("host"), q.Get("Host")); h != "" {
				opts["headers"] = map[string]interface{}{"Host": h}
			}
			if len(opts) > 0 {
				m["ws-opts"] = opts
			}
		}
	}
	if alpn := q.Get("alpn"); alpn != "" {
		if parts := splitComma(alpn); len(parts) > 0 {
			m["alpn"] = parts
		}
	}
	if insecure := q.Get("allowInsecure"); insecure == "1" || strings.EqualFold(insecure, "true") {
		m["skip-cert-verify"] = true
	}
	return m, nil
}
