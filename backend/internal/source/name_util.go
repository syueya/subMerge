package source

import (
	"fmt"
	"strings"

	common "github.com/submerge/submerge/backend/common"
)

// EnsureRegionPrefix 保证节点名带目标地区前缀。
// 若已有其它地区码前缀（1–16 位 alnum + '-'）则剥掉再加目标前缀。
func EnsureRegionPrefix(name, region string) string {
	region = strings.ToUpper(strings.TrimSpace(region))
	name = strings.TrimSpace(name)
	if region == "" {
		return name
	}
	prefix := region + "-"
	if strings.HasPrefix(strings.ToUpper(name), strings.ToUpper(prefix)) {
		return name
	}
	rest := stripRegionPrefix(name)
	if rest == "" {
		rest = name
	}
	return prefix + rest
}

// SanitizeSourceSuffix 将订阅源名称整理为节点名后缀（如「良心云」）。
// 去掉路径分隔与控制字符；空白压成单个连字符；过长截断。
func SanitizeSourceSuffix(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(name))
	prevDash := false
	for _, r := range name {
		switch {
		case r < 0x20 || r == 0x7f:
			continue
		case r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' ||
			r == '"' || r == '<' || r == '>' || r == '|' || r == '#' || r == '%' || r == '@':
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		default:
			b.WriteRune(r)
			prevDash = false
		}
	}
	out := strings.Trim(b.String(), "-")
	runes := []rune(out)
	if len(runes) > 24 {
		out = string(runes[:24])
		out = strings.TrimRight(out, "-")
	}
	return out
}

func uniqueProxyName(base string, used map[string]struct{}) string {
	if _, exists := used[base]; !exists {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if _, exists := used[candidate]; !exists {
			return candidate
		}
	}
}

// stripSourceSuffix 若 name 以 -{源后缀} 结尾则剥掉（用于改名后重写）。
func stripSourceSuffix(name, sourceName string) string {
	suffix := SanitizeSourceSuffix(sourceName)
	if suffix == "" || name == "" {
		return name
	}
	if strings.HasSuffix(name, "-"+suffix) {
		return strings.TrimRight(strings.TrimSuffix(name, "-"+suffix), "-")
	}
	// ASCII 大小写不敏感
	upper := strings.ToUpper(name)
	tail := "-" + strings.ToUpper(suffix)
	if strings.HasSuffix(upper, tail) {
		return strings.TrimRight(name[:len(name)-len(tail)], "-")
	}
	return name
}

func normalizeRegionMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(common.RegionModeFixed):
		return string(common.RegionModeFixed)
	default:
		return string(common.RegionModeAuto)
	}
}

// FormatProxyName 生成入库节点名：地区前缀 + 原名 + 源后缀。
// 例：US-香港01-良心云
func FormatProxyName(rawName, region, sourceName string) string {
	base := EnsureRegionPrefix(rawName, region)
	suffix := SanitizeSourceSuffix(sourceName)
	if suffix == "" {
		return base
	}
	// 若已以 -源名 结尾（改地区/改名重写时），先去掉再拼，避免重复后缀
	base = stripSourceSuffix(base, sourceName)
	if base == "" {
		base = EnsureRegionPrefix(rawName, region)
	}
	return base + "-" + suffix
}

// stripRegionPrefix 去掉开头的已知地区码前缀（JP-/HK-…），避免误剥 Japan- 等整词。
func stripRegionPrefix(name string) string {
	ensureRegionDict()
	i := strings.Index(name, "-")
	if i < 2 || i > 16 {
		return name
	}
	code := strings.ToUpper(name[:i])
	if !isRegionCode(code) {
		return name
	}
	if _, ok := knownRegionCodes[code]; !ok {
		return name
	}
	rest := strings.TrimSpace(name[i+1:])
	if rest == "" {
		return name
	}
	return rest
}
