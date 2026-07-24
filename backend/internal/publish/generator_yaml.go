package publish

import (
	"gopkg.in/yaml.v3"
	"regexp"
	"sort"
)

// yamlQuotedString 强制双引号输出，避免 short-id: 6314e825 被 YAML 1.1 解析成 .inf
type yamlQuotedString string

func (s yamlQuotedString) MarshalYAML() (interface{}, error) {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Style: yaml.DoubleQuotedStyle,
		Tag:   "!!str",
		Value: string(s),
	}, nil
}

// yamlMap 有序 map，保证 name/type 等关键字段排在前面（map 默认乱序不好读）
type yamlMap []yamlKV

type yamlKV struct {
	Key   string
	Value interface{}
}

func (m yamlMap) MarshalYAML() (interface{}, error) {
	node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, kv := range m {
		var valNode yaml.Node
		if err := valNode.Encode(kv.Value); err != nil {
			return nil, err
		}
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: kv.Key},
			&valNode,
		)
	}
	return node, nil
}

// proxyFieldOrder 节点字段展示顺序：name 最前，便于对照 proxy N 与策略组成员
var proxyFieldOrder = []string{
	"name", "type", "server", "port", "uuid", "password", "cipher", "alterId",
	"network", "tls", "udp", "flow", "servername", "client-fingerprint",
	"skip-cert-verify", "sni", "fingerprint", "ports",
	"reality-opts", "ws-opts", "grpc-opts", "h2-opts", "http-opts",
	"smux", "tfo", "mptcp",
}

// groupFieldOrder 策略组字段顺序
var groupFieldOrder = []string{"name", "type", "proxies", "url", "interval", "lazy", "tolerance"}

// realityFieldOrder REALITY 子字段顺序
var realityFieldOrder = []string{"public-key", "short-id", "support-x25519mlkem768"}

func orderedMap(m map[string]interface{}, prefer []string) yamlMap {
	if m == nil {
		return nil
	}
	out := make(yamlMap, 0, len(m))
	seen := map[string]struct{}{}
	for _, k := range prefer {
		if v, ok := m[k]; ok {
			out = append(out, yamlKV{Key: k, Value: v})
			seen[k] = struct{}{}
		}
	}
	rest := make([]string, 0, len(m))
	for k := range m {
		if _, ok := seen[k]; !ok {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	for _, k := range rest {
		out = append(out, yamlKV{Key: k, Value: m[k]})
	}
	return out
}

func orderedProxy(m map[string]interface{}) yamlMap {
	// 嵌套 opts 也固定顺序
	if ro, ok := m["reality-opts"].(map[string]interface{}); ok {
		m["reality-opts"] = orderedMap(ro, realityFieldOrder)
	}
	return orderedMap(m, proxyFieldOrder)
}

func orderedGroup(m map[string]interface{}) yamlMap {
	return orderedMap(m, groupFieldOrder)
}

// realityShortIDRe mihomo：hex，解码后 ≤8 字节（0~16 个 hex 字符，可为空）
var realityShortIDRe = regexp.MustCompile(`(?i)^[0-9a-f]{0,16}$`)
