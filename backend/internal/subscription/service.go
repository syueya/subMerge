package subscription

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/applog"
	"github.com/submerge/submerge/backend/internal/crypto"
	"github.com/submerge/submerge/backend/internal/database"
	"github.com/submerge/submerge/backend/internal/publish"
	"github.com/submerge/submerge/backend/internal/source"
	"gorm.io/gorm"
)

// ErrInvalidTokenConfig 令牌配置非法（源 ID / 策略组名不存在，或 custom 模式缺组名）。
// handler 用 errors.Is 判定为客户端错误（400），而非字符串匹配错误消息。
var ErrInvalidTokenConfig = errors.New("invalid token config")

// Service 订阅链接与订阅下发
type Service struct {
	db      *gorm.DB
	publish *publish.Service
	box     *crypto.Box
	baseURL string
}

func NewService(db *gorm.DB, publishSvc *publish.Service, box *crypto.Box, baseURL string) *Service {
	return &Service{db: db, publish: publishSvc, box: box, baseURL: baseURL}
}

func (s *Service) List() (common.TokenListResponse, error) {
	var rows []database.ShareToken
	if err := s.db.Order("id desc").Find(&rows).Error; err != nil {
		return common.TokenListResponse{}, err
	}
	nameByID, err := s.sourceNameMap()
	if err != nil {
		return common.TokenListResponse{}, err
	}
	items := make([]common.ShareToken, 0, len(rows))
	for _, r := range rows {
		items = append(items, s.toView(r, nameByID))
	}
	return common.TokenListResponse{Items: items}, nil
}

func (s *Service) Create(name string, sourceIDs []uint, groupMode common.TokenGroupMode, groupNames []string) (common.ShareToken, error) {
	ids, err := s.normalizeSourceIDs(sourceIDs)
	if err != nil {
		return common.ShareToken{}, err
	}
	mode := common.NormalizeTokenGroupMode(groupMode)
	names, err := s.normalizeGroupNames(mode, groupNames)
	if err != nil {
		return common.ShareToken{}, err
	}
	token, err := crypto.RandomToken(24)
	if err != nil {
		return common.ShareToken{}, err
	}
	enc, err := s.box.Encrypt(token)
	if err != nil {
		return common.ShareToken{}, err
	}
	idsJSON, err := encodeSourceIDs(ids)
	if err != nil {
		return common.ShareToken{}, err
	}
	namesJSON, err := encodeGroupNames(names)
	if err != nil {
		return common.ShareToken{}, err
	}
	row := database.ShareToken{
		Name:           name,
		TokenHash:      crypto.HashToken(token),
		TokenPrefix:    tokenPrefix(token),
		TokenEncrypted: enc,
		SourceIDsJSON:  idsJSON,
		GroupMode:      string(mode),
		GroupNamesJSON: namesJSON,
		Status:         string(common.TokenStatusActive),
	}
	if err := s.db.Create(&row).Error; err != nil {
		return common.ShareToken{}, err
	}
	nameByID, _ := s.sourceNameMap()
	return s.toView(row, nameByID), nil
}

func (s *Service) Update(id uint, req common.UpdateTokenRequest) (common.ShareToken, error) {
	var row database.ShareToken
	if err := s.db.First(&row, id).Error; err != nil {
		return common.ShareToken{}, err
	}
	if req.Name != nil {
		row.Name = *req.Name
	}
	if req.Status != nil {
		row.Status = string(*req.Status)
	}
	if req.SourceIDs != nil {
		ids, err := s.normalizeSourceIDs(*req.SourceIDs)
		if err != nil {
			return common.ShareToken{}, err
		}
		idsJSON, err := encodeSourceIDs(ids)
		if err != nil {
			return common.ShareToken{}, err
		}
		row.SourceIDsJSON = idsJSON
	}
	mode := common.NormalizeTokenGroupMode(common.TokenGroupMode(row.GroupMode))
	if req.GroupMode != nil {
		mode = common.NormalizeTokenGroupMode(*req.GroupMode)
		row.GroupMode = string(mode)
	}
	if req.GroupNames != nil || req.GroupMode != nil {
		// 更新模式或白名单时重新规范化；custom 必须有组
		namesIn := decodeGroupNames(row.GroupNamesJSON)
		if req.GroupNames != nil {
			namesIn = *req.GroupNames
		}
		names, err := s.normalizeGroupNames(mode, namesIn)
		if err != nil {
			return common.ShareToken{}, err
		}
		namesJSON, err := encodeGroupNames(names)
		if err != nil {
			return common.ShareToken{}, err
		}
		row.GroupNamesJSON = namesJSON
		row.GroupMode = string(mode)
	}
	if err := s.db.Save(&row).Error; err != nil {
		return common.ShareToken{}, err
	}
	nameByID, _ := s.sourceNameMap()
	return s.toView(row, nameByID), nil
}

// Revoke 作废令牌：保留行与访问统计，Status=revoked，旧订阅链接立即失效。
// 与 Delete 不同：可再 Regenerate 换新密钥并重新激活。
func (s *Service) Revoke(id uint) (common.ShareToken, error) {
	status := common.TokenStatusRevoked
	return s.Update(id, common.UpdateTokenRequest{Status: &status})
}

