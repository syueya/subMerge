package logs

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/applog"
)

const (
	filePrefix = "submerge"
	fileLayout = "2006-01-02"
	// 单文件读取上限，防止超大日志占满内存（约 8MiB 尾部）
	maxTailBytes = 8 << 20
	// 行数硬上限（与前端下拉一致再留余量）
	maxLines = 2000
)

// 标准库 log 行：2006/01/02 15:04:05.000000 file.go:12: [LEVEL] message
// 微秒可选；级别前缀可选（Gin 等其它源可能没有）
var entryHeadRe = regexp.MustCompile(
	`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}(?:\.\d+)?)\s+([^:]+\.go:\d+):\s+(?:\[(INFO|WARN|ERROR|DEBUG|info|warn|error|debug)\]\s+)?(.*)$`,
)

// Service 读取 applog 按日文件
type Service struct {
	dir string
}

func NewService(logDir string) *Service {
	return &Service{dir: filepath.Clean(logDir)}
}

// List 列出 log 目录下 submerge-YYYY-MM-DD.log，按日期倒序；name 为文件名子串过滤（可选）
func (s *Service) List(nameFilter string) (common.LogFileListResponse, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return common.LogFileListResponse{Files: []common.LogFileInfo{}}, nil
		}
		return common.LogFileListResponse{}, err
	}

	filter := strings.TrimSpace(strings.ToLower(nameFilter))
	files := make([]common.LogFileInfo, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !isAllowedLogName(name) {
			continue
		}
		if filter != "" && !strings.Contains(strings.ToLower(name), filter) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, common.LogFileInfo{
			Name:      name,
			Size:      info.Size(),
			UpdatedAt: info.ModTime().UnixMilli(),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		// 文件名日期倒序；同日再比 mtime
		if files[i].Name != files[j].Name {
			return files[i].Name > files[j].Name
		}
		return files[i].UpdatedAt > files[j].UpdatedAt
	})
	return common.LogFileListResponse{Files: files}, nil
}

// Details 读取指定日志文件尾部 line 条解析后的条目（时间倒序，新→旧）
func (s *Service) Details(name string, line int) (common.LogDetailsResponse, error) {
	safe, err := s.resolvePath(name)
	if err != nil {
		return common.LogDetailsResponse{}, err
	}
	if line <= 0 {
		line = 100
	}
	if line > maxLines {
		line = maxLines
	}

	raw, err := readTail(safe, maxTailBytes)
	if err != nil {
		return common.LogDetailsResponse{}, err
	}
	// 去掉可能截断的首行半截
	if len(raw) == maxTailBytes {
		if i := bytes.IndexByte(raw, '\n'); i >= 0 && i+1 < len(raw) {
			raw = raw[i+1:]
		}
	}

	entries := parseEntries(string(raw), applog.Location())
	if len(entries) > line {
		entries = entries[len(entries)-line:]
	}
	sort.SliceStable(entries, func(i, j int) bool {
		// 无法解析时间的孤立行放在末尾；其余条目按最新时间优先。
		if entries[i].Timestamp == 0 {
			return false
		}
		if entries[j].Timestamp == 0 {
			return true
		}
		return entries[i].Timestamp > entries[j].Timestamp
	})
	return common.LogDetailsResponse{Items: entries}, nil
}

func (s *Service) resolvePath(name string) (string, error) {
	name = filepath.Base(strings.TrimSpace(name))
	if !isAllowedLogName(name) {
		return "", fmt.Errorf("invalid log file name")
	}
	full := filepath.Join(s.dir, name)
	// 二次确认仍在目录内（防 symlink / 异常 Base 行为）
	cleanDir := s.dir
	cleanFull := filepath.Clean(full)
	rel, err := filepath.Rel(cleanDir, cleanFull)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid log file path")
	}
	st, err := os.Stat(cleanFull)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("log file not found")
		}
		return "", err
	}
	if st.IsDir() {
		return "", fmt.Errorf("invalid log file name")
	}
	return cleanFull, nil
}

func isAllowedLogName(name string) bool {
	if name != filepath.Base(name) || strings.Contains(name, "..") {
		return false
	}
	if !strings.HasPrefix(name, filePrefix+"-") || !strings.HasSuffix(name, ".log") {
		return false
	}
	datePart := strings.TrimSuffix(strings.TrimPrefix(name, filePrefix+"-"), ".log")
	_, err := time.Parse(fileLayout, datePart)
	return err == nil
}

// readTail 读取文件末尾最多 maxBytes 字节
func readTail(path string, maxBytes int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := st.Size()
	if size <= 0 {
		return nil, nil
	}
	var start int64
	readSize := size
	if size > maxBytes {
		start = size - maxBytes
		readSize = maxBytes
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return nil, err
	}
	buf := make([]byte, readSize)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	return buf[:n], nil
}

func parseEntries(text string, zone *time.Location) []common.LogEntry {
	if zone == nil {
		zone = time.Local
	}
	sc := bufio.NewScanner(strings.NewReader(text))
	// 单行可能很长（region samples），放宽 buffer
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var out []common.LogEntry
	var cur *common.LogEntry

	flush := func() {
		if cur == nil {
			return
		}
		cur.Content = strings.TrimRight(cur.Content, "\r\n")
		out = append(out, *cur)
		cur = nil
	}

	for sc.Scan() {
		line := sc.Text()
		// 条目之间的空行：结束当前条目
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		if m := entryHeadRe.FindStringSubmatch(line); m != nil {
			flush()
			ts := parseLogTimestamp(m[1], zone)
			level := strings.ToLower(m[3])
			if level == "" {
				level = "info"
			}
			cur = &common.LogEntry{
				Timestamp: ts,
				Caller:    m[2],
				Level:     level,
				Content:   m[4],
			}
			continue
		}
		// 续行挂到上一条
		if cur != nil {
			if cur.Content == "" {
				cur.Content = line
			} else {
				cur.Content += "\n" + line
			}
			continue
		}
		// 孤立行（文件截断半截后的残留）
		out = append(out, common.LogEntry{
			Timestamp: 0,
			Caller:    "",
			Level:     "info",
			Content:   line,
		})
	}
	flush()
	_ = sc.Err()
	return out
}

func parseLogTimestamp(s string, zone *time.Location) int64 {
	s = strings.TrimSpace(s)
	layouts := []string{
		"2006/01/02 15:04:05.999999999",
		"2006/01/02 15:04:05.999999",
		"2006/01/02 15:04:05.999",
		"2006/01/02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, zone); err == nil {
			return t.UnixMilli()
		}
	}
	// 微秒位数不固定时：拆开再拼
	parts := strings.SplitN(s, ".", 2)
	if len(parts) == 2 {
		base := parts[0]
		frac := parts[1]
		// 补齐/截断到纳秒
		if len(frac) > 9 {
			frac = frac[:9]
		}
		for len(frac) < 9 {
			frac += "0"
		}
		if t, err := time.ParseInLocation("2006/01/02 15:04:05", base, zone); err == nil {
			nsec, _ := strconv.ParseInt(frac, 10, 64)
			return t.Add(time.Duration(nsec)).UnixMilli()
		}
	}
	return 0
}
