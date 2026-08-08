package publish

import (
	"fmt"
	"strings"
	"sync"
	"time"

	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/apiresp"
	"github.com/submerge/submerge/backend/internal/database"
	"github.com/submerge/submerge/backend/internal/rule"
	"github.com/submerge/submerge/backend/internal/source"
	"gorm.io/gorm"
)

const maxReleaseHistory = 50

// Service 发布服务
type Service struct {
	db     *gorm.DB
	source *source.Service
	rule   *rule.Service
	gen    *Generator
	mu     sync.Mutex
}

func NewService(db *gorm.DB, sourceSvc *source.Service, ruleSvc *rule.Service) *Service {
	return &Service{
		db:     db,
		source: sourceSvc,
		rule:   ruleSvc,
		gen:    NewGenerator(),
	}
}

// Preview 生成预览（不落库）
func (s *Service) Preview() (common.ReleasePreview, error) {
	res, err := s.build()
	if err != nil {
		return common.ReleasePreview{}, err
	}
	return toPreview(res), nil
}

// Publish 校验并发布新版本
func (s *Service) Publish(note, actor string) (common.PublishResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.build()
	if err != nil {
		return common.PublishResponse{}, err
	}
	release, err := s.createPublishedRelease(res, note, actor)
	if err != nil {
		return common.PublishResponse{}, err
	}
	return common.PublishResponse{Release: toRelease(release), Preview: toPreview(res)}, nil
}

// List 发布历史
func (s *Service) List() (common.ReleaseListResponse, error) {
	var rows []database.Release
	if err := s.db.Order("version desc").Limit(50).Find(&rows).Error; err != nil {
		return common.ReleaseListResponse{}, err
	}
	items := make([]common.Release, 0, len(rows))
	for _, r := range rows {
		items = append(items, toRelease(r))
	}
	return common.ReleaseListResponse{Items: items}, nil
}

// Get 发布版本详情（完整 YAML + 解析出的规则/策略组名）
func (s *Service) Get(id uint) (common.ReleaseDetail, error) {
	var row database.Release
	if err := s.db.First(&row, id).Error; err != nil {
		return common.ReleaseDetail{}, err
	}
	rules, groups := parseReleaseYAML(row.ConfigYAML)
	return common.ReleaseDetail{
		Release:    toRelease(row),
		ConfigYAML: row.ConfigYAML,
		Rules:      rules,
		Groups:     groups,
	}, nil
}

// CurrentPublished 当前生效的已发布版本详情；无则 RecordNotFound
func (s *Service) CurrentPublished() (common.ReleaseDetail, error) {
	var row database.Release
	if err := s.db.Where("status = ?", string(common.ReleaseStatusPublished)).
		Order("version desc").First(&row).Error; err != nil {
		return common.ReleaseDetail{}, err
	}
	rules, groups := parseReleaseYAML(row.ConfigYAML)
	return common.ReleaseDetail{
		Release:    toRelease(row),
		ConfigYAML: row.ConfigYAML,
		Rules:      rules,
		Groups:     groups,
	}, nil
}

// Rollback 回滚到指定版本
func (s *Service) Rollback(id uint, actor string) (common.Release, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var target database.Release
	if err := s.db.First(&target, id).Error; err != nil {
		return common.Release{}, err
	}
	if target.ConfigYAML == "" {
		return common.Release{}, fmt.Errorf("target release has empty config")
	}
	release, err := s.createPublishedRelease(&BuildResult{
		YAML:       target.ConfigYAML,
		Hash:       target.ConfigHash,
		ProxyCount: target.ProxyCount,
		RuleCount:  target.RuleCount,
	}, fmt.Sprintf("rollback to v%d by %s", target.Version, actor), actor)
	if err != nil {
		return common.Release{}, err
	}
	return toRelease(release), nil
}

// Delete 删除指定版本记录（禁止删除当前生效的已发布版本）
func (s *Service) Delete(id uint, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var row database.Release
	if err := s.db.First(&row, id).Error; err != nil {
		return err
	}
	if row.Status == string(common.ReleaseStatusPublished) {
		return fmt.Errorf("cannot delete the currently published release")
	}
	return s.db.Delete(&row).Error
}

