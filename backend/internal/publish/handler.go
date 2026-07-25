package publish

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/apiresp"
	"github.com/submerge/submerge/backend/internal/audit"
	"github.com/submerge/submerge/backend/internal/middleware"
)

// Handler 发布 HTTP
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
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "list releases failed")
		return
	}
	apiresp.OK(c, res)
}

func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}
	res, err := h.svc.Get(uint(id))
	if err != nil {
		apiresp.Fail(c, http.StatusNotFound, "not_found", "release not found")
		return
	}
	apiresp.OK(c, res)
}

func (h *Handler) Current(c *gin.Context) {
	res, err := h.svc.CurrentPublished()
	if err != nil {
		apiresp.Fail(c, http.StatusNotFound, "not_found", "no published config")
		return
	}
	apiresp.OK(c, res)
}

func (h *Handler) Preview(c *gin.Context) {
	res, err := h.svc.Preview()
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "preview_failed", err.Error())
		return
	}
	apiresp.OK(c, res)
}

func (h *Handler) DraftStatus(c *gin.Context) {
	res, err := h.svc.DraftStatus()
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "draft status failed")
		return
	}
	apiresp.OK(c, res)
}

func (h *Handler) Publish(c *gin.Context) {
	var req common.PublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	actor := middleware.GetUsername(c)
	res, err := h.svc.Publish(req.Note, actor)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "publish_failed", err.Error())
		return
	}
	h.audit.Log(actor, "publish", "release", "v"+strconv.Itoa(res.Release.Version), c.ClientIP())
	apiresp.OK(c, res)
}

func (h *Handler) Rollback(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}
	actor := middleware.GetUsername(c)
	res, err := h.svc.Rollback(uint(id), actor)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "rollback_failed", err.Error())
		return
	}
	h.audit.Log(actor, "rollback", "release", "v"+strconv.Itoa(res.Version), c.ClientIP())
	apiresp.OK(c, res)
}
