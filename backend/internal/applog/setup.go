package applog

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	// 嵌入 IANA 时区数据，保证精简镜像 / Windows 也能 LoadLocation("Asia/Shanghai")
	_ "time/tzdata"
)

const (
	filePrefix = "submerge"
	fileLayout = "2006-01-02" // submerge-YYYY-MM-DD.log
	// DefaultTimezone 日志时间戳与按日切分的默认时区
	DefaultTimezone = "Asia/Shanghai"
)

var (
	mu       sync.Mutex
	writer   *dailyWriter
	output   string // console | file | both | none
	retainD  int
	stopOnce sync.Once
	stopCh   chan struct{}
	// loc 进程日志时区；InitTimezone / Setup 后非 nil
	loc atomic.Pointer[time.Location]
)

// InitTimezone 设置进程本地时区（影响标准库 log 时间戳、按日日志文件名与保留清理）。
// 优先环境变量 TZ；未设置时默认 Asia/Shanghai。
// 可在 Setup 之前单独调用；Setup 也会确保调用一次。
func InitTimezone() error {
	name := strings.TrimSpace(os.Getenv("TZ"))
	if name == "" {
		name = DefaultTimezone
	}
	loaded, err := time.LoadLocation(name)
	if err != nil {
		// 默认上海：加载失败时退回固定 UTC+8，避免日志落到 UTC
		if name == DefaultTimezone || name == "Asia/Chongqing" || name == "PRC" {
			loaded = time.FixedZone("CST", 8*3600)
		} else {
			return fmt.Errorf("load timezone %q: %w", name, err)
		}
	}
	time.Local = loaded
	loc.Store(loaded)
	return nil
}

// Location 当前日志时区（未初始化时返回 Asia/Shanghai 或 FixedZone 兜底）
func Location() *time.Location {
	if l := loc.Load(); l != nil {
		return l
	}
	loaded, err := time.LoadLocation(DefaultTimezone)
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loaded
}

// Setup 配置标准库 log 的输出目标。
// outputMode: console（默认）| file | both | none
// logDir: 按日分文件目录；file/both 时使用
// retentionDays: 保留天数；<=0 表示不自动清理
// 时间戳与按日切分使用 InitTimezone 确定的本地时区（默认上海）
func Setup(outputMode, logDir string, retentionDays int) error {
	// 先定好时区，再写文件 / 设 log flags
	if err := InitTimezone(); err != nil {
		return err
	}

	mode := normalizeOutput(outputMode)
	if (mode == "file" || mode == "both") && strings.TrimSpace(logDir) == "" {
		return fmt.Errorf("log dir is required when LOG_OUTPUT=%s", mode)
	}

	// 在持锁外创建 writer，避免 rotate/Location 与 mu 交叉
	var newWriter *dailyWriter
	var writers []io.Writer
	switch mode {
	case "none":
		// discard
	case "file":
		w, err := newDailyWriter(logDir)
		if err != nil {
			return err
		}
		newWriter = w
		writers = append(writers, w)
	case "both":
		w, err := newDailyWriter(logDir)
		if err != nil {
			return err
		}
		newWriter = w
		writers = append(writers, os.Stderr, w)
	default: // console
		writers = append(writers, os.Stderr)
	}

	mu.Lock()
	// 关闭旧 writer / 后台清理
	if stopCh != nil {
		select {
		case <-stopCh:
		default:
			close(stopCh)
		}
		stopCh = nil
	}
	oldWriter := writer
	writer = newWriter
	stopOnce = sync.Once{}
	output = mode
	retainD = retentionDays
	if len(writers) == 0 {
		log.SetOutput(io.Discard)
	} else if len(writers) == 1 {
		log.SetOutput(writers[0])
	} else {
		log.SetOutput(io.MultiWriter(writers...))
	}
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)

	var retentionStop chan struct{}
	if writer != nil && retentionDays > 0 {
		retentionStop = make(chan struct{})
		stopCh = retentionStop
	}
	mu.Unlock()

	if oldWriter != nil {
		_ = oldWriter.Close()
	}

	// 启动时清理 + 定时清理（仅写文件模式）；Info 在持锁外调用
	if newWriter != nil && retentionDays > 0 {
		if n, err := cleanOldLogs(logDir, retentionDays); err == nil && n > 0 {
			Info("[applog] cleaned %d log file(s) older than %d day(s) in %s", n, retentionDays, logDir)
		}
		go retentionLoop(logDir, retentionDays, retentionStop)
	}
	return nil
}

