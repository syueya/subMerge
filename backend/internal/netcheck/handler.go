package netcheck

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/submerge/submerge/backend/internal/apiresp"
	"github.com/submerge/submerge/backend/internal/audit"
	"github.com/submerge/submerge/backend/internal/middleware"
)

// Handler 网络检测 HTTP 接口。
type Handler struct {
	svc   *Service
	audit *audit.Service
}

func NewHandler(svc *Service, auditSvc *audit.Service) *Handler {
	return &Handler{svc: svc, audit: auditSvc}
}

func (h *Handler) GetConfig(c *gin.Context) {
	cfg, err := h.svc.GetConfig()
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	apiresp.OK(c, cfg)
}

func (h *Handler) SaveConfig(c *gin.Context) {
	var req Config
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	cfg, err := h.svc.SaveConfig(req)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	h.audit.Log(middleware.GetUsername(c), "update_net_check", "net_check", "save config", c.ClientIP())
	apiresp.OK(c, cfg)
}

func (h *Handler) ResetConfig(c *gin.Context) {
	cfg, err := h.svc.ResetConfig()
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	h.audit.Log(middleware.GetUsername(c), "reset_net_check", "net_check", "reset defaults", c.ClientIP())
	apiresp.OK(c, cfg)
}

func (h *Handler) Check(c *gin.Context) {
	var req CheckRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	result, err := h.svc.Check(req)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	apiresp.OK(c, result)
}
