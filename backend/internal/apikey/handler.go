package apikey

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/apiresp"
	"github.com/submerge/submerge/backend/internal/middleware"
)

// Handler API 密钥 HTTP（仅 Session 可访问，由 RequireSession 守卫）
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) List(c *gin.Context) {
	res, err := h.svc.List()
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "list api keys failed")
		return
	}
	apiresp.OK(c, res)
}

func (h *Handler) Create(c *gin.Context) {
	var req common.CreateAPIKeyRequest
	if !apiresp.BindJSON(c, &req) {
		return
	}
	item, err := h.svc.Create(req, middleware.GetUsername(c))
	if err != nil {
		if isClientErr(err) {
			apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "create api key failed")
		return
	}
	apiresp.OK(c, item)
}

func (h *Handler) Update(c *gin.Context) {
	id, ok := apiresp.RequireID(c)
	if !ok {
		return
	}
	var req common.UpdateAPIKeyRequest
	if !apiresp.BindJSON(c, &req) {
		return
	}
	item, err := h.svc.Update(id, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apiresp.Fail(c, http.StatusNotFound, "not_found", "api key not found")
			return
		}
		if isClientErr(err) {
			apiresp.Fail(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "update api key failed")
		return
	}
	apiresp.OK(c, item)
}

func (h *Handler) Revoke(c *gin.Context) {
	id, ok := apiresp.RequireID(c)
	if !ok {
		return
	}
	item, err := h.svc.Revoke(id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apiresp.Fail(c, http.StatusNotFound, "not_found", "api key not found")
			return
		}
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "revoke api key failed")
		return
	}
	apiresp.OK(c, item)
}

func (h *Handler) Regenerate(c *gin.Context) {
	id, ok := apiresp.RequireID(c)
	if !ok {
		return
	}
	item, err := h.svc.Regenerate(id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apiresp.Fail(c, http.StatusNotFound, "not_found", "api key not found")
			return
		}
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "regenerate api key failed")
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
		if errors.Is(err, ErrNotFound) {
			apiresp.Fail(c, http.StatusNotFound, "not_found", "api key not found")
			return
		}
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "delete api key failed")
		return
	}
	apiresp.Success(c)
}

func (h *Handler) Secret(c *gin.Context) {
	id, ok := apiresp.RequireID(c)
	if !ok {
		return
	}
	res, err := h.svc.Secret(id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apiresp.Fail(c, http.StatusNotFound, "not_found", "api key not found")
			return
		}
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "get api key secret failed")
		return
	}
	apiresp.OK(c, res)
}

func isClientErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrInvalidStatus) || errors.Is(err, ErrInvalidExpire) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "scopes") ||
		strings.Contains(msg, "name required") ||
		strings.Contains(msg, "invalid scope") ||
		strings.Contains(msg, "expiresAt")
}