// Regenerate 轮换密钥：写入新 TokenHash/密文，Status 置回 active。
// 旧链接永久失效；行与名称/源范围/策略组配置保留。
func (s *Service) Regenerate(id uint) (common.ShareToken, error) {
	var row database.ShareToken
	if err := s.db.First(&row, id).Error; err != nil {
		return common.ShareToken{}, err
	}
	token, err := crypto.RandomToken(24)
	if err != nil {
		return common.ShareToken{}, err
	}
	enc, err := s.box.Encrypt(token)
	if err != nil {
		return common.ShareToken{}, err
	}
	row.TokenHash = crypto.HashToken(token)
	row.TokenPrefix = tokenPrefix(token)
	row.TokenEncrypted = enc
	row.Status = string(common.TokenStatusActive)
	if err := s.db.Save(&row).Error; err != nil {
		return common.ShareToken{}, err
	}
	nameByID, _ := s.sourceNameMap()
	return s.toView(row, nameByID), nil
}

// Delete 硬删除令牌行，释放 TokenHash 唯一索引。
// 不可恢复；需要留痕时用 Revoke，审计记在 audit 表而非软删。
func (s *Service) Delete(id uint) error {
	res := s.db.Unscoped().Delete(&database.ShareToken{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ResolveConfig 校验 token 并返回配置 YAML（源过滤 + 策略组投影）。
func (s *Service) ResolveConfig(rawToken string) (string, error) {
	row, err := s.findActiveToken(rawToken)
	if err != nil {
		return "", err
	}
	sourceIDs := decodeSourceIDs(row.SourceIDsJSON)
	mode := string(common.NormalizeTokenGroupMode(common.TokenGroupMode(row.GroupMode)))
	groupNames := decodeGroupNames(row.GroupNamesJSON)
	yamlBody, err := s.publish.CurrentYAMLForToken(sourceIDs, mode, groupNames)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("no published config")
		}
		return "", err
	}
	now := time.Now()
	if err := s.db.Model(&row).Updates(map[string]interface{}{
		"access_count":   gorm.Expr("access_count + 1"),
		"last_access_at": now,
	}).Error; err != nil {
		applog.Warn("subscription access counter update failed token_id=%d: %v", row.ID, err)
	}
	return yamlBody, nil
}

// SubscriptionUserInfoHeader 合并已启用源的上游流量，生成 Subscription-Userinfo
// 策略：upload/download/total 求和；expire 取最早非零到期（与 Clash Verge 单源头格式一致）
func (s *Service) SubscriptionUserInfoHeader() string {
	return s.SubscriptionUserInfoHeaderForSources(nil)
}

// SubscriptionUserInfoHeaderForSources 按源过滤后的流量头；sourceIDs 空=全部启用源
func (s *Service) SubscriptionUserInfoHeaderForSources(sourceIDs []uint) string {
	q := s.db.Where("enabled = ?", true)
	if len(sourceIDs) > 0 {
		q = q.Where("id IN ?", sourceIDs)
	}
	var rows []database.Source
	if err := q.Find(&rows).Error; err != nil {
		return source.FormatSubscriptionUserInfoHeader(source.SubscriptionUserInfo{})
	}
	items := make([]source.SubscriptionUserInfo, 0, len(rows))
	for _, r := range rows {
		info := source.SubscriptionUserInfo{
			Upload:   r.TrafficUpload,
			Download: r.TrafficDownload,
			Total:    r.TrafficTotal,
			Expire:   r.TrafficExpire,
		}
		if info.HasAny() {
			items = append(items, info)
		}
	}
	return source.FormatSubscriptionUserInfoHeader(source.MergeSubscriptionUserInfo(items))
}

// ResolveSubscription 校验 token，返回 YAML 与对应该令牌的流量头
func (s *Service) ResolveSubscription(rawToken string) (yamlBody string, userInfo string, err error) {
	row, err := s.findActiveToken(rawToken)
	if err != nil {
		return "", "", err
	}
	sourceIDs := decodeSourceIDs(row.SourceIDsJSON)
	mode := string(common.NormalizeTokenGroupMode(common.TokenGroupMode(row.GroupMode)))
	groupNames := decodeGroupNames(row.GroupNamesJSON)
	yamlBody, err = s.publish.CurrentYAMLForToken(sourceIDs, mode, groupNames)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", "", fmt.Errorf("no published config")
		}
		return "", "", err
	}
	now := time.Now()
	if err := s.db.Model(&row).Updates(map[string]interface{}{
		"access_count":   gorm.Expr("access_count + 1"),
		"last_access_at": now,
	}).Error; err != nil {
		applog.Warn("subscription access counter update failed token_id=%d: %v", row.ID, err)
	}
	return yamlBody, s.SubscriptionUserInfoHeaderForSources(sourceIDs), nil
}

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

func (s *Service) toView(r database.ShareToken, nameByID map[uint]string) common.ShareToken {
	plain := ""
	if r.TokenEncrypted != "" && s.box != nil {
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
		CreatedAt:   r.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   r.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if r.LastAccessAt != nil {
		ts := r.LastAccessAt.UTC().Format(time.RFC3339)
		v.LastAccessAt = &ts
	}
	if plain != "" {
		v.SubscribeURL = s.baseURL + "/subscribe/" + plain
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
