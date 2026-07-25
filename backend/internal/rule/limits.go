package rule

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	maxBatchIDs         = 1000
	maxReorderIDs       = 10000
	maxBatchImportBytes = 8 << 20
	maxBatchImportLines = 20000
	maxBatchImportLine  = 4096
	maxImportErrors     = 1000
)

func validateBatchIDs(ids []uint, limit int) error {
	seen := make(map[uint]struct{}, len(ids))
	count := 0
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		count++
		if count > limit {
			return fmt.Errorf("too many ids; maximum is %d", limit)
		}
	}
	return nil
}

func validateBatchImportText(text string) error {
	if len(text) > maxBatchImportBytes {
		return fmt.Errorf("import text exceeds %d bytes", maxBatchImportBytes)
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	nonEmpty := 0
	for _, raw := range lines {
		if len(raw) > maxBatchImportLine {
			return fmt.Errorf("import line exceeds %d bytes", maxBatchImportLine)
		}
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		nonEmpty++
		if nonEmpty > maxBatchImportLines {
			return fmt.Errorf("too many import lines; maximum is %d", maxBatchImportLines)
		}
	}
	return nil
}

func appendImportError(errors []string, message string) []string {
	if len(errors) >= maxImportErrors {
		return errors
	}
	return append(errors, message)
}

func validateTextField(name, value string, max int) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("%s must be valid UTF-8", name)
	}
	if len([]rune(strings.TrimSpace(value))) > max {
		return fmt.Errorf("%s too long", name)
	}
	return nil
}
