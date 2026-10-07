package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/contractapi/contractapi/internal/constants"
	"github.com/contractapi/contractapi/internal/dto"
	"github.com/contractapi/contractapi/pkg/jwtutil"
)

const claimsKey = "jwt_claims"

// JWTAuth 校验 Authorization Bearer Token 并注入用户信息。
func JWTAuth(manager *jwtutil.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.Error(dto.NewAppError(http.StatusUnauthorized, constants.CodeUnauthorized, "missing bearer token"))
			c.Abort()
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		claims, err := manager.Parse(token)
		if err != nil {
			c.Error(dto.NewAppError(http.StatusUnauthorized, constants.CodeTokenInvalid, "invalid or expired token"))
			c.Abort()
			return
		}
		c.Set(claimsKey, claims)
		c.Next()
	}
}

// CurrentUserID 从上下文提取当前登录用户 ID。
func CurrentUserID(c *gin.Context) uint64 {
	if claims, ok := c.Get(claimsKey); ok {
		if c, ok := claims.(*jwtutil.Claims); ok {
			return c.UserID
		}
	}
	return 0
}

// CurrentUsername 从上下文提取当前登录用户名。
func CurrentUsername(c *gin.Context) string {
	if claims, ok := c.Get(claimsKey); ok {
		if c, ok := claims.(*jwtutil.Claims); ok {
			return c.Username
		}
	}
	return ""
}
