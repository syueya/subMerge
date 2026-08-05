package source

import (
	"sort"
	"strings"
	"sync"
	"unicode"

	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/regioncatalog"
)

type regionKeyword struct {
	Keyword string
	Region  string
}

var (
	flagToRegion     map[string]string
	orderedFlags     []string
	keywordToRegion  []regionKeyword
	knownRegionCodes map[string]struct{}
	regionDictOnce   sync.Once
	regionDictErr    error
)

// loadRegionDict 从 regioncatalog（defaults/regions.yaml）加载识别表（进程内一次）
func loadRegionDict() {
	regionDictOnce.Do(func() {
		if err := regioncatalog.Error(); err != nil {
			regionDictErr = err
			flagToRegion = map[string]string{}
			keywordToRegion = nil
			knownRegionCodes = map[string]struct{}{}
			return
		}
		flagToRegion = regioncatalog.Flags()
		orderedFlags = make([]string, 0, len(flagToRegion))
		for flag := range flagToRegion {
			orderedFlags = append(orderedFlags, flag)
		}
		sort.SliceStable(orderedFlags, func(i, j int) bool {
			return orderedFlags[i] < orderedFlags[j]
		})
		kws := regioncatalog.Keywords()
		keywordToRegion = make([]regionKeyword, 0, len(kws))
		for _, k := range kws {
			keywordToRegion = append(keywordToRegion, regionKeyword{Keyword: k.Keyword, Region: k.Region})
		}
		knownRegionCodes = regioncatalog.KnownCodes()
	})
}

// listRegionCatalog / fallbackRegionCode 供 handler 使用
func listRegionCatalog() []regioncatalog.Entry {
	return regioncatalog.List()
}

func fallbackRegionCode() string {
	return regioncatalog.FallbackCode
}

func ensureRegionDict() {
	loadRegionDict()
	if regionDictErr != nil {
		// 字典损坏时识别退化为空表 + 源默认地区；测试/启动可用 RegionDictError 检查
		return
	}
}

// RegionDictError 返回地区字典加载错误（正常为 nil）
func RegionDictError() error {
	loadRegionDict()
	return regionDictErr
}

// DetectMatch 单次地区识别的详细结果（用于日志）
type DetectMatch struct {
	// Region 识别到的地区码；失败为空
	Region string
	// Method 匹配方式：flag | keyword | prefix | conflict | none
	Method string
	// Matched 命中的关键词/国旗/前缀原文
	Matched string
	// FlagRegion / FlagMatched 国旗识别结果。
	FlagRegion  string
	FlagMatched string
	// KeywordRegion / KeywordMatched 节点名称关键词识别结果。
	KeywordRegion  string
	KeywordMatched string
	// Conflict 表示国旗和关键词识别到了不同地区。
	Conflict bool
}

// DetectRegion 仅识别节点名中的地区（不套源默认）
func DetectRegion(name string) string {
	return DetectRegionDetailed(name).Region
}

// DetectRegionDetailed 带方式/命中片段
func DetectRegionDetailed(name string) DetectMatch {
	ensureRegionDict()
	name = strings.TrimSpace(name)
	if name == "" {
		return DetectMatch{Method: "none"}
	}

	flagRegion, flagMatched := matchFlagRegion(name)
	keywordRegion, keywordMatched := matchKeywordRegion(name)

	if flagRegion != "" {
		if keywordRegion != "" && flagRegion != keywordRegion {
			// 国旗与关键词冲突时优先关键词（常见于错误国旗）
			return DetectMatch{
				Region:         keywordRegion,
				Method:         "conflict",
				Matched:        keywordMatched,
				FlagRegion:     flagRegion,
				FlagMatched:    flagMatched,
				KeywordRegion:  keywordRegion,
				KeywordMatched: keywordMatched,
				Conflict:       true,
			}
		}
		return DetectMatch{
			Region:         flagRegion,
			Method:         "flag",
			Matched:        flagMatched,
			FlagRegion:     flagRegion,
			FlagMatched:    flagMatched,
			KeywordRegion:  keywordRegion,
			KeywordMatched: keywordMatched,
		}
	}
	if keywordRegion != "" {
		return DetectMatch{
			Region:         keywordRegion,
			Method:         "keyword",
			Matched:        keywordMatched,
			FlagRegion:     flagRegion,
			FlagMatched:    flagMatched,
			KeywordRegion:  keywordRegion,
			KeywordMatched: keywordMatched,
		}
	}

	// 3) 已有地区前缀 US- / JP-
	if code, ok := regionPrefixFromName(name); ok {
		return DetectMatch{Region: code, Method: "prefix", Matched: code}
	}
	return DetectMatch{Method: "none"}
}

