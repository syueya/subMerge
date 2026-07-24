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
//	TYPE,payload,target
//	TYPE,payload,target,note
//	TYPE,payload,target,note,category
//	MATCH,target
//	MATCH,,target[,note[,category]]
//	payload                 （使用 defaultType / defaultTarget / defaultNote / defaultCategory）
//
// 空行与 # 开头注释忽略；行尾 # 备注也可。
// 兼容旧 4 列（无 category）：用 defaultCategory。
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
		// 行尾注释：TYPE,a,b # note  —— 仅当 # 前有空白时拆
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
			// MATCH,target 或 TYPE,payload（缺 target 时用默认）
			if strings.EqualFold(parts[0], string(common.RuleTypeMatch)) {
				typ = string(common.RuleTypeMatch)
				payload = ""
				target = parts[1]
				category = defaultCategory
			} else if defaultTarget != "" {
				typ = parts[0]
				payload = parts[1]
				target = defaultTarget
				note = defaultNote
				category = defaultCategory
			} else {
				errs = append(errs, fmt.Sprintf("第%d行：格式应为 TYPE,payload,target[,note[,category]]", lineNo))
				continue
			}
		case len(parts) == 3:
			typ = parts[0]
			payload = parts[1]
			target = parts[2]
			note = defaultNote
			category = defaultCategory
			if strings.EqualFold(typ, string(common.RuleTypeMatch)) {
				payload = ""
			}
		case len(parts) == 4:
			// TYPE,payload,target,note  （兼容旧格式；category 用默认）
			typ = parts[0]
			payload = parts[1]
			target = parts[2]
			note = parts[3]
			category = defaultCategory
			if strings.EqualFold(typ, string(common.RuleTypeMatch)) {
				payload = ""
			}
		default:
			// TYPE,payload,target,note,category  （category 取第 5 段；note 中若含逗号则 parts[3:len-1] 拼回）
			typ = parts[0]
			payload = parts[1]
			target = parts[2]
			if len(parts) == 5 {
				note = parts[3]
				category = parts[4]
			} else {
				// 多于 5 段：最后一段为 category，中间拼 note
				category = parts[len(parts)-1]
				note = strings.Join(parts[3:len(parts)-1], ",")
			}
			if strings.EqualFold(typ, string(common.RuleTypeMatch)) {
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
