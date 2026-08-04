// Package version 提供应用版本号。
//
// 发版唯一来源：frontend/version.ts
// 本目录 VERSION 由 CI 在 go build 前从 frontend 同步；本地开发可手动同步或与 version.ts 保持一致。
// 不要用环境变量覆盖。
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
