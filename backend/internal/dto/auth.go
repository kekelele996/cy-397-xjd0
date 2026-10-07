package dto

// RegisterRequest 注册请求。
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64,alphanum"`
	Password string `json:"password" binding:"required,min=6,max=72"`
}

// LoginRequest 登录请求。
type LoginRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=6,max=72"`
}

// AuthResponse 认证成功返回。
type AuthResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	User      any    `json:"user"`
}
