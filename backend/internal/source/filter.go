package source

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/submerge/submerge/backend/defaults"
	"gopkg.in/yaml.v3"
)

// FilterOptions 节点过滤选项
type FilterOptions struct {
	ExcludeNameRegex string
	ExcludeServers   string
	IncludeNameRegex string
}

// CompiledFilter 编译后的过滤器
type CompiledFilter struct {
	excludeName *regexp.Regexp
	includeName *regexp.Regexp
	servers     map[string]struct{}
}

type sourceFilterFile struct {
	ExcludeNameRegex string `yaml:"excludeNameRegex"`
	ExcludeServers   string `yaml:"excludeServers"`
	IncludeNameRegex string `yaml:"includeNameRegex"`
}

var (
	defaultFilterOnce sync.Once
	defaultFilterOpts FilterOptions
	defaultFilterErr  error
)

// DefaultFilterOptions 新建源时的默认过滤（来自 defaults/source_filter.yaml）
func DefaultFilterOptions() FilterOptions {
	loadDefaultFilter()
	return defaultFilterOpts
}

// DefaultExcludeNameRegex / DefaultExcludeServers 兼容旧测试与调用
func DefaultExcludeNameRegex() string { return DefaultFilterOptions().ExcludeNameRegex }
func DefaultExcludeServers() string   { return DefaultFilterOptions().ExcludeServers }

func loadDefaultFilter() {
	defaultFilterOnce.Do(func() {
		// 内置兜底，YAML 缺失/损坏时仍可用
		defaultFilterOpts = FilterOptions{
			ExcludeNameRegex: `剩余流量|套餐到期|流量|到期|过期|官网|电报|重置|距离下次|消耗|续费|客服|公告|测试|过滤掉|过滤了|已过滤`,
			ExcludeServers:   "127.0.0.1,0.0.0.0,localhost",
			IncludeNameRegex: "",
		}
		raw := defaults.SourceFilterYAML
		if len(raw) == 0 {
			defaultFilterErr = fmt.Errorf("embedded source_filter.yaml is empty")
			return
		}
		var doc sourceFilterFile
		if err := yaml.Unmarshal(raw, &doc); err != nil {
			defaultFilterErr = fmt.Errorf("parse source_filter.yaml: %w", err)
			return
		}
		defaultFilterOpts = FilterOptions{
			ExcludeNameRegex: strings.TrimSpace(doc.ExcludeNameRegex),
			ExcludeServers:   strings.TrimSpace(doc.ExcludeServers),
			IncludeNameRegex: strings.TrimSpace(doc.IncludeNameRegex),
		}
	})
}

// DefaultFilterError 返回默认过滤配置加载错误（正常为 nil）
func DefaultFilterError() error {
	loadDefaultFilter()
	return defaultFilterErr
}

// CompileFilter 编译过滤规则；正则非法时返回错误
func CompileFilter(opts FilterOptions) (*CompiledFilter, error) {
	cf := &CompiledFilter{
		servers: parseServerSet(opts.ExcludeServers),
	}
	if s := strings.TrimSpace(opts.ExcludeNameRegex); s != "" {
		re, err := regexp.Compile("(?i)" + s)
		if err != nil {
			return nil, fmt.Errorf("invalid excludeNameRegex: %w", err)
		}
		cf.excludeName = re
	}
	if s := strings.TrimSpace(opts.IncludeNameRegex); s != "" {
		re, err := regexp.Compile("(?i)" + s)
		if err != nil {
			return nil, fmt.Errorf("invalid includeNameRegex: %w", err)
		}
		cf.includeName = re
	}
	return cf, nil
}

// ShouldKeep 是否保留节点；reason 仅在丢弃时有值
func (cf *CompiledFilter) ShouldKeep(p ParsedProxy) (keep bool, reason string) {
	server := strings.ToLower(strings.TrimSpace(p.Server))
	if _, blocked := cf.servers[server]; blocked {
		return false, "server blocked: " + p.Server
	}
	if cf.excludeName != nil && cf.excludeName.MatchString(p.Name) {
		return false, "name excluded"
	}
	if cf.includeName != nil && !cf.includeName.MatchString(p.Name) {
		return false, "name not in include list"
	}
	return true, ""
}

func parseServerSet(raw string) map[string]struct{} {
	out := map[string]struct{}{}
	if strings.TrimSpace(raw) == "" {
		return out
	}
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	})
	for _, f := range fields {
		s := strings.ToLower(strings.TrimSpace(f))
		if s != "" {
			out[s] = struct{}{}
		}
	}
	return out
}
