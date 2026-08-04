package apikey

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/applog"
	"github.com/submerge/submerge/backend/internal/crypto"
	"github.com/submerge/submerge/backend/internal/database"
	"gorm.io/gorm"
)

const (
	// keyPrefix 明文前缀，便于中间件识别与列表展示
	keyPrefix = "smk_"
	// randomBytes 随机段长度（与 session 相当）
	randomBytes = 32
	// lastUsedMinInterval 节流更新 last_used_at，避免高频 agent 打爆 DB
	lastUsedMinInterval = time.Minute
)

var (
	ErrNotFound      = errors.New("api key not found")
	ErrInvalidStatus = errors.New("invalid status")
	ErrInvalidExpire = errors.New("invalid expiresAt")
)

// Service API 密钥 CRUD 与鉴权查找
type Service struct {
	db  *gorm.DB
	box *crypto.Box
}

func NewService(db *gorm.DB, box *crypto.Box) *Service {
	return &Service{db: db, box: box}
}

func (s *Service) List() (common.APIKeyListResponse, error) {
	var rows []database.APIKey
	if err := s.db.Order("id desc").Find(&rows).Error; err != nil {
		return common.APIKeyListResponse{}, err
	}
	items := make([]common.APIKey, 0, len(rows))
	for _, r := range rows {
		items = append(items, s.toView(r, false))
	}
	return common.APIKeyListResponse{Items: items}, nil
}

func (s *Service) Create(req common.CreateAPIKeyRequest, createdBy string) (common.APIKey, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return common.APIKey{}, fmt.Errorf("name required")
	}
	scopes, err := common.NormalizeAPIKeyScopes(req.Scopes)
	if err != nil {
		return common.APIKey{}, err
	}
	expiresAt, err := parseExpiresAt(req.ExpiresAt)
	if err != nil {
		return common.APIKey{}, err
	}
	plain, hash, prefix, enc, err := s.generateKeyMaterial()
	if err != nil {
		return common.APIKey{}, err
	}
	scopesJSON, err := encodeScopes(scopes)
	if err != nil {
		return common.APIKey{}, err
	}
	row := database.APIKey{
		Name:         name,
		KeyHash:      hash,
		KeyPrefix:    prefix,
		KeyEncrypted: enc,
		ScopesJSON:   scopesJSON,
		Status:       string(common.APIKeyStatusActive),
		Note:         strings.TrimSpace(req.Note),
		ExpiresAt:    expiresAt,
		CreatedBy:    strings.TrimSpace(createdBy),
	}
	if err := s.db.Create(&row).Error; err != nil {
		return common.APIKey{}, err
	}
	v := s.toView(row, true)
	v.Key = plain
	return v, nil
}

func (s *Service) Update(id uint, req common.UpdateAPIKeyRequest) (common.APIKey, error) {
	var row database.APIKey
	if err := s.db.First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.APIKey{}, ErrNotFound
		}
		return common.APIKey{}, err
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return common.APIKey{}, fmt.Errorf("name required")
		}
		row.Name = name
	}
	if req.Note != nil {
		row.Note = strings.TrimSpace(*req.Note)
	}
	if req.Scopes != nil {
		scopes, err := common.NormalizeAPIKeyScopes(*req.Scopes)
		if err != nil {
			return common.APIKey{}, err
		}
		scopesJSON, err := encodeScopes(scopes)
		if err != nil {
			return common.APIKey{}, err
		}
		row.ScopesJSON = scopesJSON
	}
	if req.Status != nil {
		st := common.APIKeyStatus(strings.ToLower(strings.TrimSpace(string(*req.Status))))
		switch st {
		case common.APIKeyStatusActive, common.APIKeyStatusDisabled:
			// 已吊销的不能靠 Update 改回 active；请 regenerate
			if row.Status == string(common.APIKeyStatusRevoked) && st == common.APIKeyStatusActive {
				return common.APIKey{}, ErrInvalidStatus
			}
			row.Status = string(st)
		case common.APIKeyStatusRevoked:
			return common.APIKey{}, fmt.Errorf("%w: use revoke endpoint", ErrInvalidStatus)
		default:
			return common.APIKey{}, ErrInvalidStatus
		}
	}
	if req.ExpiresAt != nil {
		if strings.TrimSpace(*req.ExpiresAt) == "" {
			row.ExpiresAt = nil
		} else {
			t, err := parseExpiresAt(req.ExpiresAt)
			if err != nil {
				return common.APIKey{}, err
			}
			row.ExpiresAt = t
		}
	}
	if err := s.db.Save(&row).Error; err != nil {
		return common.APIKey{}, err
	}
	return s.toView(row, false), nil
}

// Revoke 作废密钥：保留行，Status=revoked，旧 key 立即失效；可再 Regenerate。
func (s *Service) Revoke(id uint) (common.APIKey, error) {
	var row database.APIKey
	if err := s.db.First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.APIKey{}, ErrNotFound
		}
		return common.APIKey{}, err
	}
	row.Status = string(common.APIKeyStatusRevoked)
	if err := s.db.Save(&row).Error; err != nil {
		return common.APIKey{}, err
	}
	return s.toView(row, false), nil
}

