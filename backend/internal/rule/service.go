package rule

import (
	"encoding/json"
	"fmt"
	"strings"

	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/database"
	"gorm.io/gorm"
)

// Service 规则与策略组草稿
type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// SeedDefaults 初始化默认策略组与规则（仅空库）。
// 内容来自 backend/defaults/groups.yaml + rules.yaml（go:embed 打进二进制）。
func (s *Service) SeedDefaults() error {
	groups, rules, err := loadSeedDefaults()
	if err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var gCount, rCount int64
		if err := tx.Model(&database.ProxyGroup{}).Count(&gCount).Error; err != nil {
			return err
		}
		if err := tx.Model(&database.Rule{}).Count(&rCount).Error; err != nil {
			return err
		}
		if gCount == 0 && rCount == 0 {
			for i := range groups {
				if err := tx.Create(&groups[i]).Error; err != nil {
					return err
				}
			}
			for i := range rules {
				if err := tx.Create(&rules[i]).Error; err != nil {
					return err
				}
			}
			return nil
		}
		// 已有库轻量迁移：直连/拒绝 + 国家组改名 + 规则挂组名 + 广告规则
		if err := ensureNamedGroup(tx, "直连", "select", []string{"DIRECT"}, 0); err != nil {
			return err
		}
		if err := ensureNamedGroup(tx, "拒绝", "select", []string{"REJECT"}, 1); err != nil {
			return err
		}
		// 规则 target：引擎关键字 → 策略组名
		if err := tx.Model(&database.Rule{}).Where("target = ?", "DIRECT").
			Update("target", "直连").Error; err != nil {
			return err
		}
		if err := tx.Model(&database.Rule{}).Where("target = ?", "REJECT").
			Update("target", "拒绝").Error; err != nil {
			return err
		}
		// 旧短码国家组 → 「中文+码」（美国US…）；规则 target 同步
		if err := migrateCountryGroupNames(tx); err != nil {
			return err
		}
		// 旧业务策略组名 → 美国US
		for _, old := range []string{"AI", "流媒体", "电报", "PROXY", "OpenAI"} {
			if err := tx.Model(&database.Rule{}).Where("target = ?", old).
				Update("target", "美国US").Error; err != nil {
				return err
			}
		}
			// 旧 MATCH 指向短码/PROXY → 日本JP（大陆默认走近端代理）
			for _, bad := range []string{"US", "PROXY"} {
				if err := tx.Model(&database.Rule{}).
					Where("type = ? AND target = ?", "MATCH", bad).
					Updates(map[string]interface{}{
						"target": "日本JP",
						"note":   "默认走代理",
					}).Error; err != nil {
					return err
				}
			}
			// 国内 GEOIP 排到域名之后
			if err := tx.Model(&database.Rule{}).
				Where("type = ? AND UPPER(payload) = ? AND note = ? AND sort_order < ?",
					"GEOIP", "CN", "国内直连", 900).
				Update("sort_order", 900).Error; err != nil {
				return err
			}
// 补「其他国家」组（非常用地区节点）
						if err := ensureNamedGroup(tx, "其他国家", "url-test", []string{"REGION:OTHER"}, 50); err != nil {
							return err
						}
					// 历史库：url-test/fallback 可能缺 url/interval（前端会显示 ?s）
					if err := repairURLTestGroups(tx); err != nil {
						return err
					}
			// 补默认广告拦截规则
			if err := ensureDefaultAdRules(tx); err != nil {
				return err
			}
			// 大陆用户默认分流：Docker/GitHub→JP、电报/社交→HK、MATCH→JP 等
			return applyMainlandUserRuleDefaults(tx)
		})
	}

	// applyMainlandUserRuleDefaults 把内置默认规则调成更适合大陆线路的出口，并补 Docker 等域名。
	// 仅按 payload 精确匹配更新，不覆盖用户自建域名规则。
	func applyMainlandUserRuleDefaults(tx *gorm.DB) error {
		type retarget struct {
			payload string
			target  string
			note    string
		}
		// 已知默认域名 → 新出口
		updates := []retarget{
			// 流媒体：YouTube/Spotify 近端
			{"youtube.com", "日本JP", "流媒体"},
			{"ytimg.com", "日本JP", "流媒体"},
			{"googlevideo.com", "日本JP", "流媒体"},
			{"youtu.be", "日本JP", "流媒体"},
			{"spotify.com", "日本JP", "流媒体"},
			{"scdn.co", "日本JP", "流媒体"},
			// 电报
			{"telegram.org", "香港HK", "电报"},
			{"t.me", "香港HK", "电报"},
			{"tdesktop.com", "香港HK", "电报"},
			{"telegra.ph", "香港HK", "电报"},
			// 开发
			{"github.com", "日本JP", "GitHub"},
			{"githubusercontent.com", "日本JP", "GitHub"},
			{"githubassets.com", "日本JP", "GitHub"},
			// Google
			{"google.com", "日本JP", "Google"},
			{"gstatic.com", "日本JP", "Google"},
			// 社交
			{"twitter.com", "香港HK", "社交"},
			{"x.com", "香港HK", "社交"},
			{"twimg.com", "香港HK", "社交"},
			{"facebook.com", "香港HK", "社交"},
			{"fbcdn.net", "香港HK", "社交"},
			{"instagram.com", "香港HK", "社交"},
			{"cdninstagram.com", "香港HK", "社交"},
			{"discord.com", "香港HK", "社交"},
			{"discordapp.com", "香港HK", "社交"},
			{"discord.gg", "香港HK", "社交"},
			{"reddit.com", "香港HK", "社交"},
			{"redd.it", "香港HK", "社交"},
			{"wikipedia.org", "日本JP", ""},
			{"medium.com", "日本JP", ""},
			{"cloudflare.com", "日本JP", ""},
		}
		for _, u := range updates {
			q := tx.Model(&database.Rule{}).
				Where("type = ? AND payload = ?", "DOMAIN-SUFFIX", u.payload)
			fields := map[string]interface{}{"target": u.target}
			if u.note != "" {
				fields["note"] = u.note
			}
			if err := q.Updates(fields).Error; err != nil {
				return err
			}
		}

		// 补开发相关域名（不存在才插入，插在 GEOIP 前）
		type addRule struct {
			payload string
			target  string
			note    string
		}
		toAdd := []addRule{
			{"github.io", "日本JP", "GitHub"},
			{"ghcr.io", "日本JP", "GitHub"},
			{"docker.com", "日本JP", "Docker"},
			{"docker.io", "日本JP", "Docker"},
			{"dockerhub.com", "日本JP", "Docker"},
			{"cloudflare.docker.com", "日本JP", "Docker"},
			{"googleusercontent.com", "日本JP", "Google"},
			{"ggpht.com", "日本JP", "Google"},
			// 海外 AI
			{"x.ai", "美国US", "AI"},
			{"grok.com", "美国US", "AI"},
			{"copilot.microsoft.com", "美国US", "AI"},
			// 国内 AI → 直连
			{"zhipuai.cn", "直连", "AI国内"},
			{"bigmodel.cn", "直连", "AI国内"},
			{"chatglm.cn", "直连", "AI国内"},
			{"deepseek.com", "直连", "AI国内"},
			{"moonshot.cn", "直连", "AI国内"},
			{"kimi.com", "直连", "AI国内"},
			{"tongyi.aliyun.com", "直连", "AI国内"},
			{"dashscope.aliyuncs.com", "直连", "AI国内"},
			{"qianfan.baidubce.com", "直连", "AI国内"},
			{"volces.com", "直连", "AI国内"},
			{"doubao.com", "直连", "AI国内"},
			{"siliconflow.cn", "直连", "AI国内"},
			{"modelscope.cn", "直连", "AI国内"},
		}
		var geoOrder int
		var geo database.Rule
		if err := tx.Where("type = ? AND UPPER(payload) = ?", "GEOIP", "CN").
			Order("sort_order asc").First(&geo).Error; err == nil {
			geoOrder = geo.SortOrder
		} else {
			geoOrder = 900
		}
		// 从 GEOIP 前一段开始占位
		nextOrder := geoOrder - len(toAdd) - 5
		if nextOrder < 50 {
			nextOrder = 50
		}
		for _, a := range toAdd {
			var n int64
			if err := tx.Model(&database.Rule{}).
				Where("type = ? AND payload = ?", "DOMAIN-SUFFIX", a.payload).
				Count(&n).Error; err != nil {
				return err
			}
			if n > 0 {
				// 已存在则只纠正出口
				if err := tx.Model(&database.Rule{}).
					Where("type = ? AND payload = ?", "DOMAIN-SUFFIX", a.payload).
					Updates(map[string]interface{}{"target": a.target, "note": a.note}).Error; err != nil {
					return err
				}
				continue
			}
			if err := tx.Create(&database.Rule{
				Type:      "DOMAIN-SUFFIX",
				Payload:   a.payload,
				Target:    a.target,
				Enabled:   true,
				SortOrder: nextOrder,
				Note:      a.note,
			}).Error; err != nil {
				return err
			}
			nextOrder++
		}

		// MATCH：默认直连 → 日本JP（仅 note 仍是默认文案时，避免覆盖用户刻意改的 MATCH）
		if err := tx.Model(&database.Rule{}).
			Where("type = ? AND target = ?", "MATCH", "直连").
			Where("note IN ?", []string{"默认走直连", "默认走代理", ""}).
			Updates(map[string]interface{}{
				"target": "日本JP",
				"note":   "默认走代理",
			}).Error; err != nil {
			return err
		}
		return nil
	}

	const defaultTestURL = "https://www.gstatic.com/generate_204"
	const defaultTestInterval = 300

	// repairURLTestGroups 为 url-test / fallback 补齐测速 URL 与间隔
	func repairURLTestGroups(tx *gorm.DB) error {
		var rows []database.ProxyGroup
		if err := tx.Where("type IN ?", []string{"url-test", "fallback"}).Find(&rows).Error; err != nil {
			return err
		}
		interval := defaultTestInterval
		for _, g := range rows {
			updates := map[string]interface{}{}
			if strings.TrimSpace(g.URL) == "" {
				updates["url"] = defaultTestURL
			}
			if g.Interval == nil || *g.Interval < 1 {
				updates["interval"] = interval
			}
			if len(updates) == 0 {
				continue
			}
			if err := tx.Model(&database.ProxyGroup{}).Where("id = ?", g.ID).Updates(updates).Error; err != nil {
				return err
			}
		}
		return nil
	}

