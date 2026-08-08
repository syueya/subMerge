package outbound

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/submerge/submerge/backend/internal/apiresp"
)

type ProxyManager interface {
	View() View
	Save(UpdateRequest) (View, error)
	Reset() (View, error)
}

type Handler struct {
	manager ProxyManager
}

func NewHandler(manager ProxyManager) *Handler {
	return &Handler{manager: manager}
}

func (h *Handler) Get(c *gin.Context) {
	apiresp.OK(c, h.manager.View())
}

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
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "reset proxy setting failed")
		return
	}
	apiresp.OK(c, view)
}
