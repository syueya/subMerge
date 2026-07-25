package source

import (
	"fmt"
	"net/url"
	"strings"
)

func parseVlessURI(raw string) (map[string]interface{}, error) {
	// vless://uuid@host:port?encryption=none&security=tls&type=ws&host=&path=&sni=#name
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	uuid := u.User.Username()
	host := u.Hostname()
	port, err := parsePortOr443(u, "vless")
	if err != nil {
		return nil, err
	}
	if host == "" || uuid == "" {
		return nil, fmt.Errorf("vless missing host/uuid")
	}
	name := fragmentName(u, "vless", host, port)
	q := u.Query()
	m := map[string]interface{}{
		"name":   name,
		"type":   "vless",
		"server": host,
		"port":   port,
		"uuid":   uuid,
		"udp":    true,
	}
	if enc := q.Get("encryption"); enc != "" {
		m["cipher"] = enc // Clash Meta 多用 flow/packet-encoding；cipher 作兼容
	}
	security := strings.ToLower(q.Get("security"))
	if security == "tls" || security == "reality" {
		m["tls"] = true
	}
	if security == "reality" {
		m["reality-opts"] = map[string]interface{}{
			"public-key": q.Get("pbk"),
			"short-id":   q.Get("sid"),
		}
	}
	sni := firstNonEmpty(q.Get("sni"), q.Get("peer"))
	if sni != "" {
		m["servername"] = sni
	}
	if fp := q.Get("fp"); fp != "" {
		m["client-fingerprint"] = fp
	}
	if flow := q.Get("flow"); flow != "" {
		m["flow"] = flow
	}
	network := firstNonEmpty(q.Get("type"), "tcp")
	if network != "" {
		m["network"] = network
	}
	switch strings.ToLower(network) {
	case "ws":
		opts := map[string]interface{}{}
		if path := q.Get("path"); path != "" {
			opts["path"] = path
		}
		if h := q.Get("host"); h != "" {
			opts["headers"] = map[string]interface{}{"Host": h}
		}
		if len(opts) > 0 {
			m["ws-opts"] = opts
		}
	case "grpc":
		opts := map[string]interface{}{}
		if sn := firstNonEmpty(q.Get("serviceName"), q.Get("path")); sn != "" {
			opts["grpc-service-name"] = sn
		}
		if len(opts) > 0 {
			m["grpc-opts"] = opts
		}
	}
	if alpn := q.Get("alpn"); alpn != "" {
		if parts := splitComma(alpn); len(parts) > 0 {
			m["alpn"] = parts
		}
	}
	return m, nil
}
