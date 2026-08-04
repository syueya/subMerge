package apikey

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/apiresp"
	"github.com/submerge/submerge/backend/internal/audit"
	"github.com/submerge/submerge/backend/internal/middleware"
)

// Handler API 密钥 HTTP（仅 Session 可访问，由 RequireSession 守卫）
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
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "list api keys failed")
		return
	}
	apiresp.OK(c, res)
}

func (h *Handler) Create(c *gin.Context) {
	var req common.CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
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
	h.audit.Log(middleware.GetUsername(c), "create_apikey", "apikey", item.Name, c.ClientIP())
	apiresp.OK(c, item)
}

func (h *Handler) Update(c *gin.Context) {
	id, err := apiresp.ParseID(c)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}
	var req common.UpdateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
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
	h.audit.Log(middleware.GetUsername(c), "update_apikey", "apikey", item.Name, c.ClientIP())
	apiresp.OK(c, item)
}

func (h *Handler) Revoke(c *gin.Context) {
	id, err := apiresp.ParseID(c)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid id")
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
	h.audit.Log(middleware.GetUsername(c), "revoke_apikey", "apikey", item.Name, c.ClientIP())
	apiresp.OK(c, item)
}

func (h *Handler) Regenerate(c *gin.Context) {
	id, err := apiresp.ParseID(c)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid id")
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
	h.audit.Log(middleware.GetUsername(c), "regenerate_apikey", "apikey", item.Name, c.ClientIP())
	apiresp.OK(c, item)
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := apiresp.ParseID(c)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid id")
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
	h.audit.Log(middleware.GetUsername(c), "delete_apikey", "apikey", strconv.FormatUint(uint64(id), 10), c.ClientIP())
	apiresp.OK(c, map[string]bool{"success": true})
}

func (h *Handler) Secret(c *gin.Context) {
	id, err := apiresp.ParseID(c)
	if err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid id")
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
	h.audit.Log(middleware.GetUsername(c), "view_apikey_secret", "apikey", strconv.FormatUint(uint64(id), 10), c.ClientIP())
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
