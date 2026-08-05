package applog

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestLevelPrefixes(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(nil)
		log.SetFlags(log.LstdFlags)
		SetDebugEnabled(false)
	}()

	// DEBUG 默认关闭
	Debug("hidden")
	if buf.Len() != 0 {
		t.Fatalf("disabled debug wrote output: %q", buf.String())
	}

	SetDebugEnabled(true)
	Info("hello %s", "world")
	Warn("careful")
	Error("boom")
	Debug("detail")

	out := buf.String()
	for _, want := range []string{"[INFO] hello world", "[WARN] careful", "[ERROR] boom", "[DEBUG] detail"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	// 每条日志后空一行（内容\n + log.Output 再 \n）
	if !strings.Contains(out, "[INFO] hello world\n\n[WARN] careful\n\n") {
		t.Fatalf("expected blank line between entries, got %q", out)
	}

	// 不重复叠前缀
	buf.Reset()
	Info("[INFO] already")
	if strings.Count(buf.String(), "[INFO]") != 1 {
		t.Fatalf("double prefix: %q", buf.String())
	}
}
