package source

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func parseVmessURI(raw string) (map[string]interface{}, error) {
	payload := raw[len("vmess://"):]
	// 去掉可能的 fragment 前先 base64
	payload = strings.TrimSpace(payload)
	decoded, ok := tryBase64Decode([]byte(payload))
	if !ok {
		// 部分实现 payload 本身是 JSON 明文（少见）
		decoded = []byte(payload)
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(decoded, &obj); err != nil {
		return nil, fmt.Errorf("vmess json: %w", err)
	}
	// v2rayN 字段：v,ps,add,port,id,aid,scy,net,type,host,path,tls,sni,alpn
	name := firstNonEmpty(
		coerceString(obj["ps"]),
		coerceString(obj["remark"]),
		coerceString(obj["name"]),
	)
	server := firstNonEmpty(coerceString(obj["add"]), coerceString(obj["host"]), coerceString(obj["server"]))
	// host 在 vmess 里常作 sniffer 域名；server 优先 add
	if add := coerceString(obj["add"]); add != "" {
		server = add
	}
	port, ok := coercePort(obj["port"])
	if !ok {
		return nil, fmt.Errorf("vmess invalid port")
	}
	uuid := firstNonEmpty(coerceString(obj["id"]), coerceString(obj["uuid"]))
	if server == "" || uuid == "" {
		return nil, fmt.Errorf("vmess missing server/uuid")
	}
	if name == "" {
		name = fmt.Sprintf("vmess-%s-%d", server, port)
	}
	alterID := 0
	if v, ok := coercePort(obj["aid"]); ok {
		alterID = v
	} else if n, err := strconv.Atoi(coerceString(obj["aid"])); err == nil && n >= 0 {
		alterID = n
	}
	cipher := firstNonEmpty(coerceString(obj["scy"]), coerceString(obj["security"]), "auto")
	network := firstNonEmpty(coerceString(obj["net"]), coerceString(obj["network"]), "tcp")
	tls := strings.ToLower(coerceString(obj["tls"]))
	m := map[string]interface{}{
		"name":    name,
		"type":    "vmess",
		"server":  server,
		"port":    port,
		"uuid":    uuid,
		"alterId": alterID,
		"cipher":  cipher,
		"udp":     true,
	}
	if network != "" && network != "tcp" {
		m["network"] = network
	} else if network == "tcp" {
		m["network"] = "tcp"
	}
	if tls == "tls" || tls == "true" || tls == "1" {
		m["tls"] = true
	}
	sni := firstNonEmpty(coerceString(obj["sni"]), coerceString(obj["servername"]))
	if sni != "" {
		m["servername"] = sni
	}
	host := coerceString(obj["host"])
	path := coerceString(obj["path"])
	switch strings.ToLower(network) {
	case "ws":
		opts := map[string]interface{}{}
		if path != "" {
			opts["path"] = path
		}
		if host != "" {
			opts["headers"] = map[string]interface{}{"Host": host}
		}
		if len(opts) > 0 {
			m["ws-opts"] = opts
		}
	case "http":
		opts := map[string]interface{}{}
		if path != "" {
			opts["path"] = []interface{}{path}
		}
		if host != "" {
			opts["headers"] = map[string]interface{}{"Host": []interface{}{host}}
		}
		if len(opts) > 0 {
			m["http-opts"] = opts
		}
	case "grpc":
		opts := map[string]interface{}{}
		if path != "" {
			opts["grpc-service-name"] = path
		}
		if len(opts) > 0 {
			m["grpc-opts"] = opts
		}
	case "h2":
		opts := map[string]interface{}{}
		if path != "" {
			opts["path"] = path
		}
		if host != "" {
			opts["host"] = []interface{}{host}
		}
		if len(opts) > 0 {
			m["h2-opts"] = opts
		}
	}
	if alpn := coerceString(obj["alpn"]); alpn != "" {
		parts := splitComma(alpn)
		if len(parts) > 0 {
			m["alpn"] = parts
		}
	}
	return m, nil
}
