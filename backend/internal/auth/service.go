package auth

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/apiresp"
	"github.com/submerge/submerge/backend/internal/crypto"
	"github.com/submerge/submerge/backend/internal/database"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// bootstrapMu 串行化首次建管理员，避免并发 bootstrap 都看到 count=0。
// 事务内 count+create 仍保留，作为第二道防线。
var bootstrapMu sync.Mutex

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrWeakPassword       = errors.New("password too weak")
	ErrSetupNotNeeded     = errors.New("admin already exists")
	ErrUsernameTaken      = errors.New("username already taken")
	ErrInvalidUsername    = errors.New("invalid username")
)

// Service 认证服务
type Service struct {
	db         *gorm.DB
	sessionTTL time.Duration
}

func NewService(db *gorm.DB, sessionTTL time.Duration) *Service {
	return &Service{db: db, sessionTTL: sessionTTL}
}

// minPasswordLen 口令最小长度
const minPasswordLen = 10

// weakPasswords 常见弱口令黑名单（小写比较），拒绝明显易猜的口令。
var weakPasswords = map[string]struct{}{
	"password":      {},
	"password1":     {},
	"password123":   {},
	"12345678":      {},
	"123456789":     {},
	"1234567890":    {},
	"qwertyuiop":    {},
	"admin123":      {},
	"administrator": {},
	"changeme123":   {},
	"letmein123":    {},
	"iloveyou123":   {},
}

// validatePassword 校验口令强度：长度下限 + 常见弱口令黑名单。
func validatePassword(password string) error {
	if len(password) < minPasswordLen {
		return ErrWeakPassword
	}
	if _, ok := weakPasswords[strings.ToLower(strings.TrimSpace(password))]; ok {
		return ErrWeakPassword
	}
	return nil
}

// PurgeExpiredSessions 硬删所有已过期会话，防止无限堆积。
func (s *Service) PurgeExpiredSessions() (int64, error) {
	res := s.db.Unscoped().
		Where("expires_at <= ?", time.Now()).
		Delete(&database.Session{})
	return res.RowsAffected, res.Error
}

// NeedsSetup 库中是否还没有管理员
func (s *Service) NeedsSetup() (bool, error) {
	var count int64
	if err := s.db.Model(&database.Admin{}).Count(&count).Error; err != nil {
		return false, err
	}
	return count == 0, nil
}

// Bootstrap 首次创建管理员并直接登录（仅空库）
func (s *Service) Bootstrap(username, password, displayName, avatar string) (token string, user common.AdminUser, err error) {
	username = strings.TrimSpace(username)
	if err := validateUsername(username); err != nil {
		return "", user, err
	}
	if err := validatePassword(password); err != nil {
		return "", user, err
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = username
	}
	if len([]rune(displayName)) > 32 {
		return "", user, fmt.Errorf("display name too long")
	}
	avatar = strings.TrimSpace(avatar)
	if avatar != "" {
		if len(avatar) > maxAvatarBytes {
			return "", user, fmt.Errorf("avatar too large")
		}
		if !isAllowedAvatarDataURL(avatar) {
			return "", user, fmt.Errorf("avatar must be a PNG/JPEG/WebP/GIF data URL")
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", user, err
	}

	// 进程内互斥 + 事务：两层防护，保证全局只有一个管理员
	bootstrapMu.Lock()
	defer bootstrapMu.Unlock()

	err = s.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&database.Admin{}).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrSetupNotNeeded
		}
		return tx.Create(&database.Admin{
			Username:     username,
			PasswordHash: string(hash),
			DisplayName:  displayName,
			Avatar:       avatar,
		}).Error
	})
	if err != nil {
		return "", user, err
	}
	return s.Login(username, password)
}

func validateUsername(username string) error {
	username = strings.TrimSpace(username)
	if username == "" || len(username) > 32 {
		return ErrInvalidUsername
	}
	for _, r := range username {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.'
		if !ok {
			return ErrInvalidUsername
		}
	}
	return nil
}

// dummyBcryptHash 用户不存在时也跑一次 bcrypt 比较，抹平「用户存在与否」的
// 响应耗时差异，避免用户名枚举时序侧信道。值为任意固定的 cost=10 有效哈希，
// 与真实密码永不匹配，仅用于消耗与真实比较相当的 CPU 时间。
const dummyBcryptHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