func (s *Service) Regenerate(id uint) (common.APIKey, error) {
	var row database.APIKey
	if err := s.db.First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.APIKey{}, ErrNotFound
		}
		return common.APIKey{}, err
	}
	plain, hash, prefix, enc, err := s.generateKeyMaterial()
	if err != nil {
		return common.APIKey{}, err
	}
	row.KeyHash = hash
	row.KeyPrefix = prefix
	row.KeyEncrypted = enc
	row.Status = string(common.APIKeyStatusActive)
	if err := s.db.Save(&row).Error; err != nil {
		return common.APIKey{}, err
	}
	v := s.toView(row, true)
	v.Key = plain
	return v, nil
}

func (s *Service) Delete(id uint) error {
	res := s.db.Unscoped().Delete(&database.APIKey{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Secret 解密返回完整密钥（仅 Session 管理端）
func (s *Service) Secret(id uint) (common.APIKeySecretResponse, error) {
	var row database.APIKey
	if err := s.db.First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.APIKeySecretResponse{}, ErrNotFound
		}
		return common.APIKeySecretResponse{}, err
	}
	plain, err := s.box.Decrypt(row.KeyEncrypted)
	if err != nil {
		return common.APIKeySecretResponse{}, fmt.Errorf("decrypt key: %w", err)
	}
	return common.APIKeySecretResponse{ID: row.ID, Key: plain}, nil
}

// FindActiveByRaw 鉴权：校验 hash / 状态 / 过期，并节流更新 last_used_at
func (s *Service) FindActiveByRaw(raw string) (database.APIKey, []common.APIKeyScope, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || !strings.HasPrefix(raw, keyPrefix) {
		return database.APIKey{}, nil, fmt.Errorf("invalid api key")
	}
	var row database.APIKey
	if err := s.db.Where("key_hash = ?", crypto.HashToken(raw)).First(&row).Error; err != nil {
		return database.APIKey{}, nil, fmt.Errorf("invalid api key")
	}
	switch row.Status {
	case string(common.APIKeyStatusActive):
		// ok
	case string(common.APIKeyStatusRevoked):
		return database.APIKey{}, nil, fmt.Errorf("api key revoked")
	default:
		return database.APIKey{}, nil, fmt.Errorf("api key disabled")
	}
	if row.ExpiresAt != nil && !row.ExpiresAt.After(time.Now()) {
		return database.APIKey{}, nil, fmt.Errorf("api key expired")
	}
	scopes := decodeScopes(row.ScopesJSON)
	if len(scopes) == 0 {
		return database.APIKey{}, nil, fmt.Errorf("api key has no scopes")
	}
	s.touchLastUsed(row.ID, row.LastUsedAt)
	return row, scopes, nil
}

func (s *Service) touchLastUsed(id uint, prev *time.Time) {
	now := time.Now()
	if prev != nil && now.Sub(*prev) < lastUsedMinInterval {
		return
	}
	if err := s.db.Model(&database.APIKey{}).Where("id = ?", id).Update("last_used_at", now).Error; err != nil {
		applog.Warn("api key last_used_at update failed id=%d: %v", id, err)
	}
}

func (s *Service) generateKeyMaterial() (plain, hash, prefix, enc string, err error) {
	randPart, err := crypto.RandomToken(randomBytes)
	if err != nil {
		return "", "", "", "", err
	}
	plain = keyPrefix + randPart
	hash = crypto.HashToken(plain)
	prefix = keyDisplayPrefix(plain)
	enc, err = s.box.Encrypt(plain)
	if err != nil {
		return "", "", "", "", err
	}
	return plain, hash, prefix, enc, nil
}

func (s *Service) toView(r database.APIKey, withPlainHint bool) common.APIKey {
	_ = withPlainHint
	scopes := decodeScopes(r.ScopesJSON)
	masked := r.KeyPrefix + "****"
	if plain, err := s.box.Decrypt(r.KeyEncrypted); err == nil && plain != "" {
		masked = crypto.MaskToken(plain)
	}
	v := common.APIKey{
		ID:        r.ID,
		Name:      r.Name,
		KeyMasked: masked,
		Scopes:    common.ScopeStrings(scopes),
		Status:    common.APIKeyStatus(r.Status),
		Note:      r.Note,
		CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: r.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if r.ExpiresAt != nil {
		ts := r.ExpiresAt.UTC().Format(time.RFC3339)
		v.ExpiresAt = &ts
	}
	if r.LastUsedAt != nil {
		ts := r.LastUsedAt.UTC().Format(time.RFC3339)
		v.LastUsedAt = &ts
	}
	return v
}

func keyDisplayPrefix(plain string) string {
	// 展示前 12 字符：smk_ + 部分 random
	if len(plain) >= 12 {
		return plain[:12]
	}
	return plain
}

func parseExpiresAt(raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}
	s := strings.TrimSpace(*raw)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidExpire, err)
	}
	return &t, nil
}

func encodeScopes(scopes []common.APIKeyScope) (string, error) {
	b, err := json.Marshal(common.ScopeStrings(scopes))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func decodeScopes(raw string) []common.APIKeyScope {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" || raw == "[]" {
		return nil
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	scopes, err := common.NormalizeAPIKeyScopes(items)
	if err != nil {
		return nil
	}
	return scopes
}
