package source

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/apiresp"
	"github.com/submerge/submerge/backend/internal/audit"
	"github.com/submerge/submerge/backend/internal/middleware"
	"gorm.io/gorm"
)

// 地区码：任意字母数字，如 US / PH / JP / HK / SG1
var regionCodeRe = regexp.MustCompile(`^[A-Za-z0-9]{1,16}$`)

// Handler 订阅源 HTTP
type Handler struct {
	svc   *Service
	audit *audit.Service
}

func NewHandler(svc *Service, auditSvc *audit.Service) *Handler {
	return &Handler{svc: svc, audit: auditSvc}
}

func (h *Handler) List(c *gin.Context) {
	res, err := h.svc.List()
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "list sources failed")
		return
	}
	apiresp.OK(c, res)
}

// ListRegions 地区目录（固定地区下拉 / 回退选项），数据来自 defaults/regions.yaml
func (h *Handler) ListRegions(c *gin.Context) {
	items := make([]common.RegionCatalogEntry, 0)
	for _, r := range listRegionCatalog() {
		items = append(items, common.RegionCatalogEntry{Code: r.Code, Name: r.Name})
	}
	apiresp.OK(c, common.RegionCatalogResponse{
		Items:          items,
		FallbackRegion: fallbackRegionCode(),
	})
}

func (h *Handler) Create(c *gin.Context) {
	var req common.CreateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	if req.RegionMode != nil {
		mode := common.RegionMode(normalizeRegionMode(string(*req.RegionMode)))
		req.RegionMode = &mode
	}
	mode := string(common.RegionModeAuto)
	if req.RegionMode != nil {
		mode = string(*req.RegionMode)
	}
	// 自动模式未填地区 → UNKNOWN；固定模式必须选具体地区
	rawRegion := strings.TrimSpace(string(req.Region))
	if rawRegion == "" {
		if mode == string(common.RegionModeFixed) {
			apiresp.Fail(c, http.StatusBadRequest, "bad_request", "fixed mode requires a region code")
			return
		}
		rawRegion = fallbackRegionCode()
	}
	region, ok := normalizeRegion(rawRegion)
	if !ok {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "region must be 1-16 letters/digits (e.g. US, JP, HK, UNKNOWN)")
		return
	}
	req.Region = common.Region(region)
	item, err := h.svc.Create(req)
	if err != nil {
		if isClientFilterErr(err) {
			apiresp.FailDetails(c, http.StatusBadRequest, "bad_request", "invalid filter config", err.Error())
			return
		}
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "create source failed")
		return
	}
	h.audit.Log(middleware.GetUsername(c), "create_source", "source", item.Name, c.ClientIP())
	apiresp.OK(c, item)
}

func (h *Handler) Update(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}
	var req common.UpdateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	if req.Region != nil {
		region, ok := normalizeRegion(string(*req.Region))
		if !ok {
			apiresp.Fail(c, http.StatusBadRequest, "bad_request", "region must be 1-16 letters/digits (e.g. US, PH, JP, HK)")
			return
		}
		r := common.Region(region)
		req.Region = &r
	}
	if req.RegionMode != nil {
		mode := common.RegionMode(normalizeRegionMode(string(*req.RegionMode)))
		req.RegionMode = &mode
	}
	item, err := h.svc.Update(id, req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apiresp.Fail(c, http.StatusNotFound, "not_found", "source not found")
			return
		}
		if isClientFilterErr(err) {
			apiresp.FailDetails(c, http.StatusBadRequest, "bad_request", "invalid filter config", err.Error())
			return
		}
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "update source failed")
		return
	}
	h.audit.Log(middleware.GetUsername(c), "update_source", "source", item.Name, c.ClientIP())
	apiresp.OK(c, item)
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}
	if err := h.svc.Delete(id); err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "delete source failed")
		return
	}
	h.audit.Log(middleware.GetUsername(c), "delete_source", "source", strconv.FormatUint(uint64(id), 10), c.ClientIP())
	apiresp.OK(c, map[string]bool{"success": true})
}

func (h *Handler) BatchDelete(c *gin.Context) {
	var req common.BatchDeleteSourcesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	if len(req.IDs) == 0 {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "ids required")
		return
	}
	n, err := h.svc.DeleteMany(req.IDs)
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "batch delete sources failed")
		return
	}
	h.audit.Log(middleware.GetUsername(c), "batch_delete_sources", "source", strconv.Itoa(n), c.ClientIP())
	apiresp.OK(c, map[string]int{"deleted": n})
}

func (h *Handler) Refresh(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}
	res, err := h.svc.Refresh(id)
	if err != nil {
		apiresp.FailDetails(c, http.StatusBadRequest, "refresh_failed", "refresh failed, previous snapshot retained", err.Error())
		return
	}
	h.audit.Log(middleware.GetUsername(c), "refresh_source", "source", res.Source.Name, c.ClientIP())
	apiresp.OK(c, res)
}

// RefreshAll 刷新全部启用订阅源
func (h *Handler) RefreshAll(c *gin.Context) {
	res := h.svc.RefreshAll()
	h.audit.Log(middleware.GetUsername(c), "refresh_all_sources", "source",
		strconv.Itoa(res.OK)+"/"+strconv.Itoa(res.Total), c.ClientIP())
	apiresp.OK(c, res)
}

func (h *Handler) ListProxies(c *gin.Context) {
	var sourceID *uint
	if v := c.Query("sourceId"); v != "" {
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid sourceId")
			return
		}
		u := uint(n)
		sourceID = &u
	}
	res, err := h.svc.ListProxies(sourceID)
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "list proxies failed")
		return
	}
	apiresp.OK(c, res)
}

func (h *Handler) UpdateProxy(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}
	var req common.UpdateProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	item, err := h.svc.UpdateProxy(id, *req.Enabled)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apiresp.Fail(c, http.StatusNotFound, "not_found", "proxy not found")
			return
		}
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "update proxy failed")
		return
	}
	h.audit.Log(middleware.GetUsername(c), "update_proxy", "proxy", item.Name, c.ClientIP())
	apiresp.OK(c, item)
}

func (h *Handler) BatchUpdateProxies(c *gin.Context) {
	var req common.BatchUpdateProxiesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	if len(req.IDs) == 0 {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "ids required")
		return
	}
	n, err := h.svc.BatchUpdateProxies(req.IDs, req.Enabled)
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "batch update proxies failed")
		return
	}
	h.audit.Log(middleware.GetUsername(c), "batch_update_proxies", "proxy", strconv.Itoa(n), c.ClientIP())
	apiresp.OK(c, map[string]int{"updated": n})
}

func parseID(c *gin.Context) (uint, error) {
	n, err := strconv.ParseUint(c.Param("id"), 10, 64)
	return uint(n), err
}

func normalizeRegion(raw string) (string, bool) {
	r := strings.ToUpper(strings.TrimSpace(raw))
	if !regionCodeRe.MatchString(r) {
		return "", false
	}
	return r, true
}

func isClientFilterErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "excludeNameRegex") ||
		strings.Contains(msg, "includeNameRegex") ||
		strings.Contains(msg, "invalid exclude") ||
		strings.Contains(msg, "invalid include")
}
