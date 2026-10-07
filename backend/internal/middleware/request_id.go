package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

const requestIDKey = "request_id"

// RequestID 为每个请求生成 request_id。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			buf := make([]byte, 16)
			if _, err := rand.Read(buf); err == nil {
				rid = hex.EncodeToString(buf)
			}
		}
		c.Set(requestIDKey, rid)
		c.Header("X-Request-ID", rid)
		c.Next()
	}
}
