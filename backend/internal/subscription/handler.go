package subscription

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/apiresp"
	"gorm.io/gorm"
)

// Handler 令牌与订阅 HTTP
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) List(c *gin.Context) {
	res, err := h.svc.List()
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "list tokens failed")
		return
	}
	apiresp.OK(c, res)
}

func (h *Handler) Create(c *gin.Context) {
	var req common.CreateTokenRequest
	if !apiresp.BindJSON(c, &req) {
		return
	}
	item, err := h.svc.Create(req.Name, req.SourceIDs, req.GroupMode, req.GroupNames)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "source ids not found") ||
			strings.Contains(msg, "group names not found") ||
			strings.Contains(msg, "custom group mode") {
			apiresp.Fail(c, http.StatusBadRequest, "bad_request", msg)
			return
		}
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "create token failed")
		return
	}
	apiresp.OK(c, item)
}

func (h *Handler) Update(c *gin.Context) {
	id, ok := apiresp.RequireID(c)
	if !ok {
		return
	}
	var req common.UpdateTokenRequest
	if !apiresp.BindJSON(c, &req) {
		return
	}
	item, err := h.svc.Update(id, req)
	if err != nil {
		if errors.Is(err, ErrInvalidTokenConfig) {
			apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "update token failed")
		return
	}
	apiresp.OK(c, item)
}

// Revoke 作废但保留记录（可再生成）
func (h *Handler) Revoke(c *gin.Context) {
	id, ok := apiresp.RequireID(c)
	if !ok {
		return
	}
	item, err := h.svc.Revoke(id)
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "revoke token failed")
		return
	}
	apiresp.OK(c, item)
}

func (h *Handler) Regenerate(c *gin.Context) {
	id, ok := apiresp.RequireID(c)
	if !ok {
		return
	}
	item, err := h.svc.Regenerate(id)
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "regenerate token failed")
		return
	}
	apiresp.OK(c, item)
}

// Delete 永久删除（硬删）；作废请用 Revoke
func (h *Handler) Delete(c *gin.Context) {
	id, ok := apiresp.RequireID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apiresp.Fail(c, http.StatusNotFound, "not_found", "token not found")
			return
		}
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "delete token failed")
		return
	}
	apiresp.Success(c)
}

// Subscribe 公开订阅接口（无鉴权，依赖 token）
// 响应头按 Clash Meta / mihomo / Clash Verge 常见约定设置，便于识别为配置订阅。
// 令牌可绑定订阅源子集：仅下发所选源节点与对应流量信息。
func (h *Handler) Subscribe(c *gin.Context) {
	token := c.Param("token")
	yamlBody, userInfo, err := h.svc.ResolveSubscription(token)
	if err != nil {
		c.Header("Cache-Control", "no-store")
		// 明确失败原因，便于 Clash / 浏览器排查（仍用 403，避免被当成可缓存的 404）
		c.String(http.StatusForbidden, subscribeErrorMessage(err))
		return
	}
	// 缓存
	c.Header("Cache-Control", "no-store, private")
	c.Header("Pragma", "no-cache")
	// 内容类型：多数客户端认 text/yaml / application/yaml
	c.Header("Content-Type", "text/yaml; charset=utf-8")
	// 配置文件名（Clash Verge / mihomo 面板显示用）
	c.Header("Content-Disposition", `attachment; filename="SubMerge.yaml"; filename*=UTF-8''SubMerge.yaml`)
	// 更新间隔（小时）
	c.Header("Profile-Update-Interval", "24")
	// 多源流量合并：upload/download/total 求和，expire 取最早到期
	// 客户端（Clash Verge 等）据此显示剩余流量与到期日；已按令牌允许的源过滤
	c.Header("Subscription-Userinfo", userInfo)
	c.String(http.StatusOK, yamlBody)
}

// subscribeErrorMessage 将内部错误映射为对外可读的短文案（不含敏感细节）
func subscribeErrorMessage(err error) string {
	if err == nil {
		return "forbidden"
	}
	msg := strings.TrimSpace(err.Error())
	switch {
	case msg == "missing token":
		return "missing token"
	case msg == "invalid token":
		return "invalid token"
	case msg == "token disabled":
		return "token disabled"
	case msg == "token revoked":
		return "token revoked"
	case msg == "no published config":
		return "no published config"
	case strings.Contains(msg, "暂无可用节点"), strings.Contains(msg, "no proxies"):
		return "暂无可用节点，请先添加并刷新订阅源"
	case strings.Contains(msg, "没有可用策略组"), strings.Contains(msg, "no usable proxy groups"):
		return "没有可用策略组"
	case strings.Contains(msg, "没有启用的分流规则"), strings.Contains(msg, "no enabled rules"),
		strings.Contains(msg, "MATCH 兜底"), strings.Contains(msg, "MATCH rule"), strings.Contains(msg, "MATCH"):
		return "规则配置无效"
	default:
		return "config unavailable"
	}
}
