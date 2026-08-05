package rule

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/apiresp"
	"github.com/submerge/submerge/backend/internal/audit"
	"github.com/submerge/submerge/backend/internal/geo"
	"github.com/submerge/submerge/backend/internal/middleware"
)

// Handler 规则 HTTP
type Handler struct {
	svc   *Service
	audit *audit.Service
	geo   *geo.Service
}

func NewHandler(svc *Service, auditSvc *audit.Service, geoSvc *geo.Service) *Handler {
	return &Handler{svc: svc, audit: auditSvc, geo: geoSvc}
}

func (h *Handler) bindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
		return false
	}
	return true
}

func (h *Handler) parseID(c *gin.Context) (uint, bool) {
	id, err := apiresp.ParseID(c)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid id")
		return 0, false
	}
	return id, true
}

func (h *Handler) requireBatchIDs(c *gin.Context, ids []uint) bool {
	if err := validateBatchIDs(ids, maxBatchIDs); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
		return false
	}
	if len(ids) == 0 {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "ids required")
		return false
	}
	return true
}

func (h *Handler) logAction(c *gin.Context, action, resource, detail string) {
	h.audit.Log(middleware.GetUsername(c), action, resource, detail, c.ClientIP())
}

// MatchRules 按调用方传入的规则快照模拟匹配（含 GEOSITE/GEOIP）。
func (h *Handler) MatchRules(c *gin.Context) {
	var req struct {
		Input   string          `json:"input"`
		Rules   []geo.MatchRule `json:"rules"`
		Resolve bool            `json:"resolve"`
	}
	if !h.bindJSON(c, &req) {
		return
	}
	if h.geo == nil {
		apiresp.Fail(c, http.StatusServiceUnavailable, "geo_unavailable", "geo service not configured")
		return
	}
	result, err := h.geo.MatchRules(req.Input, req.Rules, req.Resolve)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	apiresp.OK(c, result)
}

func (h *Handler) ListRules(c *gin.Context) {
	res, err := h.svc.ListRules()
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "list rules failed")
		return
	}
	apiresp.OK(c, res)
}

func (h *Handler) CreateRule(c *gin.Context) {
	var req common.UpsertRuleRequest
	if !h.bindJSON(c, &req) {
		return
	}
	item, err := h.svc.CreateRule(req)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	h.logAction(c, "create_rule", "rule", item.Target)
	apiresp.OK(c, item)
}

func (h *Handler) BatchImportRules(c *gin.Context) {
	var req common.BatchImportRulesRequest
	if !h.bindJSON(c, &req) {
		return
	}
	if strings.TrimSpace(req.Text) == "" {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "text required")
		return
	}
	if err := validateBatchImportText(req.Text); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	res, err := h.svc.BatchImportRules(req)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	h.logAction(c, "batch_import_rules", "rule", strconv.Itoa(res.Created)+" created")
	apiresp.OK(c, res)
}

func (h *Handler) UpdateRule(c *gin.Context) {
	id, ok := h.parseID(c)
	if !ok {
		return
	}
	var req common.UpsertRuleRequest
	if !h.bindJSON(c, &req) {
		return
	}
	item, err := h.svc.UpdateRule(id, req)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	h.logAction(c, "update_rule", "rule", strconv.FormatUint(uint64(id), 10))
	apiresp.OK(c, item)
}

func (h *Handler) BatchUpdateRulesTarget(c *gin.Context) {
	var req common.BatchUpdateRulesTargetRequest
	if !h.bindJSON(c, &req) || !h.requireBatchIDs(c, req.IDs) {
		return
	}
	if strings.TrimSpace(req.Target) == "" {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "target required")
		return
	}
	n, err := h.svc.BatchUpdateRulesTarget(req.IDs, req.Target)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	h.logAction(c, "batch_update_rules_target", "rule",
		strconv.Itoa(n)+" → "+strings.TrimSpace(req.Target))
	apiresp.OK(c, common.BatchUpdateRulesTargetResponse{Updated: n})
}

