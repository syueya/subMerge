package rule

import (
	"fmt"
	"strings"

	common "github.com/submerge/submerge/backend/common"
)

// parsedImportLine 解析后的一条导入规则
type parsedImportLine struct {
	Type     string
	Payload  string
	Target   string
	Note     string
	Category string
	LineNo   int
}

// parseBatchImportText 解析批量导入文本。
// 支持：
//
//	category,TYPE,payload,target
//	category,TYPE,payload,target,note
//	category,TYPE,payload          （出口用 defaultTarget）
//	category,MATCH,,target[,note]
//	category,MATCH,target[,note]
//	payload                        （使用 defaultType / defaultTarget / defaultNote / defaultCategory）
//
// 空行与 # 开头注释忽略；行尾 # 备注也可。
// 分类可留空（,DOMAIN-SUFFIX,...）以使用 defaultCategory。
func parseBatchImportText(text, defaultType, defaultTarget, defaultNote, defaultCategory string) (ok []parsedImportLine, errs []string) {
	defaultType = strings.TrimSpace(defaultType)
	if defaultType == "" {
		defaultType = string(common.RuleTypeDomainSuffix)
	}
	defaultTarget = strings.TrimSpace(defaultTarget)
	defaultNote = strings.TrimSpace(defaultNote)
	defaultCategory = strings.TrimSpace(defaultCategory)

	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for i, raw := range lines {
		lineNo := i + 1
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// 行尾注释：a,b,c # note  —— 仅当 # 前有空白时拆
		inlineNote := ""
		if idx := strings.Index(line, " #"); idx >= 0 {
			inlineNote = strings.TrimSpace(line[idx+2:])
			line = strings.TrimSpace(line[:idx])
		}

		parts := splitCSVLike(line)
		if len(parts) == 0 {
			continue
		}

		var typ, payload, target, note, category string
		switch {
		case len(parts) == 1:
			// 仅 payload
			if defaultTarget == "" {
				errs = append(errs, fmt.Sprintf("第%d行：仅写匹配内容时需指定默认出口", lineNo))
				continue
			}
			typ = defaultType
			payload = parts[0]
			target = defaultTarget
			note = defaultNote
			category = defaultCategory
		case len(parts) == 2:
			// category,MATCH 不完整；TYPE,payload 也不再支持
			errs = append(errs, fmt.Sprintf("第%d行：格式应为 分类,类型,匹配内容,出口[,备注]", lineNo))
			continue
		case len(parts) == 3:
			// category,TYPE,payload  （缺 target 用默认）
			// 或 category,MATCH,target
			category = parts[0]
			if isKnownRuleType(parts[1]) {
				typ = normalizeRuleType(parts[1])
				if strings.EqualFold(typ, string(common.RuleTypeMatch)) {
					payload = ""
					target = parts[2]
					note = defaultNote
				} else {
					if defaultTarget == "" {
						errs = append(errs, fmt.Sprintf("第%d行：缺少出口，且未设置默认出口", lineNo))
						continue
					}
					payload = parts[2]
					target = defaultTarget
					note = defaultNote
				}
			} else {
				errs = append(errs, fmt.Sprintf("第%d行：格式应为 分类,类型,匹配内容,出口[,备注]", lineNo))
				continue
			}
		case len(parts) == 4:
			// category,TYPE,payload,target
			// 或 category,MATCH,,target / category,MATCH,target,note
			category = parts[0]
			if !isKnownRuleType(parts[1]) {
				errs = append(errs, fmt.Sprintf("第%d行：未知规则类型 %q", lineNo, parts[1]))
				continue
			}
			typ = normalizeRuleType(parts[1])
			if strings.EqualFold(typ, string(common.RuleTypeMatch)) {
				// category,MATCH,,target  或 category,MATCH,target,note
				if parts[2] == "" {
					payload = ""
					target = parts[3]
					note = defaultNote
				} else {
					payload = ""
					target = parts[2]
					note = parts[3]
				}
			} else {
				payload = parts[2]
				target = parts[3]
				note = defaultNote
			}
		default:
			// category,TYPE,payload,target,note
			// note 中若含逗号则 parts[4:] 拼回
			category = parts[0]
			if !isKnownRuleType(parts[1]) {
				errs = append(errs, fmt.Sprintf("第%d行：未知规则类型 %q", lineNo, parts[1]))
				continue
			}
			typ = normalizeRuleType(parts[1])
			payload = parts[2]
			target = parts[3]
			note = strings.Join(parts[4:], ",")
			if strings.EqualFold(typ, string(common.RuleTypeMatch)) {
				// category,MATCH,,target,note —— payload 应为空
				if payload != "" && target == "" {
					// category,MATCH,target,note... 已在 4 段处理；5+ 段时 payload 可能误填
					target = payload
				}
				payload = ""
			}
		}

		if inlineNote != "" {
			if note != "" {
				note = note + " " + inlineNote
			} else {
				note = inlineNote
			}
		}

		typ = strings.TrimSpace(typ)
		payload = strings.TrimSpace(payload)
		target = strings.TrimSpace(target)
		note = strings.TrimSpace(note)
		category = strings.TrimSpace(category)
		if category == "" {
			category = defaultCategory
		}

		if err := validateRule(typ, payload, target); err != nil {
			errs = append(errs, fmt.Sprintf("第%d行：%s", lineNo, err.Error()))
			continue
		}

		ok = append(ok, parsedImportLine{
			Type:     typ,
			Payload:  payload,
			Target:   target,
			Note:     note,
			Category: category,
			LineNo:   lineNo,
		})
	}
	return ok, errs
}

// isKnownRuleType 是否为支持的 Clash 规则类型名
func isKnownRuleType(s string) bool {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "DOMAIN", "DOMAIN-SUFFIX", "DOMAIN-KEYWORD",
		"GEOSITE", "GEOIP", "IP-CIDR", "IP-CIDR6",
		"SRC-IP-CIDR", "SRC-PORT", "DST-PORT",
		"PROCESS-NAME", "PROCESS-PATH", "RULE-SET", "MATCH":
		return true
	default:
		return false
	}
}

func normalizeRuleType(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// splitCSVLike 按逗号分割，保留空字段；不做引号转义（规则 payload 一般无逗号）。
func splitCSVLike(s string) []string {
	raw := strings.Split(s, ",")
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		out = append(out, strings.TrimSpace(p))
	}
	// 去掉尾部全空（避免 "a,b," 多一段），但保留中间空
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return out
}
