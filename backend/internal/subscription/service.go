package subscription

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/applog"
	"github.com/submerge/submerge/backend/internal/config"
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
	baseMu  sync.RWMutex
	baseURL string
}

func NewService(db *gorm.DB, publishSvc *publish.Service, box *crypto.Box, baseURL string) *Service {
	return &Service{db: db, publish: publishSvc, box: box, baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/")}
}

func (s *Service) SetBaseURL(baseURL string) error {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if err := config.ValidatePublicBaseURL(baseURL); err != nil {
		return err
	}
	s.baseMu.Lock()
	s.baseURL = baseURL
	s.baseMu.Unlock()
	return nil
}

func (s *Service) baseURLSnapshot() string {
	s.baseMu.RLock()
	defer s.baseMu.RUnlock()
	return s.baseURL
}

func (s *Service) List(allowPlain bool) (common.TokenListResponse, error) {
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
		items = append(items, s.toView(r, nameByID, allowPlain))
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
	return s.toView(row, nameByID, true), nil
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
	return s.toView(row, nameByID, true), nil
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
	return s.toView(row, nameByID, true), nil
}

// Delete 硬删除令牌行，释放 TokenHash 唯一索引。
// 不可恢复；需要撤销时用 Revoke。
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
