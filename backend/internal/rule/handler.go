package rule

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/apiresp"
	"github.com/submerge/submerge/backend/internal/audit"
	"github.com/submerge/submerge/backend/internal/middleware"
)

// Handler 规则 HTTP
type Handler struct {
	svc   *Service
	audit *audit.Service
}

func NewHandler(svc *Service, auditSvc *audit.Service) *Handler {
	return &Handler{svc: svc, audit: auditSvc}
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
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	item, err := h.svc.CreateRule(req)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	h.audit.Log(middleware.GetUsername(c), "create_rule", "rule", item.Target, c.ClientIP())
	apiresp.OK(c, item)
}

func (h *Handler) BatchImportRules(c *gin.Context) {
	var req common.BatchImportRulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
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
	h.audit.Log(middleware.GetUsername(c), "batch_import_rules", "rule",
		strconv.Itoa(res.Created)+" created", c.ClientIP())
	apiresp.OK(c, res)
}

func (h *Handler) UpdateRule(c *gin.Context) {
	id, err := apiresp.ParseID(c)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}
	var req common.UpsertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	item, err := h.svc.UpdateRule(id, req)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	h.audit.Log(middleware.GetUsername(c), "update_rule", "rule", strconv.FormatUint(uint64(id), 10), c.ClientIP())
	apiresp.OK(c, item)
}

func (h *Handler) BatchUpdateRulesTarget(c *gin.Context) {
	var req common.BatchUpdateRulesTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	if err := validateBatchIDs(req.IDs, maxBatchIDs); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if len(req.IDs) == 0 {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "ids required")
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
	h.audit.Log(middleware.GetUsername(c), "batch_update_rules_target", "rule",
		strconv.Itoa(n)+" → "+strings.TrimSpace(req.Target), c.ClientIP())
	apiresp.OK(c, common.BatchUpdateRulesTargetResponse{Updated: n})
}

func (h *Handler) BatchUpdateRulesEnabled(c *gin.Context) {
	var req common.BatchUpdateRulesEnabledRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	if err := validateBatchIDs(req.IDs, maxBatchIDs); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if len(req.IDs) == 0 {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "ids required")
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
	h.audit.Log(middleware.GetUsername(c), "batch_update_rules_enabled", "rule",
		strconv.Itoa(n)+" "+state, c.ClientIP())
	apiresp.OK(c, common.BatchUpdateRulesEnabledResponse{Updated: n})
}

func (h *Handler) BatchUpdateRulesCategory(c *gin.Context) {
	var req common.BatchUpdateRulesCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	if err := validateBatchIDs(req.IDs, maxBatchIDs); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if len(req.IDs) == 0 {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "ids required")
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
	h.audit.Log(middleware.GetUsername(c), "batch_update_rules_category", "rule",
		strconv.Itoa(n)+" → "+cat, c.ClientIP())
	apiresp.OK(c, common.BatchUpdateRulesCategoryResponse{Updated: n})
}

func (h *Handler) BatchDeleteRules(c *gin.Context) {
	var req common.BatchDeleteRulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	if err := validateBatchIDs(req.IDs, maxBatchIDs); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if len(req.IDs) == 0 {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "ids required")
		return
	}
	n, err := h.svc.BatchDeleteRules(req.IDs)
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "batch delete rules failed")
		return
	}
	h.audit.Log(middleware.GetUsername(c), "batch_delete_rules", "rule",
		strconv.Itoa(n)+" deleted", c.ClientIP())
	apiresp.OK(c, common.BatchDeleteRulesResponse{Deleted: n})
}

func (h *Handler) DeleteRule(c *gin.Context) {
	id, err := apiresp.ParseID(c)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}
	if err := h.svc.DeleteRule(id); err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "delete rule failed")
		return
	}
	h.audit.Log(middleware.GetUsername(c), "delete_rule", "rule", strconv.FormatUint(uint64(id), 10), c.ClientIP())
	apiresp.OK(c, map[string]bool{"success": true})
}

func (h *Handler) ReorderRules(c *gin.Context) {
	var req common.ReorderRulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	if err := h.svc.ReorderRules(req.OrderedIDs); err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "reorder failed")
		return
	}
	h.audit.Log(middleware.GetUsername(c), "reorder_rules", "rule", "", c.ClientIP())
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
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	item, err := h.svc.CreateGroup(req)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	h.audit.Log(middleware.GetUsername(c), "create_group", "proxy_group", item.Name, c.ClientIP())
	apiresp.OK(c, item)
}

func (h *Handler) UpdateGroup(c *gin.Context) {
	id, err := apiresp.ParseID(c)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}
	var req common.UpsertProxyGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	item, err := h.svc.UpdateGroup(id, req)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	h.audit.Log(middleware.GetUsername(c), "update_group", "proxy_group", item.Name, c.ClientIP())
	apiresp.OK(c, item)
}

func (h *Handler) DeleteGroup(c *gin.Context) {
	id, err := apiresp.ParseID(c)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}
	cascade := c.Query("cascadeRules") == "1" || c.Query("cascadeRules") == "true"
	if err := h.svc.DeleteGroup(id, cascade); err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "delete group failed")
		return
	}
	h.audit.Log(middleware.GetUsername(c), "delete_group", "proxy_group", strconv.FormatUint(uint64(id), 10), c.ClientIP())
	apiresp.OK(c, map[string]bool{"success": true})
}
