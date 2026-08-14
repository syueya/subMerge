// Package version 提供应用版本号。
//
// 发版唯一来源是仓库根目录 VERSION。构建流程通过 -ldflags 将该值写入
// Value，因此运行中的二进制不依赖外部版本文件，也不允许环境变量覆盖。
package version

// Value is set by release and development build commands with:
//
//	-X github.com/submerge/submerge/backend/version.Value=<VERSION>
//
// A direct bare go build is intentionally identifiable as a development build.
var Value = "0.0.0-dev"

// String 返回编入当前二进制的应用版本。
func String() string {
	return Value
}