// migrateCountryGroupNames 将裸码策略组（US）重命名为 美国US，并同步规则 target
func migrateCountryGroupNames(tx *gorm.DB) error {
	// code → 展示名（与 defaults/groups.yaml 一致）
	renames := []struct{ old, neu string }{
		{"US", "美国US"},
		{"JP", "日本JP"},
		{"HK", "香港HK"},
		{"TW", "台湾TW"},
		{"SG", "新加坡SG"},
		{"KR", "韩国KR"},
		{"GB", "英国GB"},
		{"PH", "菲律宾PH"},
		{"TR", "土耳其TR"},
	}
	for _, r := range renames {
		var oldCount, newCount int64
		if err := tx.Model(&database.ProxyGroup{}).Where("name = ?", r.old).Count(&oldCount).Error; err != nil {
			return err
		}
		if err := tx.Model(&database.ProxyGroup{}).Where("name = ?", r.neu).Count(&newCount).Error; err != nil {
			return err
		}
		if oldCount > 0 && newCount == 0 {
			if err := tx.Model(&database.ProxyGroup{}).Where("name = ?", r.old).
				Update("name", r.neu).Error; err != nil {
				return err
			}
		}
		// 规则 / 其它组里引用旧名 → 新名
		if err := tx.Model(&database.Rule{}).Where("target = ?", r.old).
			Update("target", r.neu).Error; err != nil {
			return err
		}
	}
	return nil
}

