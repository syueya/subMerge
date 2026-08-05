package source

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"unicode"
)

// DecodeSubscriptionBody 规范化上游订阅正文。
// 机场常见：整份 Clash YAML 再 Base64 一次；也可能是明文 YAML 或分享链接列表。
func DecodeSubscriptionBody(body []byte) ([]byte, error) {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil, fmt.Errorf("empty subscription body")
	}
	// UTF-8 BOM
	body = bytes.TrimPrefix(body, []byte{0xEF, 0xBB, 0xBF})

	// 已是 YAML 或明文 URI 列表则直接用
	if looksLikeYAML(body) || looksLikeURIList(body) {
		return body, nil
	}

	// 尝试 Base64（标准 / URL / 无填充）。
	// tryBase64Decode 已保证解码结果为「基本可打印文本」，此处直接采用；
	// 后续 ParseClashProxiesDetailed 会再判定它是 YAML 还是 URI 列表。
	if decoded, ok := tryBase64Decode(body); ok {
		decoded = bytes.TrimSpace(decoded)
		decoded = bytes.TrimPrefix(decoded, []byte{0xEF, 0xBB, 0xBF})
		if len(decoded) > 0 {
			return decoded, nil
		}
	}

	// 原样返回，交给后续解析器给出明确错误
	return body, nil
}

func looksLikeYAML(b []byte) bool {
	s := strings.TrimLeftFunc(string(b), unicode.IsSpace)
	if s == "" {
		return false
	}
	// 注释或文档开始
	if strings.HasPrefix(s, "#") || strings.HasPrefix(s, "---") {
		return true
	}
	lower := strings.ToLower(s)
	for _, p := range []string{
		"proxies:", "proxy-groups:", "rules:", "port:", "mixed-port:",
		"socks-port:", "mode:", "allow-lan:", "log-level:", "dns:",
	} {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	// 任意一行以 proxies: 开头也算
	for _, line := range strings.Split(lower, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "proxies:") {
			return true
		}
	}
	return false
}

func tryBase64Decode(raw []byte) ([]byte, bool) {
	// 去掉空白/换行（部分订阅会折行）
	compact := make([]byte, 0, len(raw))
	for _, c := range raw {
		if c == ' ' || c == '\n' || c == '\r' || c == '\t' {
			continue
		}
		compact = append(compact, c)
	}
	if len(compact) < 8 {
		return nil, false
	}
	// 仅允许 base64 字符
	for _, c := range compact {
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') || c == '+' || c == '/' || c == '=' ||
			c == '-' || c == '_' {
			continue
		}
		return nil, false
	}

	// 补齐 padding
	pad := len(compact) % 4
	if pad > 0 {
		compact = append(compact, bytes.Repeat([]byte("="), 4-pad)...)
	}

	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.RawURLEncoding,
	}
	for _, enc := range encodings {
		out, err := enc.DecodeString(string(compact))
		if err != nil || len(out) == 0 {
			continue
		}
		// 解码结果应主要是可打印文本
		if !mostlyPrintable(out) {
			continue
		}
		return out, true
	}
	return nil, false
}

func mostlyPrintable(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	bad := 0
	for _, c := range b {
		// 允许可打印字符与常见空白（\t=0x09 \n=0x0a \r=0x0d）；
		// 其余控制字符（0x00–0x08、0x0b、0x0c、0x0e–0x1f、0x7f）计为坏字节
		if c == '\t' || c == '\n' || c == '\r' {
			continue
		}
		if c < 0x20 || c == 0x7f {
			bad++
		}
	}
	return bad*20 < len(b) // 允许少量二进制噪声
}

// extractYAMLBlock 从可能夹杂前后缀的文本中截取 YAML 主体
func extractYAMLBlock(b []byte) []byte {
	s := string(b)
	idx := strings.Index(strings.ToLower(s), "proxies:")
	if idx < 0 {
		return nil
	}
	// 向前找到行首
	start := idx
	for start > 0 && s[start-1] != '\n' {
		start--
	}
	return []byte(strings.TrimSpace(s[start:]))
}
