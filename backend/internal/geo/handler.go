package geo

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/submerge/submerge/backend/internal/apiresp"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
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
	if !apiresp.BindJSON(c, &req) {
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
	if !apiresp.BindJSON(c, &req) {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	result, err := h.svc.LookupIPGeo(ctx, req.IP)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	apiresp.OK(c, result)
}

func (h *Handler) Reverse(c *gin.Context) {
	var req struct {
		File     string `json:"file"`
		Category string `json:"category"`
		Limit    int    `json:"limit"`
		Offset   int    `json:"offset"`
	}
	if !apiresp.BindJSON(c, &req) {
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
	if !apiresp.BindJSON(c, &req) {
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
	apiresp.OK(c, result)
}

func normalizeFileName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
