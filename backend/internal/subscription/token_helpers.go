package subscription

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/apiresp"
	"github.com/submerge/submerge/backend/internal/crypto"
	"github.com/submerge/submerge/backend/internal/database"
)

func (s *Service) findActiveToken(rawToken string) (database.ShareToken, error) {
	if rawToken == "" {
		return database.ShareToken{}, fmt.Errorf("missing token")
	}
	hash := crypto.HashToken(rawToken)
	var row database.ShareToken
	if err := s.db.Where("token_hash = ?", hash).First(&row).Error; err != nil {
		return database.ShareToken{}, fmt.Errorf("invalid token")
	}
	switch row.Status {
	case string(common.TokenStatusActive):
		return row, nil
	case string(common.TokenStatusRevoked):
		return database.ShareToken{}, fmt.Errorf("token revoked")
	default:
		return database.ShareToken{}, fmt.Errorf("token disabled")
	}
}

func (s *Service) toView(r database.ShareToken, nameByID map[uint]string, allowPlain bool) common.ShareToken {
	plain := ""
	if allowPlain && r.TokenEncrypted != "" && s.box != nil {
		if p, err := s.box.Decrypt(r.TokenEncrypted); err == nil {
			plain = p
		}
	}

	masked := r.TokenPrefix + "****"
	if plain != "" {
		masked = plain
	}

	sourceIDs := decodeSourceIDs(r.SourceIDsJSON)
	sourceNames := make([]string, 0, len(sourceIDs))
	for _, id := range sourceIDs {
		if n, ok := nameByID[id]; ok && n != "" {
			sourceNames = append(sourceNames, n)
		} else {
			sourceNames = append(sourceNames, fmt.Sprintf("#%d(已失效)", id))
		}
	}

	mode := common.NormalizeTokenGroupMode(common.TokenGroupMode(r.GroupMode))
	groupNames := decodeGroupNames(r.GroupNamesJSON)
	v := common.ShareToken{
		ID:          r.ID,
		Name:        r.Name,
		Token:       plain,
		TokenMasked: masked,
		Status:      common.TokenStatus(r.Status),
		SourceIDs:   sourceIDs,
		SourceNames: sourceNames,
		GroupMode:   mode,
		GroupNames:  groupNames,
		AccessCount: r.AccessCount,
		CreatedAt:   apiresp.FormatRFC3339(&r.CreatedAt),
		UpdatedAt:   apiresp.FormatRFC3339(&r.UpdatedAt),
	}
	if r.LastAccessAt != nil {
		ts := apiresp.FormatRFC3339(r.LastAccessAt)
		v.LastAccessAt = &ts
	}
	if plain != "" {
		v.SubscribeURL = s.baseURLSnapshot() + "/subscribe/" + plain
	}
	return v
}

func (s *Service) sourceNameMap() (map[uint]string, error) {
	var rows []database.Source
	if err := s.db.Select("id", "name").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uint]string, len(rows))
	for _, r := range rows {
		out[r.ID] = r.Name
	}
	return out, nil
}

// normalizeSourceIDs 去重、校验存在；空输入表示全部源
func (s *Service) normalizeSourceIDs(ids []uint) ([]uint, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	seen := map[uint]struct{}{}
	clean := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		clean = append(clean, id)
	}
	if len(clean) == 0 {
		return nil, nil
	}
	var count int64
	if err := s.db.Model(&database.Source{}).Where("id IN ?", clean).Count(&count).Error; err != nil {
		return nil, err
	}
	if int(count) != len(clean) {
		return nil, fmt.Errorf("%w: one or more source ids not found", ErrInvalidTokenConfig)
	}
	sort.Slice(clean, func(i, j int) bool { return clean[i] < clean[j] })
	return clean, nil
}

func encodeSourceIDs(ids []uint) (string, error) {
	if len(ids) == 0 {
		return "", nil
	}
	b, err := json.Marshal(ids)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func decodeSourceIDs(raw string) []uint {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" || raw == "[]" {
		return []uint{}
	}
	var ids []uint
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return []uint{}
	}
	// 规范化展示顺序
	seen := map[uint]struct{}{}
	out := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// normalizeGroupNames custom 模式校验组名存在；其它模式清空白名单
func (s *Service) normalizeGroupNames(mode common.TokenGroupMode, names []string) ([]string, error) {
	if mode != common.TokenGroupModeCustom {
		return nil, nil
	}
	seen := map[string]struct{}{}
	clean := make([]string, 0, len(names))
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		clean = append(clean, n)
	}
	if len(clean) == 0 {
		return nil, fmt.Errorf("%w: custom group mode requires at least one group name", ErrInvalidTokenConfig)
	}
	var count int64
	if err := s.db.Model(&database.ProxyGroup{}).Where("name IN ?", clean).Count(&count).Error; err != nil {
		return nil, err
	}
	if int(count) != len(clean) {
		return nil, fmt.Errorf("%w: one or more group names not found", ErrInvalidTokenConfig)
	}
	sort.Strings(clean)
	return clean, nil
}

func encodeGroupNames(names []string) (string, error) {
	if len(names) == 0 {
		return "", nil
	}
	b, err := json.Marshal(names)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func decodeGroupNames(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" || raw == "[]" {
		return []string{}
	}
	var names []string
	if err := json.Unmarshal([]byte(raw), &names); err != nil {
		return []string{}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(names))
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func tokenPrefix(token string) string {
	if len(token) >= 4 {
		return token[:4]
	}
	return token
}
