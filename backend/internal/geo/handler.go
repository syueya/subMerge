package geo

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/submerge/submerge/backend/internal/apiresp"
	"github.com/submerge/submerge/backend/internal/audit"
	"github.com/submerge/submerge/backend/internal/middleware"
)

type Handler struct {
	svc   *Service
	audit *audit.Service
}

func NewHandler(svc *Service, auditSvc *audit.Service) *Handler {
	return &Handler{svc: svc, audit: auditSvc}
}

func (h *Handler) Status(c *gin.Context) {
	apiresp.OK(c, h.svc.Status())
}

func (h *Handler) Categories(c *gin.Context) {
	apiresp.OK(c, h.svc.Categories())
}

func (h *Handler) Query(c *gin.Context) {
	var req struct {
		Domain  string `json:"domain"`
		Resolve bool   `json:"resolve"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	result, err := h.svc.Query(req.Domain, req.Resolve)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	apiresp.OK(c, result)
}

func (h *Handler) IPGeo(c *gin.Context) {
	var req struct {
		IP string `json:"ip"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	result, err := h.svc.LookupIPGeo(ctx, req.IP)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	h.audit.Log(middleware.GetUsername(c), "lookup_ip_geo", "geo", result.IP, c.ClientIP())
	apiresp.OK(c, result)
}

func (h *Handler) Reverse(c *gin.Context) {
	var req struct {
		File     string `json:"file"`
		Category string `json:"category"`
		Limit    int    `json:"limit"`
		Offset   int    `json:"offset"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	result, err := h.svc.Reverse(req.File, req.Category, req.Limit, req.Offset)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	apiresp.OK(c, result)
}

func (h *Handler) Search(c *gin.Context) {
	var req struct {
		File    string `json:"file"`
		Field   string `json:"field"`
		Keyword string `json:"keyword"`
		Limit   int    `json:"limit"`
		Offset  int    `json:"offset"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	result, err := h.svc.Search(req.File, req.Field, req.Keyword, req.Limit, req.Offset)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	apiresp.OK(c, result)
}

func (h *Handler) Update(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Minute)
	defer cancel()
	result := h.svc.Update(ctx)
	failed := 0
	for _, item := range result.Items {
		if !item.Updated {
			failed++
		}
	}
	detail := "all resources updated"
	if failed > 0 {
		detail = "resource update completed with failures"
	}
	h.audit.Log(middleware.GetUsername(c), "update_geo", "geo", detail, c.ClientIP())
	apiresp.OK(c, result)
}

func normalizeFileName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
