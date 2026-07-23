package source

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// identityKeys 用于指纹的稳定身份字段（按优先级取第一个非空）
var identityKeys = []string{
	"uuid", "password", "private-key", "private_key", "token",
	"auth", "auth-str", "auth_str", "psk", "secret",
}

// ProxyFingerprint 计算节点稳定指纹，用于刷新时保留 enabled
func ProxyFingerprint(p ParsedProxy) string {
	id := firstString(p.Raw, identityKeys...)
	if id == "" {
		id = p.Name
	}
	base := fmt.Sprintf("%s|%s|%d|%s",
		strings.ToLower(strings.TrimSpace(p.Type)),
		strings.ToLower(strings.TrimSpace(p.Server)),
		p.Port,
		strings.TrimSpace(id),
	)
	sum := sha256.Sum256([]byte(base))
	return hex.EncodeToString(sum[:])
}

func firstString(m map[string]interface{}, keys ...string) string {
	if m == nil {
		return ""
	}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch t := v.(type) {
			case string:
				if s := strings.TrimSpace(t); s != "" {
					return s
				}
			}
		}
	}
	return ""
}
