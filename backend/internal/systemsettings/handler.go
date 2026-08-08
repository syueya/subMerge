package systemsettings

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/submerge/submerge/backend/internal/apiresp"
	"github.com/submerge/submerge/backend/internal/audit"
	"github.com/submerge/submerge/backend/internal/middleware"
)

type Handler struct {
	manager *Manager
	audit   *audit.Service
}

func NewHandler(manager *Manager, auditSvc *audit.Service) *Handler {
	return &Handler{manager: manager, audit: auditSvc}
}
func (h *Handler) Get(c *gin.Context) { apiresp.OK(c, h.manager.View()) }
func (h *Handler) Save(c *gin.Context) {
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	view, err := h.manager.Save(req)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	h.audit.Log(middleware.GetUsername(c), "update_system_settings", "system_settings", "save system settings", c.ClientIP())
	apiresp.OK(c, view)
}
func (h *Handler) Reset(c *gin.Context) {
	view, err := h.manager.Reset()
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "reset system settings failed")
		return
	}
	h.audit.Log(middleware.GetUsername(c), "reset_system_settings", "system_settings", "reset system settings", c.ClientIP())
	apiresp.OK(c, view)
}
