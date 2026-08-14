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
	"gorm.io/gorm"
)

// 地区码：任意字母数字，如 US / PH / JP / HK / SG1
var regionCodeRe = regexp.MustCompile(`^[A-Za-z0-9]{1,16}$`)

// Handler 订阅源 HTTP
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
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
	if !apiresp.BindJSON(c, &req) {
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
	// 自动模式未填地区 → UNK；固定模式必须选具体地区
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
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "region must be 1-16 letters/digits (e.g. US, JP, HK, UNK)")
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
	apiresp.OK(c, item)
}

func (h *Handler) CreateManual(c *gin.Context) {
	var req common.ManualSourceRequest
	if !apiresp.BindJSON(c, &req) {
		return
	}
	if !h.normalizeManualRequest(c, &req) {
		return
	}
	res, err := h.svc.CreateManual(req)
	if err != nil {
		apiresp.FailDetails(c, http.StatusBadRequest, "manual_import_failed", "manual node import failed", err.Error())
		return
	}
	apiresp.OK(c, res)
}

func (h *Handler) UpdateManual(c *gin.Context) {
	id, ok := apiresp.RequireID(c)
	if !ok {
		return
	}
	var req common.ManualSourceRequest
	if !apiresp.BindJSON(c, &req) {
		return
	}
	if !h.normalizeManualRequest(c, &req) {
		return
	}
	res, err := h.svc.UpdateManual(id, req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apiresp.Fail(c, http.StatusNotFound, "not_found", "source not found")
			return
		}
		apiresp.FailDetails(c, http.StatusBadRequest, "manual_import_failed", "manual node import failed", err.Error())
		return
	}
	apiresp.OK(c, res)
}

func (h *Handler) normalizeManualRequest(c *gin.Context, req *common.ManualSourceRequest) bool {
	mode := string(common.RegionModeAuto)
	if req.RegionMode != nil {
		m := common.RegionMode(normalizeRegionMode(string(*req.RegionMode)))
		req.RegionMode = &m
		mode = string(m)
	}
	rawRegion := strings.TrimSpace(string(req.Region))
	if rawRegion == "" {
		if mode == string(common.RegionModeFixed) {
			apiresp.Fail(c, http.StatusBadRequest, "bad_request", "fixed mode requires a region code")
			return false
		}
		rawRegion = fallbackRegionCode()
	}
	region, ok := normalizeRegion(rawRegion)
	if !ok {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "region must be 1-16 letters/digits (e.g. US, JP, HK, UNK)")
		return false
	}
	req.Region = common.Region(region)
	return true
}

func (h *Handler) Update(c *gin.Context) {
	id, ok := apiresp.RequireID(c)
	if !ok {
		return
	}
	var req common.UpdateSourceRequest
	if !apiresp.BindJSON(c, &req) {
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
	apiresp.OK(c, item)
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := apiresp.RequireID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(id); err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "delete source failed")
		return
	}
	apiresp.Success(c)
}

func (h *Handler) BatchDelete(c *gin.Context) {
	var req common.BatchDeleteSourcesRequest
	if !apiresp.BindJSON(c, &req) {
		return
	}
	if err := validateSourceBatchIDs(req.IDs); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
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
	apiresp.OK(c, map[string]int{"deleted": n})
}

func (h *Handler) Refresh(c *gin.Context) {
	id, ok := apiresp.RequireID(c)
	if !ok {
		return
	}
	res, err := h.svc.Refresh(id)
	if err != nil {
		apiresp.FailDetails(c, http.StatusBadRequest, "refresh_failed", "refresh failed, previous snapshot retained", err.Error())
		return
	}
	apiresp.OK(c, res)
}

// RefreshAll 刷新全部启用订阅源
func (h *Handler) RefreshAll(c *gin.Context) {
	res := h.svc.RefreshAll()
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
	id, ok := apiresp.RequireID(c)
	if !ok {
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
	apiresp.OK(c, item)
}

func (h *Handler) BatchUpdateProxies(c *gin.Context) {
	var req common.BatchUpdateProxiesRequest
	if !apiresp.BindJSON(c, &req) {
		return
	}
	if err := validateSourceBatchIDs(req.IDs); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
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
	apiresp.OK(c, map[string]int{"updated": n})
}

func normalizeRegion(raw string) (string, bool) {
	r := strings.ToUpper(strings.TrimSpace(raw))
	if !regionCodeRe.MatchString(r) {
		return "", false
	}
	return r, true
}

func isClientFilterErr(err error) bool {
	return errors.Is(err, ErrInvalidFilter)
}