// ensureDefaultAdRules 用一条 GEOSITE 广告规则替代零散域名列表
func ensureDefaultAdRules(tx *gorm.DB) error {
	var geoCount int64
	if err := tx.Model(&database.Rule{}).
		Where("type = ? AND payload = ?", "GEOSITE", "category-ads-all").
		Count(&geoCount).Error; err != nil {
		return err
	}
	if geoCount == 0 {
		if err := tx.Create(&database.Rule{
			Type:      "GEOSITE",
			Payload:   "category-ads-all",
			Target:    "拒绝",
			Enabled:   true,
			SortOrder: 1,
			Note:      "广告",
		}).Error; err != nil {
			return err
		}
	}
	// 清理旧版默认「一长串域名广告规则」，避免列表臃肿（仅匹配我们曾写入的 note）
	return tx.Where(
		"type IN ? AND note IN ? AND payload <> ?",
		[]string{"DOMAIN-SUFFIX", "DOMAIN-KEYWORD"},
		[]string{
			"广告", "广告追踪", "统计", "广告统计", "广告关键词",
			"友盟统计", "CNZZ 统计", "百度联盟", "阿里妈妈", "阿里统计",
			"头条广告", "腾讯广告", "广点通", "小米追踪", "Meta 追踪",
		},
		"category-ads-all",
	).Delete(&database.Rule{}).Error
}