func (s *Service) createPublishedRelease(res *BuildResult, note, actor string) (database.Release, error) {
	var release database.Release
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var last database.Release
		next := 1
		if err := tx.Order("version desc").First(&last).Error; err == nil {
			next = last.Version + 1
		} else if err != gorm.ErrRecordNotFound {
			return err
		}
		if err := tx.Model(&database.Release{}).
			Where("status = ?", string(common.ReleaseStatusPublished)).
			Update("status", string(common.ReleaseStatusRolledBack)).Error; err != nil {
			return err
		}
		now := time.Now()
		release = database.Release{
			Version:     next,
			Status:      string(common.ReleaseStatusPublished),
			Note:        note,
			ProxyCount:  res.ProxyCount,
			RuleCount:   res.RuleCount,
			ConfigHash:  res.Hash,
			ConfigYAML:  res.YAML,
			PublishedAt: &now,
			CreatedBy:   actor,
		}
		if err := tx.Create(&release).Error; err != nil {
			return err
		}
		return pruneReleaseHistory(tx)
	})
	return release, err
}

func pruneReleaseHistory(tx *gorm.DB) error {
	var keep []database.Release
	if err := tx.Order("version desc").Limit(maxReleaseHistory).Find(&keep).Error; err != nil {
		return err
	}
	if len(keep) < maxReleaseHistory {
		return nil
	}
	cutoff := keep[len(keep)-1].Version
	return tx.Unscoped().Where("status = ? AND version < ?",
		string(common.ReleaseStatusRolledBack), cutoff).
		Delete(&database.Release{}).Error
}

// CurrentYAML 当前已发布配置（全部启用源）
func (s *Service) CurrentYAML() (string, error) {
	return s.CurrentYAMLForToken(nil, string(common.TokenGroupModeAuto), nil)
}

// CurrentYAMLForSources 按订阅源过滤；策略组默认 auto 投影。
func (s *Service) CurrentYAMLForSources(sourceIDs []uint) (string, error) {
	return s.CurrentYAMLForToken(sourceIDs, string(common.TokenGroupModeAuto), nil)
}

// CurrentYAMLForToken 令牌投影：源过滤 + 策略组模式。
// - 全部源且 auto：直接返回已发布快照（稳定、可回滚）
// - 指定源 / all / custom：即时 build（规则/组取当前启用草稿 + 节点按源过滤）
func (s *Service) CurrentYAMLForToken(sourceIDs []uint, groupMode string, groupNames []string) (string, error) {
	mode := string(common.NormalizeTokenGroupMode(common.TokenGroupMode(groupMode)))
	// 至少应有一个已发布版本，避免未发布时对外下发草稿
	var pub database.Release
	if err := s.db.Where("status = ?", string(common.ReleaseStatusPublished)).
		Order("version desc").First(&pub).Error; err != nil {
		return "", err
	}

	useSnapshot := len(sourceIDs) == 0 && mode == string(common.TokenGroupModeAuto)
	if useSnapshot {
		return pub.ConfigYAML, nil
	}

	res, err := s.buildForToken(sourceIDs, mode, groupNames)
	if err != nil {
		return "", err
	}
	return res.YAML, nil
}

