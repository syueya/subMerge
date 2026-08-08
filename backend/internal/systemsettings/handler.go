package systemsettings

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/submerge/submerge/backend/internal/apiresp"
)

type Handler struct {
	manager *Manager
}

func NewHandler(manager *Manager) *Handler {
	return &Handler{manager: manager}
}
func (h *Handler) Get(c *gin.Context) { apiresp.OK(c, h.manager.View()) }
func (h *Handler) Save(c *gin.Context) {
	var req UpdateRequest
	if !apiresp.BindJSON(c, &req) {
		return
	}
	view, err := h.manager.Save(req)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	apiresp.OK(c, view)
}
func (h *Handler) Reset(c *gin.Context) {
	view, err := h.manager.Reset()
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "reset system settings failed")
		return
	}
	apiresp.OK(c, view)
}
