package source

import (
	"regexp"
	"strings"

	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/regioncatalog"
)

// 机场常见「说明/统计」假节点名（已入库的也会标异常）
var infoNamePattern = regexp.MustCompile(
	`(?i)剩余流量|套餐到期|流量|到期|过期|官网|电报|重置|距离下次|消耗|续费|客服|公告|测试|过滤掉|过滤了|已过滤|到期时间|可用流量|套餐流量|距离.*重置`,
)

// IsInfoNodeName 判断节点名是否为机场「说明/统计」类假节点（非线路）。
// 供刷新流程直接判定，避免对 AssessProxy 返回文案做字符串匹配。
func IsInfoNodeName(name string) bool {
	return infoNamePattern.MatchString(strings.TrimSpace(name))
}

// AssessProxy 判断节点是否正常，并给出原因（展示用，不改库）
func AssessProxy(name, region, typ, server string, port int) (ok bool, issue string) {
	name = strings.TrimSpace(name)
	region = strings.ToUpper(strings.TrimSpace(region))
	typ = strings.TrimSpace(typ)
	server = strings.TrimSpace(server)

	if name == "" {
		return false, "缺少名称"
	}
	if IsInfoNodeName(name) {
		return false, "疑似信息节点（非线路）"
	}
	if typ == "" {
		return false, "缺少协议类型"
	}
	if server == "" {
		return false, "缺少服务器地址"
	}
	if port < 1 || port > 65535 {
		return false, "端口无效"
	}
	// 常见无效/占位 server
	ls := strings.ToLower(server)
	if ls == "127.0.0.1" || ls == "0.0.0.0" || ls == "localhost" || ls == "null" || ls == "none" {
		return false, "服务器地址无效"
	}
	if regioncatalog.IsFallback(region) {
		// 仍可能是可用线路，但地区未识别
		return false, "地区未识别（UNK）"
	}
	return true, ""
}

func toProxyNode(id, sourceID uint, name, region, typ, server string, port int, enabled bool) common.ProxyNode {
	ok, issue := AssessProxy(name, region, typ, server, port)
	return common.ProxyNode{
		ID:       id,
		SourceID: sourceID,
		Name:     name,
		Region:   common.Region(region),
		Type:     typ,
		Server:   server,
		Port:     port,
		Enabled:  enabled,
		OK:       ok,
		Issue:    issue,
	}
}
