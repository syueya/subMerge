package applog

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync/atomic"
)

// 日志级别前缀（统一大写，便于检索 / 高亮）
const (
	prefixInfo  = "[INFO] "
	prefixWarn  = "[WARN] "
	prefixError = "[ERROR] "
	prefixDebug = "[DEBUG] "
)

var debugEnabled atomic.Bool

// SetDebugEnabled 控制详细 DEBUG 日志（地区识别样本、过滤明细等）。
func SetDebugEnabled(enabled bool) {
	debugEnabled.Store(enabled)
}

// Info 常规运行信息
func Info(format string, args ...any) {
	writeLevel(2, prefixInfo, format, args...)
}

// Warn 可恢复异常 / 部分失败
func Warn(format string, args ...any) {
	writeLevel(2, prefixWarn, format, args...)
}

// Error 错误（进程通常继续）
func Error(format string, args ...any) {
	writeLevel(2, prefixError, format, args...)
}

// Debug 明细（过滤清单、样本等）
func Debug(format string, args ...any) {
	if !debugEnabled.Load() {
		return
	}
	writeLevel(2, prefixDebug, format, args...)
}

// Fatalf 致命错误并退出（带 [ERROR] 前缀）
func Fatalf(format string, args ...any) {
	writeLevel(2, prefixError, format, args...)
	os.Exit(1)
}

func writeLevel(calldepth int, levelPrefix, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	// 去掉调用方误带的尾部换行，再统一补空行
	msg = strings.TrimRight(msg, "\r\n")
	// log.Output：若字符串已以 \n 结尾则不再追加；以 \n\n 结尾 → 条目后空一行，阅读不挤
	// 避免调用方已手写 [INFO] 再叠一层
	if hasLevelPrefix(msg) {
		_ = log.Output(calldepth+1, msg+"\n\n")
		return
	}
	_ = log.Output(calldepth+1, levelPrefix+msg+"\n\n")
}

func hasLevelPrefix(msg string) bool {
	s := strings.TrimSpace(msg)
	for _, p := range []string{"[INFO]", "[WARN]", "[ERROR]", "[DEBUG]", "[info]", "[warn]", "[error]", "[debug]"} {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
