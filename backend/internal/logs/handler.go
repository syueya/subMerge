package logs

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/submerge/submerge/backend/internal/apiresp"
)

// Handler 系统日志只读 API
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// List GET /api/logs?name=
func (h *Handler) List(c *gin.Context) {
	name := strings.TrimSpace(c.Query("name"))
	res, err := h.svc.List(name)
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "list logs failed")
		return
	}
	apiresp.OK(c, res)
}

// Details GET /api/logs/details?name=&line=
func (h *Handler) Details(c *gin.Context) {
	name := strings.TrimSpace(c.Query("name"))
	if name == "" {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "name is required")
		return
	}
	line := 100
	if v := strings.TrimSpace(c.Query("line")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			apiresp.Fail(c, http.StatusBadRequest, "bad_request", "line must be a positive integer")
			return
		}
		line = n
	}
	res, err := h.svc.Details(name, line)
	if err != nil {
		msg := err.Error()
		switch {
		case strings.Contains(msg, "not found"):
			apiresp.Fail(c, http.StatusNotFound, "not_found", msg)
		case strings.Contains(msg, "invalid"):
			apiresp.Fail(c, http.StatusBadRequest, "bad_request", msg)
		default:
			apiresp.Fail(c, http.StatusInternalServerError, "internal", "read log failed")
		}
		return
	}
	apiresp.OK(c, res)
}
