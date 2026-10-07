package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/contractapi/contractapi/internal/dto"
	"github.com/contractapi/contractapi/internal/service"
)

// AuthHandler 注册与登录处理。
type AuthHandler struct {
	authService *service.AuthService
	logger      *slog.Logger
}

// NewAuthHandler 构造认证处理器。
func NewAuthHandler(authService *service.AuthService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{authService: authService, logger: logger}
}

// Register POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if !bindJSON(c, &req) {
		return
	}
	user, err := h.authService.Register(req.Username, req.Password)
	if err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, gin.H{"id": user.ID, "username": user.Username, "created_at": user.CreatedAt})
}

// Login POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if !bindJSON(c, &req) {
		return
	}
	token, expiresAt, user, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, dto.AuthResponse{
		Token:     token,
		ExpiresAt: expiresAt.Unix(),
		User:      gin.H{"id": user.ID, "username": user.Username},
	})
}