func matchFlagRegion(name string) (region, matched string) {
	for _, flag := range orderedFlags {
		if flag != "" && strings.Contains(name, flag) {
			return flagToRegion[flag], flag
		}
	}
	return "", ""
}

func matchKeywordRegion(name string) (region, matched string) {
	lower := strings.ToLower(name)
	for _, item := range keywordToRegion {
		kw := item.Keyword
		if kw == "" {
			continue
		}
		// 短码（<=3 且全 alnum）要求词边界，避免 in 命中 singapore 等
		if isShortCode(kw) {
			if matchShortCode(lower, strings.ToLower(kw)) {
				return item.Region, kw
			}
			continue
		}
		if strings.Contains(lower, strings.ToLower(kw)) {
			return item.Region, kw
		}
	}
	return "", ""
}

func isShortCode(s string) bool {
	if len(s) == 0 || len(s) > 3 {
		return false
	}
	for _, c := range s {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}

func matchShortCode(lowerName, lowerKW string) bool {
	// 简单词边界：前后非字母数字
	idx := 0
	for {
		i := strings.Index(lowerName[idx:], lowerKW)
		if i < 0 {
			return false
		}
		i += idx
		beforeOK := i == 0 || !isWordChar(rune(lowerName[i-1]))
		after := i + len(lowerKW)
		afterOK := after >= len(lowerName) || !isWordChar(rune(lowerName[after]))
		if beforeOK && afterOK {
			return true
		}
		idx = i + 1
		if idx >= len(lowerName) {
			return false
		}
	}
}

func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func regionPrefixFromName(name string) (string, bool) {
	ensureRegionDict()
	i := strings.Index(name, "-")
	if i < 2 || i > 16 {
		return "", false
	}
	code := strings.ToUpper(name[:i])
	if !isRegionCode(code) {
		return "", false
	}
	if _, ok := knownRegionCodes[code]; !ok {
		return "", false
	}
	return code, true
}

func isRegionCode(code string) bool {
	if code == "" {
		return false
	}
	for _, c := range code {
		if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

// ResolveRegion 按模式解析最终地区码
func ResolveRegion(name, mode, defaultRegion string) string {
	return ResolveRegionDetailed(name, mode, defaultRegion).Region
}

// RegionResolveResult 含 fallback 标记
type RegionResolveResult struct {
	Region       string
	Detect       DetectMatch
	UsedFallback bool
}

// ResolveRegionDetailed 自动/固定模式解析
func ResolveRegionDetailed(name, mode, defaultRegion string) RegionResolveResult {
	mode = strings.ToLower(strings.TrimSpace(mode))
	defaultRegion = strings.ToUpper(strings.TrimSpace(defaultRegion))
	if mode == string(common.RegionModeFixed) {
		if defaultRegion == "" {
			defaultRegion = fallbackRegionCode()
		}
		return RegionResolveResult{
			Region: defaultRegion,
			Detect: DetectMatch{Region: defaultRegion, Method: "fixed", Matched: defaultRegion},
		}
	}
	// auto
	d := DetectRegionDetailed(name)
	if d.Region != "" {
		return RegionResolveResult{Region: d.Region, Detect: d}
	}
	if defaultRegion == "" {
		defaultRegion = fallbackRegionCode()
	}
	return RegionResolveResult{
		Region:       defaultRegion,
		Detect:       d,
		UsedFallback: true,
	}
}
