package source

import (
	"net/http"
	"strconv"
	"strings"
)

// SubscriptionUserInfo 上游响应头 Subscription-Userinfo
// 格式：upload=…; download=…; total=…; expire=…
// 与 Clash Verge / mihomo 客户端约定一致。
type SubscriptionUserInfo struct {
	Upload   int64 // 已上传字节
	Download int64 // 已下载字节
	Total    int64 // 套餐总量字节；0 常表示未知/不限
	Expire   int64 // 到期 Unix 秒；0 常表示未知/不限
}

// HasAny 是否带有任何有效字段（用于判断是否写入库）
func (u SubscriptionUserInfo) HasAny() bool {
	return u.Upload > 0 || u.Download > 0 || u.Total > 0 || u.Expire > 0
}

// Used 已用流量（上传+下载）
func (u SubscriptionUserInfo) Used() int64 {
	if u.Upload < 0 {
		u.Upload = 0
	}
	if u.Download < 0 {
		u.Download = 0
	}
	return u.Upload + u.Download
}

// ParseSubscriptionUserInfo 解析单条 header 值
func ParseSubscriptionUserInfo(raw string) (SubscriptionUserInfo, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return SubscriptionUserInfo{}, false
	}
	var out SubscriptionUserInfo
	found := false
	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		k = strings.ToLower(strings.TrimSpace(k))
		v = strings.TrimSpace(v)
		// 去掉引号
		v = strings.Trim(v, `"'`)
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 0 {
			continue
		}
		switch k {
		case "upload":
			out.Upload = n
			found = true
		case "download":
			out.Download = n
			found = true
		case "total":
			out.Total = n
			found = true
		case "expire":
			out.Expire = n
			found = true
		}
	}
	return out, found
}

// ParseSubscriptionUserInfoFromHeaders 从响应头提取（兼容 x-*-meta-subscription-userinfo）
func ParseSubscriptionUserInfoFromHeaders(h http.Header) (SubscriptionUserInfo, bool) {
	if h == nil {
		return SubscriptionUserInfo{}, false
	}
	// 精确标准名优先
	if v := h.Get("Subscription-Userinfo"); v != "" {
		if info, ok := ParseSubscriptionUserInfo(v); ok {
			return info, true
		}
	}
	// 兼容 CDN / 对象存储 meta 前缀：x-amz-meta-subscription-userinfo 等
	for k, vals := range h {
		if len(vals) == 0 {
			continue
		}
		kl := strings.ToLower(k)
		if kl == "subscription-userinfo" || strings.HasSuffix(kl, "-subscription-userinfo") {
			if info, ok := ParseSubscriptionUserInfo(vals[0]); ok {
				return info, true
			}
		}
	}
	return SubscriptionUserInfo{}, false
}

// MergeSubscriptionUserInfo 合并多个上游流量（用于对外分享订阅）
//
// 策略：
//   - upload / download / total：求和（多机场套餐独立累加）
//   - expire：取非零中的最小值（最早到期，对客户端最保守）
//   - 全 0 的源不参与 expire 竞选
//
// 注意：total=0 的源若仍有 used，合计 total 仍可能偏小；展示侧应允许 used>total。
func MergeSubscriptionUserInfo(items []SubscriptionUserInfo) SubscriptionUserInfo {
	var out SubscriptionUserInfo
	var expireMin int64
	hasExpire := false
	for _, it := range items {
		if it.Upload > 0 {
			out.Upload += it.Upload
		}
		if it.Download > 0 {
			out.Download += it.Download
		}
		if it.Total > 0 {
			out.Total += it.Total
		}
		if it.Expire > 0 {
			if !hasExpire || it.Expire < expireMin {
				expireMin = it.Expire
				hasExpire = true
			}
		}
	}
	if hasExpire {
		out.Expire = expireMin
	}
	return out
}

// FormatSubscriptionUserInfoHeader 生成响应头值
func FormatSubscriptionUserInfoHeader(u SubscriptionUserInfo) string {
	return "upload=" + strconv.FormatInt(u.Upload, 10) +
		"; download=" + strconv.FormatInt(u.Download, 10) +
		"; total=" + strconv.FormatInt(u.Total, 10) +
		"; expire=" + strconv.FormatInt(u.Expire, 10)
}
