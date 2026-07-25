package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	common "github.com/submerge/submerge/backend/common"
	"github.com/submerge/submerge/backend/internal/apiresp"
	"github.com/submerge/submerge/backend/internal/audit"
	"github.com/submerge/submerge/backend/internal/middleware"
)

// Handler HTTP 处理器
type Handler struct {
	svc          *Service
	audit        *audit.Service
	sessionTTL   time.Duration
	secureCookie bool
}

func NewHandler(svc *Service, auditSvc *audit.Service, sessionTTL time.Duration, secureCookie bool) *Handler {
	return &Handler{svc: svc, audit: auditSvc, sessionTTL: sessionTTL, secureCookie: secureCookie}
}

func (h *Handler) SetupStatus(c *gin.Context) {
	need, err := h.svc.NeedsSetup()
	if err != nil {
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "setup status failed")
		return
	}
	apiresp.OK(c, common.SetupStatusResponse{NeedsSetup: need})
}

func (h *Handler) Bootstrap(c *gin.Context) {
	var req common.BootstrapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid setup payload")
		return
	}
	token, user, err := h.svc.Bootstrap(req.Username, req.Password, req.DisplayName)
	if err != nil {
		switch {
		case errors.Is(err, ErrSetupNotNeeded):
			apiresp.Fail(c, http.StatusConflict, "setup_done", "admin already exists, please login")
		case errors.Is(err, ErrWeakPassword):
			apiresp.Fail(c, http.StatusBadRequest, "weak_password", "password must be at least 10 characters and not a common weak password")
		case errors.Is(err, ErrInvalidUsername):
			apiresp.Fail(c, http.StatusBadRequest, "bad_username", "username: 1-32 letters, digits, _ - .")
		default:
			apiresp.Fail(c, http.StatusInternalServerError, "internal", "setup failed")
		}
		return
	}
	h.audit.Log(req.Username, "bootstrap", "auth", "admin created", c.ClientIP())
	h.setSessionCookie(c, token)
	apiresp.OK(c, common.LoginResponse{User: user})
}

func (h *Handler) Login(c *gin.Context) {
	var req common.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid login payload")
		return
	}
	// 空库时引导创建管理员
	if need, err := h.svc.NeedsSetup(); err == nil && need {
		apiresp.Fail(c, http.StatusConflict, "needs_setup", "no admin yet, please create one first")
		return
	}
	token, user, err := h.svc.Login(req.Username, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			apiresp.Fail(c, http.StatusUnauthorized, "invalid_credentials", "invalid username or password")
			return
		}
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "login failed")
		return
	}
	h.audit.Log(req.Username, "login", "auth", "success", c.ClientIP())
	h.setSessionCookie(c, token)
	apiresp.OK(c, common.LoginResponse{User: user})
}

func (h *Handler) Logout(c *gin.Context) {
	token := bearerToken(c)
	_ = h.svc.Logout(token)
	h.clearSessionCookie(c)
	h.audit.Log(middleware.GetUsername(c), "logout", "auth", "", c.ClientIP())
	apiresp.OK(c, map[string]bool{"success": true})
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.svc.GetAdmin(middleware.GetAdminID(c))
	if err != nil {
		apiresp.Fail(c, http.StatusUnauthorized, "unauthorized", "not logged in")
		return
	}
	apiresp.OK(c, common.MeResponse{User: user})
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req common.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	err := h.svc.ChangePassword(middleware.GetAdminID(c), req.OldPassword, req.NewPassword)
	if err != nil {
		// 旧密码错误用 403，避免前端把 401 当成「会话失效」而强制登出
		if errors.Is(err, ErrInvalidCredentials) {
			apiresp.Fail(c, http.StatusForbidden, "invalid_credentials", "old password incorrect")
			return
		}
		if errors.Is(err, ErrWeakPassword) {
			apiresp.Fail(c, http.StatusBadRequest, "weak_password", "password must be at least 10 characters and not a common weak password")
			return
		}
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "change password failed")
		return
	}
	h.audit.Log(middleware.GetUsername(c), "change_password", "auth", "", c.ClientIP())
	apiresp.OK(c, map[string]bool{"success": true})
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	var req common.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}
	if req.Username == nil && req.DisplayName == nil && req.Avatar == nil {
		apiresp.Fail(c, http.StatusBadRequest, "bad_request", "nothing to update")
		return
	}
	user, err := h.svc.UpdateProfile(middleware.GetAdminID(c), req)
	if err != nil {
		if errors.Is(err, ErrUsernameTaken) {
			apiresp.Fail(c, http.StatusConflict, "username_taken", "username already taken")
			return
		}
		if errors.Is(err, ErrInvalidUsername) {
			apiresp.Fail(c, http.StatusBadRequest, "bad_username", "username: 1-32 letters, digits, _ - .")
			return
		}
		msg := err.Error()
		if strings.Contains(msg, "too long") ||
			strings.Contains(msg, "too large") ||
			strings.Contains(msg, "avatar must be") {
			apiresp.FailDetails(c, http.StatusBadRequest, "bad_request", "invalid profile", msg)
			return
		}
		apiresp.Fail(c, http.StatusInternalServerError, "internal", "update profile failed")
		return
	}
	h.audit.Log(middleware.GetUsername(c), "update_profile", "auth", user.DisplayName, c.ClientIP())
	apiresp.OK(c, common.MeResponse{User: user})
}

func bearerToken(c *gin.Context) string {
	raw := c.GetHeader("Authorization")
	if strings.HasPrefix(raw, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(raw, "Bearer "))
	}
	if cookie, err := c.Cookie("submerge_session"); err == nil {
		return cookie
	}
	return ""
}

// setSessionCookie 写入会话 Cookie：HttpOnly + SameSite=Lax；
// Secure 由 COOKIE_SECURE 控制（默认 false，HTTPS 部署设 true）。
func (h *Handler) setSessionCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("submerge_session", token, int(h.sessionTTL.Seconds()), "/", "", h.secureCookie, true)
}

func (h *Handler) clearSessionCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("submerge_session", "", -1, "/", "", h.secureCookie, true)
}
