// Package version 提供应用版本号（来自同目录 VERSION 文件，go:embed）。
// 发版时只改 VERSION 文件即可，不要用环境变量覆盖。
package version

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var raw string

// String 返回当前应用版本（已 trim）
func String() string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "0.0.0-dev"
	}
	return v
}
