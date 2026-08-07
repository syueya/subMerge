package source

import (
	"fmt"
	"net/url"
	"strings"
)

func parseHysteria2URI(raw string) (map[string]interface{}, error) {
	// hysteria2://password@host:port?sni=...&insecure=1#name
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	password, _ := url.QueryUnescape(u.User.String())
	if password == "" {
		password = u.User.Username()
	}
	host := u.Hostname()
	port, err := parsePortOr443(u, "hysteria2")
	if err != nil {
		return nil, err
	}
	if host == "" || password == "" {
		return nil, fmt.Errorf("hysteria2 missing host/password")
	}
	name := fragmentName(u, "hy2", host, port)
	q := u.Query()
	m := map[string]interface{}{
		"name":     name,
		"type":     "hysteria2",
		"server":   host,
		"port":     port,
		"password": password,
		"udp":      true,
	}
	if sni := q.Get("sni"); sni != "" {
		m["sni"] = sni
	}
	if insecure := q.Get("insecure"); insecure == "1" || strings.EqualFold(insecure, "true") {
		m["skip-cert-verify"] = true
	}
	if obfs := q.Get("obfs"); obfs != "" {
		m["obfs"] = obfs
	}
	if obfsPass := q.Get("obfs-password"); obfsPass != "" {
		m["obfs-password"] = obfsPass
	}
	return m, nil
}
