package appupdate

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/apiresp"
	"github.com/submerge/submerge/backend/internal/updater"
)

type Handler struct {
	svc *Service
}

func NewHandler(service *Service) *Handler { return &Handler{svc: service} }

func (h *Handler) Status(c *gin.Context) { apiresp.OK(c, h.svc.Status()) }

func (h *Handler) Check(c *gin.Context) {
	ctx, cancel := contextWithTimeout(c, 30*time.Second)
	defer cancel()
	status, err := h.svc.Check(ctx)
	if err != nil {
		respondError(c, err)
		return
	}
	apiresp.OK(c, status)
}

func (h *Handler) Download(c *gin.Context) {
	status, err := h.svc.StartDownload()
	if err != nil {
		respondError(c, err)
		return
	}
	accepted(c, status)
}

func (h *Handler) Install(c *gin.Context) {
	status, request, err := h.svc.RequestInstall()
	if err != nil {
		respondError(c, err)
		return
	}
	accepted(c, status)
	c.Writer.Flush()
	h.svc.DispatchShutdown(request)
}

func (h *Handler) Rollback(c *gin.Context) {
	status, request, err := h.svc.RequestRollback()
	if err != nil {
		respondError(c, err)
		return
	}
	accepted(c, status)
	c.Writer.Flush()
	h.svc.DispatchShutdown(request)
}

func accepted(c *gin.Context, status Status) {
	c.JSON(http.StatusAccepted, common.ApiResponse[Status]{OK: true, Data: &status})
}

func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrDisabled):
		apiresp.Fail(c, http.StatusServiceUnavailable, "update_disabled", err.Error())
	case errors.Is(err, ErrBusy):
		apiresp.Fail(c, http.StatusConflict, "update_busy", err.Error())
	case errors.Is(err, ErrNotAvailable), errors.Is(err, ErrNotReady), errors.Is(err, ErrNoShutdown), errors.Is(err, updater.ErrNoRollback):
		apiresp.Fail(c, http.StatusConflict, "update_not_ready", err.Error())
	default:
		apiresp.Fail(c, http.StatusBadGateway, "update_failed", err.Error())
	}
}

func contextWithTimeout(c *gin.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Request.Context(), timeout)
}
