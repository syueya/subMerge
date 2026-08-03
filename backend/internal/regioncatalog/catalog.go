// Package regioncatalog 解析 defaults/regions.yaml：地区目录 + 识别表 + 常用标记。
// defaults 包只负责 go:embed 原始 YAML，不放业务逻辑。
package regioncatalog

import (
	"fmt"
	"strings"
	"sync"

	"github.com/submerge/submerge/backend/defaults"
	"gopkg.in/yaml.v3"
)

// Entry 地区目录项（UI 下拉 / 回退码）
type Entry struct {
	Code    string `json:"code" yaml:"code"`
	Name    string `json:"name" yaml:"name"`
	Primary bool   `json:"primary,omitempty" yaml:"primary,omitempty"`
}

// Keyword 节点名关键词 → 地区码
type Keyword struct {
	Keyword string
	Region  string
}

// FallbackCode 自动识别失败时的默认回退地区（短码，节点前缀如 UNK-xxx）
const FallbackCode = "UNK"

// LegacyFallbackCode 旧版回退码，兼容已有源/节点数据
const LegacyFallbackCode = "UNKNOWN"

// IsFallback 是否为回退/未识别地区（含旧码 UNKNOWN）
func IsFallback(code string) bool {
	c := strings.ToUpper(strings.TrimSpace(code))
	return c == "" || c == FallbackCode || c == LegacyFallbackCode
}

// OtherRegionToken 策略组成员 REGION:OTHER 展开用（非常用国家节点）
const OtherRegionToken = "OTHER"

type regionYAML struct {
	Code     string   `yaml:"code"`
	Name     string   `yaml:"name"`
	Primary  bool     `yaml:"primary"`
	Flag     string   `yaml:"flag"`
	Keywords []string `yaml:"keywords"`
}

type regionsFile struct {
	Regions []regionYAML `yaml:"regions"`
}

var (
	once     sync.Once
	list     []Entry
	byCode   map[string]Entry
	flags    map[string]string
	keywords []Keyword
	primary  map[string]struct{}
	loadErr  error
)

func load() {
	once.Do(func() {
		byCode = map[string]Entry{}
		flags = map[string]string{}
		primary = map[string]struct{}{}
		keywords = nil
		fallback := []Entry{{Code: FallbackCode, Name: "未知"}}
		raw := defaults.RegionsYAML
		if len(raw) == 0 {
			loadErr = fmt.Errorf("embedded regions.yaml is empty")
			list = fallback
			byCode[FallbackCode] = fallback[0]
			return
		}
		var doc regionsFile
		if err := yaml.Unmarshal(raw, &doc); err != nil {
			loadErr = fmt.Errorf("parse regions.yaml: %w", err)
			list = fallback
			byCode[FallbackCode] = fallback[0]
			return
		}
		out := make([]Entry, 0, len(doc.Regions))
		kws := make([]Keyword, 0, 128)
		for _, r := range doc.Regions {
			code := strings.ToUpper(strings.TrimSpace(r.Code))
			name := strings.TrimSpace(r.Name)
			if code == "" {
				continue
			}
			if name == "" {
				name = code
			}
			e := Entry{Code: code, Name: name, Primary: r.Primary && code != FallbackCode}
			out = append(out, e)
			byCode[code] = e
			if e.Primary {
				primary[code] = struct{}{}
			}
			if flag := strings.TrimSpace(r.Flag); flag != "" && code != FallbackCode {
				flags[flag] = code
			}
			for _, kw := range r.Keywords {
				kw = strings.TrimSpace(kw)
				if kw == "" || code == FallbackCode {
					continue
				}
				kws = append(kws, Keyword{Keyword: kw, Region: code})
			}
		}
		if len(out) == 0 {
			loadErr = fmt.Errorf("regions.yaml has no regions")
			list = fallback
			byCode[FallbackCode] = fallback[0]
			return
		}
		if _, ok := byCode[FallbackCode]; !ok {
			e := Entry{Code: FallbackCode, Name: "未知"}
			out = append([]Entry{e}, out...)
			byCode[FallbackCode] = e
		}
		// 长关键词优先，避免短码抢先
		for i := 0; i < len(kws); i++ {
			for j := i + 1; j < len(kws); j++ {
				if len(kws[j].Keyword) > len(kws[i].Keyword) {
					kws[i], kws[j] = kws[j], kws[i]
				}
			}
		}
		list = out
		keywords = kws
	})
}

// List 返回地区目录（含 UNK 回退项）
func List() []Entry {
	load()
	out := make([]Entry, len(list))
	copy(out, list)
	return out
}

// Name 查中文名，未知码原样返回
func Name(code string) string {
	load()
	c := strings.ToUpper(strings.TrimSpace(code))
	if e, ok := byCode[c]; ok {
		return e.Name
	}
	return c
}

// DisplayName 策略组展示名：美国US；未知码返回 code
func DisplayName(code string) string {
	load()
	c := strings.ToUpper(strings.TrimSpace(code))
	if IsFallback(c) {
		return c
	}
	if e, ok := byCode[c]; ok {
		if e.Name != "" && e.Name != c {
			return e.Name + c
		}
	}
	return c
}

// IsPrimary 是否常用国家（默认独立策略组）
func IsPrimary(code string) bool {
	load()
	c := strings.ToUpper(strings.TrimSpace(code))
	_, ok := primary[c]
	return ok
}

// PrimaryCodes 常用国家码列表（稳定顺序，跟随 YAML）
func PrimaryCodes() []string {
	load()
	out := make([]string, 0, len(primary))
	for _, e := range list {
		if e.Primary {
			out = append(out, e.Code)
		}
	}
	return out
}

// Flags 国旗 emoji → 地区码
func Flags() map[string]string {
	load()
	out := make(map[string]string, len(flags))
	for k, v := range flags {
		out[k] = v
	}
	return out
}

// Keywords 节点名关键词表（已按长度降序）
func Keywords() []Keyword {
	load()
	out := make([]Keyword, len(keywords))
	copy(out, keywords)
	return out
}

// KnownCodes 全部已知地区码集合
func KnownCodes() map[string]struct{} {
	load()
	out := make(map[string]struct{}, len(byCode))
	for c := range byCode {
		out[c] = struct{}{}
	}
	return out
}

// Error 目录加载错误（正常为 nil）
func Error() error {
	load()
	return loadErr
}