func (h *Handler) BatchUpdateRulesEnabled(c *gin.Context) {
	var req common.BatchUpdateRulesEnabledRequest
	if !h.bindJSON(c, &req) || !h.requireBatchIDs(c, req.IDs) {
		return
	}
	n, err := h.svc.BatchUpdateRulesEnabled(req.IDs, req.Enabled)
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "batch update rules enabled failed")
		return
	}
	state := "disabled"
	if req.Enabled {
		state = "enabled"
	}
	h.logAction(c, "batch_update_rules_enabled", "rule", strconv.Itoa(n)+" "+state)
	apiresp.OK(c, common.BatchUpdateRulesEnabledResponse{Updated: n})
}

func (h *Handler) BatchUpdateRulesCategory(c *gin.Context) {
	var req common.BatchUpdateRulesCategoryRequest
	if !h.bindJSON(c, &req) || !h.requireBatchIDs(c, req.IDs) {
		return
	}
	n, err := h.svc.BatchUpdateRulesCategory(req.IDs, req.Category)
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "batch update rules category failed")
		return
	}
	cat := strings.TrimSpace(req.Category)
	if cat == "" {
		cat = "(未分类)"
	}
	h.logAction(c, "batch_update_rules_category", "rule", strconv.Itoa(n)+" → "+cat)
	apiresp.OK(c, common.BatchUpdateRulesCategoryResponse{Updated: n})
}

func (h *Handler) BatchDeleteRules(c *gin.Context) {
	var req common.BatchDeleteRulesRequest
	if !h.bindJSON(c, &req) || !h.requireBatchIDs(c, req.IDs) {
		return
	}
	n, err := h.svc.BatchDeleteRules(req.IDs)
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "batch delete rules failed")
		return
	}
	h.logAction(c, "batch_delete_rules", "rule", strconv.Itoa(n)+" deleted")
	apiresp.OK(c, common.BatchDeleteRulesResponse{Deleted: n})
}

func (h *Handler) DeleteRule(c *gin.Context) {
	id, ok := h.parseID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteRule(id); err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "delete rule failed")
		return
	}
	h.logAction(c, "delete_rule", "rule", strconv.FormatUint(uint64(id), 10))
	apiresp.OK(c, map[string]bool{"success": true})
}

func (h *Handler) ReorderRules(c *gin.Context) {
	var req common.ReorderRulesRequest
	if !h.bindJSON(c, &req) {
		return
	}
	if err := h.svc.ReorderRules(req.OrderedIDs); err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "reorder failed")
		return
	}
	h.logAction(c, "reorder_rules", "rule", "")
	apiresp.OK(c, map[string]bool{"success": true})
}

func (h *Handler) ListGroups(c *gin.Context) {
	res, err := h.svc.ListGroups()
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "list groups failed")
		return
	}
	apiresp.OK(c, res)
}

func (h *Handler) CreateGroup(c *gin.Context) {
	var req common.UpsertProxyGroupRequest
	if !h.bindJSON(c, &req) {
		return
	}
	item, err := h.svc.CreateGroup(req)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	h.logAction(c, "create_group", "proxy_group", item.Name)
	apiresp.OK(c, item)
}

func (h *Handler) UpdateGroup(c *gin.Context) {
	id, ok := h.parseID(c)
	if !ok {
		return
	}
	var req common.UpsertProxyGroupRequest
	if !h.bindJSON(c, &req) {
		return
	}
	item, err := h.svc.UpdateGroup(id, req)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	h.logAction(c, "update_group", "proxy_group", item.Name)
	apiresp.OK(c, item)
}

func (h *Handler) DeleteGroup(c *gin.Context) {
	id, ok := h.parseID(c)
	if !ok {
		return
	}
	cascade := c.Query("cascadeRules") == "1" || c.Query("cascadeRules") == "true"
	if err := h.svc.DeleteGroup(id, cascade); err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "delete group failed")
		return
	}
	h.logAction(c, "delete_group", "proxy_group", strconv.FormatUint(uint64(id), 10))
	apiresp.OK(c, map[string]bool{"success": true})
}