// ensureNamedGroup 若不存在指定名策略组则创建
			func ensureNamedGroup(tx *gorm.DB, name, typ string, proxies []string, sortOrder int) error {
		var count int64
		if err := tx.Model(&database.ProxyGroup{}).Where("name = ?", name).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		g := database.ProxyGroup{
			Name:      name,
			Type:      typ,
			Proxies:   mustJSON(proxies),
			Enabled:   true,
			SortOrder: sortOrder,
		}
		// url-test / fallback 默认带测速参数，避免前端显示 ?s、发布配置缺 interval
		if typ == "url-test" || typ == "fallback" {
			interval := defaultTestInterval
			g.URL = defaultTestURL
			g.Interval = &interval
		}
		return tx.Create(&g).Error
	}

func (s *Service) ListRules() (common.RuleListResponse, error) {
	var rows []database.Rule
	if err := s.db.Order("sort_order asc, id asc").Find(&rows).Error; err != nil {
		return common.RuleListResponse{}, err
	}
	items := make([]common.Rule, 0, len(rows))
	for _, r := range rows {
		items = append(items, toRule(r))
	}
	return common.RuleListResponse{Items: items}, nil
}

func (s *Service) CreateRule(req common.UpsertRuleRequest) (common.Rule, error) {
	if err := validateRule(req.Type, req.Payload, req.Target); err != nil {
		return common.Rule{}, err
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	order := 100
	if req.SortOrder != nil {
		order = *req.SortOrder
	}
	row := database.Rule{
		Type:      req.Type,
		Payload:   strings.TrimSpace(req.Payload),
		Target:    strings.TrimSpace(req.Target),
		Enabled:   enabled,
		SortOrder: order,
		Note:      req.Note,
	}
	if err := s.db.Create(&row).Error; err != nil {
		return common.Rule{}, err
	}
	return toRule(row), nil
}

// BatchImportRules 批量导入规则（一行一条），插在 GEOIP CN / MATCH 之前。
// 解析错误不阻断整批：合法行照常写入，错误汇总返回。
func (s *Service) BatchImportRules(req common.BatchImportRulesRequest) (common.BatchImportRulesResponse, error) {
	parsed, parseErrs := parseBatchImportText(req.Text, req.DefaultType, req.DefaultTarget, req.DefaultNote)
	res := common.BatchImportRulesResponse{
		Errors: parseErrs,
		Items:  []common.Rule{},
	}
	if len(parsed) == 0 {
		res.Skipped = len(parseErrs)
		return res, nil
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 已有规则：跳过 type+payload+target 完全相同的
		var existing []database.Rule
		if err := tx.Find(&existing).Error; err != nil {
			return err
		}
		seen := make(map[string]struct{}, len(existing))
		for _, r := range existing {
			key := strings.ToUpper(r.Type) + "\x00" + strings.ToLower(r.Payload) + "\x00" + r.Target
			seen[key] = struct{}{}
		}

		// 插入锚点：第一条 GEOIP CN 或 MATCH 的 sort_order；没有则接在末尾
		anchor := 0
		hasAnchor := false
		for _, r := range existing {
			if r.Type == string(common.RuleTypeMatch) ||
				(r.Type == string(common.RuleTypeGeoIP) && strings.EqualFold(r.Payload, "CN")) {
				if !hasAnchor || r.SortOrder < anchor {
					anchor = r.SortOrder
					hasAnchor = true
				}
			}
		}
		if !hasAnchor {
			maxOrder := 0
			for _, r := range existing {
				if r.SortOrder > maxOrder {
					maxOrder = r.SortOrder
				}
			}
			anchor = maxOrder + 10
		}

		// 给新规则腾位置：锚点及之后整体后移
		shift := len(parsed) * 10
		if hasAnchor && shift > 0 {
			if err := tx.Model(&database.Rule{}).
				Where("sort_order >= ?", anchor).
				Update("sort_order", gorm.Expr("sort_order + ?", shift)).Error; err != nil {
				return err
			}
		}

		created := make([]common.Rule, 0, len(parsed))
		skipped := 0
		for i, p := range parsed {
			key := strings.ToUpper(p.Type) + "\x00" + strings.ToLower(p.Payload) + "\x00" + p.Target
			if _, dup := seen[key]; dup {
				skipped++
				res.Errors = append(res.Errors, fmt.Sprintf("第%d行：已存在，已跳过", p.LineNo))
				continue
			}
			row := database.Rule{
				Type:      p.Type,
				Payload:   p.Payload,
				Target:    p.Target,
				Enabled:   enabled,
				SortOrder: anchor + i*10,
				Note:      p.Note,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			seen[key] = struct{}{}
			created = append(created, toRule(row))
		}
		res.Created = len(created)
		res.Skipped = skipped
		res.Items = created
		return nil
	})
	if err != nil {
		return common.BatchImportRulesResponse{}, err
	}
	return res, nil
}

func (s *Service) UpdateRule(id uint, req common.UpsertRuleRequest) (common.Rule, error) {
	var row database.Rule
	if err := s.db.First(&row, id).Error; err != nil {
		return common.Rule{}, err
	}
	if err := validateRule(req.Type, req.Payload, req.Target); err != nil {
		return common.Rule{}, err
	}
	row.Type = req.Type
	row.Payload = strings.TrimSpace(req.Payload)
	row.Target = strings.TrimSpace(req.Target)
	row.Note = req.Note
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if req.SortOrder != nil {
		row.SortOrder = *req.SortOrder
	}
	if err := s.db.Save(&row).Error; err != nil {
		return common.Rule{}, err
	}
	return toRule(row), nil
}

func (s *Service) DeleteRule(id uint) error {
	return s.db.Delete(&database.Rule{}, id).Error
}

func (s *Service) ReorderRules(ids []uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var rows []database.Rule
		if err := tx.Order("sort_order asc, id asc").Find(&rows).Error; err != nil {
			return err
		}
		if len(ids) != len(rows) {
			return fmt.Errorf("orderedIds must include every rule exactly once")
		}
		expected := make(map[uint]struct{}, len(rows))
		for _, row := range rows {
			expected[row.ID] = struct{}{}
		}
		seen := make(map[uint]struct{}, len(ids))
		for _, id := range ids {
			if _, ok := expected[id]; !ok {
				return fmt.Errorf("orderedIds contains unknown rule %d", id)
			}
			if _, ok := seen[id]; ok {
				return fmt.Errorf("orderedIds contains duplicate rule %d", id)
			}
			seen[id] = struct{}{}
		}
		for i, id := range ids {
			if err := tx.Model(&database.Rule{}).Where("id = ?", id).Update("sort_order", (i+1)*10).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Service) ListGroups() (common.ProxyGroupListResponse, error) {
	var rows []database.ProxyGroup
	if err := s.db.Order("sort_order asc, id asc").Find(&rows).Error; err != nil {
		return common.ProxyGroupListResponse{}, err
	}
	items := make([]common.ProxyGroup, 0, len(rows))
	for _, r := range rows {
		items = append(items, toGroup(r))
	}
	return common.ProxyGroupListResponse{Items: items}, nil
}

func (s *Service) CreateGroup(req common.UpsertProxyGroupRequest) (common.ProxyGroup, error) {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	order := 0
	if req.SortOrder != nil {
		order = *req.SortOrder
	}
	if req.Proxies == nil {
		req.Proxies = []string{}
	}
	row := database.ProxyGroup{
		Name:      strings.TrimSpace(req.Name),
		Type:      req.Type,
		Proxies:   mustJSON(req.Proxies),
		URL:       req.URL,
		Interval:  req.Interval,
		Enabled:   enabled,
		SortOrder: order,
	}
	if err := s.db.Create(&row).Error; err != nil {
		return common.ProxyGroup{}, err
	}
	return toGroup(row), nil
}

func (s *Service) UpdateGroup(id uint, req common.UpsertProxyGroupRequest) (common.ProxyGroup, error) {
	var row database.ProxyGroup
	if err := s.db.First(&row, id).Error; err != nil {
		return common.ProxyGroup{}, err
	}
	row.Name = strings.TrimSpace(req.Name)
	row.Type = req.Type
	if req.Proxies == nil {
		req.Proxies = []string{}
	}
	row.Proxies = mustJSON(req.Proxies)
	row.URL = req.URL
	row.Interval = req.Interval
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if req.SortOrder != nil {
		row.SortOrder = *req.SortOrder
	}
	if err := s.db.Save(&row).Error; err != nil {
		return common.ProxyGroup{}, err
	}
	return toGroup(row), nil
}

// DeleteGroup 删除策略组；cascadeRules 为 true 时一并删除指向该组的规则
func (s *Service) DeleteGroup(id uint, cascadeRules bool) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var row database.ProxyGroup
		if err := tx.First(&row, id).Error; err != nil {
			return err
		}
		if cascadeRules {
			if err := tx.Unscoped().Where("target = ?", row.Name).Delete(&database.Rule{}).Error; err != nil {
				return err
			}
		}
		// 硬删除：Name 带唯一索引，软删除残留行会导致同名重建冲突
		return tx.Unscoped().Delete(&database.ProxyGroup{}, id).Error
	})
}