// Login 校验密码并创建会话
func (s *Service) Login(username, password string) (token string, user common.AdminUser, err error) {
	var admin database.Admin
	if err = s.db.Where("username = ?", username).First(&admin).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 用户不存在也执行一次 bcrypt，使两条路径耗时相当（防用户名枚举）
			_ = bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(password))
			return "", user, ErrInvalidCredentials
		}
		return "", user, err
	}
	if err = bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		return "", user, ErrInvalidCredentials
	}

	token, err = crypto.RandomToken(32)
	if err != nil {
		return "", user, err
	}
	now := time.Now()
	sess := database.Session{
		AdminID:   admin.ID,
		TokenHash: crypto.HashToken(token),
		ExpiresAt: now.Add(s.sessionTTL),
	}
	if err = s.db.Create(&sess).Error; err != nil {
		return "", user, err
	}
	admin.LastLoginAt = &now
	_ = s.db.Save(&admin).Error

	user = toAdminUser(admin)
	return token, user, nil
}

// Logout 使会话失效（硬删：token_hash 有唯一索引，软删无恢复价值）
func (s *Service) Logout(token string) error {
	if token == "" {
		return nil
	}
	return s.db.Unscoped().
		Where("token_hash = ?", crypto.HashToken(token)).
		Delete(&database.Session{}).Error
}

// ChangePassword 修改密码并清除全部会话（需重新登录）
func (s *Service) ChangePassword(adminID uint, oldPassword, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var admin database.Admin
		if err := tx.First(&admin, adminID).Error; err != nil {
			return err
		}
		if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(oldPassword)); err != nil {
			return ErrInvalidCredentials
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		admin.PasswordHash = string(hash)
		if err := tx.Save(&admin).Error; err != nil {
			return err
		}
		// 硬删全部会话，避免软删堆积且撞 token_hash 唯一索引
		return tx.Unscoped().Where("admin_id = ?", adminID).Delete(&database.Session{}).Error
	})
}

// maxAvatarBytes 头像 data URL 最大长度（约 200KB 字符）
const maxAvatarBytes = 200 * 1024

// allowedAvatarPrefixes 头像仅允许位图格式；显式排除 svg+xml，
// 避免 data:image/svg+xml 内嵌脚本导致的存储型 XSS。
var allowedAvatarPrefixes = []string{
	"data:image/png;",
	"data:image/jpeg;",
	"data:image/webp;",
	"data:image/gif;",
}

func isAllowedAvatarDataURL(av string) bool {
	lower := strings.ToLower(av)
	for _, p := range allowedAvatarPrefixes {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	return false
}

// UpdateProfile 更新登录名 / 昵称 / 头像
func (s *Service) UpdateProfile(adminID uint, req common.UpdateProfileRequest) (common.AdminUser, error) {
	var admin database.Admin
	if err := s.db.First(&admin, adminID).Error; err != nil {
		return common.AdminUser{}, err
	}
	if req.Username != nil {
		u := strings.TrimSpace(*req.Username)
		if err := validateUsername(u); err != nil {
			return common.AdminUser{}, err
		}
		if !strings.EqualFold(u, admin.Username) {
			var n int64
			if err := s.db.Model(&database.Admin{}).
				Where("username = ? AND id <> ?", u, adminID).
				Count(&n).Error; err != nil {
				return common.AdminUser{}, err
			}
			if n > 0 {
				return common.AdminUser{}, ErrUsernameTaken
			}
			admin.Username = u
		}
	}
	if req.DisplayName != nil {
		name := strings.TrimSpace(*req.DisplayName)
		if len([]rune(name)) > 32 {
			return common.AdminUser{}, fmt.Errorf("display name too long")
		}
		admin.DisplayName = name
	}
	if req.Avatar != nil {
		av := strings.TrimSpace(*req.Avatar)
		if av == "" {
			admin.Avatar = ""
		} else {
			if len(av) > maxAvatarBytes {
				return common.AdminUser{}, fmt.Errorf("avatar too large")
			}
			if !isAllowedAvatarDataURL(av) {
				return common.AdminUser{}, fmt.Errorf("avatar must be a PNG/JPEG/WebP/GIF data URL")
			}
			admin.Avatar = av
		}
	}
	if err := s.db.Save(&admin).Error; err != nil {
		return common.AdminUser{}, err
	}
	return toAdminUser(admin), nil
}

// GetAdmin 获取管理员信息
func (s *Service) GetAdmin(id uint) (common.AdminUser, error) {
	var admin database.Admin
	if err := s.db.First(&admin, id).Error; err != nil {
		return common.AdminUser{}, err
	}
	return toAdminUser(admin), nil
}

func toAdminUser(a database.Admin) common.AdminUser {
	display := strings.TrimSpace(a.DisplayName)
	if display == "" {
		display = a.Username
	}
	u := common.AdminUser{
		ID:          a.ID,
		Username:    a.Username,
		DisplayName: display,
		Avatar:      a.Avatar,
		CreatedAt:   apiresp.FormatRFC3339(&a.CreatedAt),
	}
	if a.LastLoginAt != nil {
		s := apiresp.FormatRFC3339(a.LastLoginAt)
		u.LastLoginAt = &s
	}
	return u
}