// DraftStatus 比较当前草稿与已发布配置是否一致。
// withChanges=false 时只算 dirty/hash（列表角标、概览用），不跑 YAML diff；
// withChanges=true 时在 dirty 时填充 Changes（查看差异弹窗用）。
func (s *Service) DraftStatus(withChanges bool) (common.DraftStatusResponse, error) {
	out := common.DraftStatusResponse{}

	var pub database.Release
	err := s.db.Where("status = ?", string(common.ReleaseStatusPublished)).
		Order("version desc").First(&pub).Error
	if err == nil {
		out.HasPublished = true
		out.PublishedHash = pub.ConfigHash
		out.PublishedVersion = pub.Version
	} else if err != gorm.ErrRecordNotFound {
		return out, err
	}

	res, buildErr := s.build()
	if buildErr != nil {
		out.BuildError = buildErr.Error()
		// 无已发布配置且草稿也建不出来 → 不算 dirty；有已发布则视为有变更/异常
		out.Dirty = out.HasPublished
		return out, nil
	}
	out.DraftHash = res.Hash
	if !out.HasPublished {
		out.Dirty = true
		return out, nil
	}
	out.Dirty = out.DraftHash != out.PublishedHash
	if out.Dirty && withChanges {
		out.Changes = diffConfigs(pub.ConfigYAML, res.YAML)
	}
	return out, nil
}
func toPreview(res *BuildResult) common.ReleasePreview {
	preview := []rune(res.YAML)
	if len(preview) > 8000 {
		preview = append(preview[:8000], []rune("\n# ... truncated ...")...)
	}
	return common.ReleasePreview{
		ProxyCount:  res.ProxyCount,
		RuleCount:   res.RuleCount,
		Groups:      res.GroupNames,
		YAMLPreview: string(preview),
		Warnings:    res.Warnings,
	}
}

func (s *Service) build() (*BuildResult, error) {
	// 全量发布：GroupMode 空 = 严格规则校验 + auto 剪空组
	return s.buildForToken(nil, "", nil)
}

// buildForToken 按源与策略组模式生成配置
func (s *Service) buildForToken(sourceIDs []uint, groupMode string, groupNames []string) (*BuildResult, error) {
	proxies, err := s.source.EnabledProxiesBySourceIDs(sourceIDs)
	if err != nil {
		return nil, err
	}
	groups, err := s.rule.EnabledGroups()
	if err != nil {
		return nil, err
	}
	rules, err := s.rule.EnabledRules()
	if err != nil {
		return nil, err
	}
	return s.gen.Build(BuildInput{
		Proxies:       proxies,
		Groups:        groups,
		Rules:         rules,
		GroupMode:     groupMode,
		AllowedGroups: groupNames,
	})
}

func toRelease(r database.Release) common.Release {
	out := common.Release{
		ID:         r.ID,
		Version:    r.Version,
		Status:     common.ReleaseStatus(r.Status),
		Note:       r.Note,
		ProxyCount: r.ProxyCount,
		RuleCount:  r.RuleCount,
		ConfigHash: r.ConfigHash,
		CreatedAt:  apiresp.FormatRFC3339(&r.CreatedAt),
		CreatedBy:  r.CreatedBy,
	}
	if r.PublishedAt != nil {
		s := apiresp.FormatRFC3339(r.PublishedAt)
		out.PublishedAt = &s
	}
	return out
}

// parseReleaseYAML 从已发布 YAML 抽出 rules / proxy-groups 名（查看历史、匹配测试用）。
// 复用 parseConfigSnapshot 做原始抽取，这里只把规则行解析成结构。
func parseReleaseYAML(yamlText string) (rules []common.ReleaseRuleLine, groups []string) {
	snap := parseConfigSnapshot(yamlText)
	rules = make([]common.ReleaseRuleLine, 0, len(snap.rules))
	for _, line := range snap.rules {
		rules = append(rules, parseRuleLine(line))
	}
	groups = make([]string, 0, len(snap.order))
	groups = append(groups, snap.order...)
	return rules, groups
}

func parseRuleLine(raw string) common.ReleaseRuleLine {
	parts := strings.Split(raw, ",")
	out := common.ReleaseRuleLine{Raw: raw}
	if len(parts) == 0 {
		return out
	}
	out.Type = strings.TrimSpace(parts[0])
	if strings.EqualFold(out.Type, "MATCH") {
		if len(parts) >= 2 {
			out.Target = strings.TrimSpace(parts[1])
		}
		return out
	}
	if len(parts) >= 3 {
		out.Payload = strings.TrimSpace(parts[1])
		out.Target = strings.TrimSpace(parts[2])
	} else if len(parts) == 2 {
		out.Target = strings.TrimSpace(parts[1])
	}
	return out
}
