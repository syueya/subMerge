package publish

import (
	"fmt"
	"github.com/submerge/submerge/backend/internal/database"
	"strings"
)

func formatRule(r database.Rule) string {
	typ := strings.TrimSpace(r.Type)
	payload := strings.TrimSpace(r.Payload)
	target := strings.TrimSpace(r.Target)
	if typ == "" || target == "" {
		return ""
	}
	if typ == "MATCH" {
		return fmt.Sprintf("MATCH,%s", target)
	}
	if payload == "" {
		return ""
	}
	return fmt.Sprintf("%s,%s,%s", typ, payload, target)
}