// Output 当前输出模式
func Output() string {
	mu.Lock()
	defer mu.Unlock()
	if output == "" {
		return "console"
	}
	return output
}

// Close 关闭日志文件与后台清理
func Close() {
	mu.Lock()
	if stopCh != nil {
		select {
		case <-stopCh:
		default:
			close(stopCh)
		}
		stopCh = nil
	}
	old := writer
	writer = nil
	mu.Unlock()
	if old != nil {
		_ = old.Close()
	}
}

func retentionLoop(dir string, days int, stop <-chan struct{}) {
	// 每 6 小时扫一次即可
	t := time.NewTicker(6 * time.Hour)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			if n, err := cleanOldLogs(dir, days); err == nil && n > 0 {
				Info("[applog] cleaned %d log file(s) older than %d day(s) in %s", n, days, dir)
			}
		}
	}
}

// cleanOldLogs 删除 dir 下早于 retentionDays 的 submerge-YYYY-MM-DD.log
func cleanOldLogs(dir string, retentionDays int) (int, error) {
	if retentionDays <= 0 {
		return 0, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	// 保留「今天往前 retentionDays-1 天」共 retentionDays 个自然日（按时区日历日）
	// 例：保留 7 天 → 删除 cutoff 日 0 点之前的文件
	zone := Location()
	now := time.Now().In(zone)
	// Truncate 按 UTC 对齐，故用年月日重建本地 0 点
	cutoff := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, zone).
		AddDate(0, 0, -(retentionDays - 1))
	deleted := 0
	prefix := filePrefix + "-"
	suffix := ".log"
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
			continue
		}
		datePart := strings.TrimSuffix(strings.TrimPrefix(name, prefix), suffix)
		day, err := time.ParseInLocation(fileLayout, datePart, zone)
		if err != nil {
			continue
		}
		if day.Before(cutoff) {
			if err := os.Remove(filepath.Join(dir, name)); err != nil {
				continue
			}
			deleted++
		}
	}
	return deleted, nil
}

// dailyWriter 按配置时区（默认上海）日期写入 submerge-YYYY-MM-DD.log
type dailyWriter struct {
	dir  string
	mu   sync.Mutex
	day  string
	file *os.File
}

func newDailyWriter(dir string) (*dailyWriter, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir %q: %w", dir, err)
	}
	w := &dailyWriter{dir: dir}
	if err := w.rotate(time.Now()); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *dailyWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := time.Now()
	day := now.In(Location()).Format(fileLayout)
	if w.file == nil || day != w.day {
		if err := w.rotate(now); err != nil {
			return 0, err
		}
	}
	return w.file.Write(p)
}

func (w *dailyWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	w.day = ""
	return err
}

func (w *dailyWriter) rotate(now time.Time) error {
	day := now.In(Location()).Format(fileLayout)
	path := filepath.Join(w.dir, fmt.Sprintf("%s-%s.log", filePrefix, day))
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file %q: %w", path, err)
	}
	if w.file != nil {
		_ = w.file.Close()
	}
	w.file = f
	w.day = day
	return nil
}

func normalizeOutput(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "console", "stdout", "stderr", "std":
		return "console"
	case "file":
		return "file"
	case "both", "all", "console+file", "file+console":
		return "both"
	case "none", "off", "disable", "disabled":
		return "none"
	default:
		return "console"
	}
}