// EnabledRules 启用规则按顺序
func (s *Service) EnabledRules() ([]database.Rule, error) {
	var rows []database.Rule
	err := s.db.Where("enabled = ?", true).Order("sort_order asc, id asc").Find(&rows).Error
	return rows, err
}

// EnabledGroups 启用策略组
func (s *Service) EnabledGroups() ([]database.ProxyGroup, error) {
	var rows []database.ProxyGroup
	err := s.db.Where("enabled = ?", true).Order("sort_order asc, id asc").Find(&rows).Error
	return rows, err
}

func validateRule(typ, payload, target string) error {
	if strings.TrimSpace(typ) == "" {
		return fmt.Errorf("rule type required")
	}
	if strings.TrimSpace(target) == "" {
		return fmt.Errorf("rule target required")
	}
	if typ != string(common.RuleTypeMatch) && strings.TrimSpace(payload) == "" {
		return fmt.Errorf("payload required for type %s", typ)
	}
	return nil
}

func toRule(r database.Rule) common.Rule {
	return common.Rule{
		ID:        r.ID,
		Type:      common.RuleType(r.Type),
		Payload:   r.Payload,
		Target:    r.Target,
		Enabled:   r.Enabled,
		SortOrder: r.SortOrder,
		Note:      r.Note,
	}
}

func toGroup(r database.ProxyGroup) common.ProxyGroup {
	var proxies []string
	_ = json.Unmarshal([]byte(r.Proxies), &proxies)
	if proxies == nil {
		proxies = []string{}
	}
	return common.ProxyGroup{
		ID:        r.ID,
		Name:      r.Name,
		Type:      common.ProxyGroupType(r.Type),
		Proxies:   proxies,
		URL:       r.URL,
		Interval:  r.Interval,
		Enabled:   r.Enabled,
		SortOrder: r.SortOrder,
	}
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
