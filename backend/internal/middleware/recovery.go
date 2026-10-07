package middleware

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/contractapi/contractapi/internal/constants"
	"github.com/contractapi/contractapi/internal/dto"
)

// Recovery 捕获 panic，避免进程崩溃。
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				err := fmt.Errorf("panic recovered: %v", rec)
				logger.Error("panic recovered", "request_id", c.GetString(requestIDKey), "error", err)
				c.AbortWithStatusJSON(http.StatusInternalServerError, dto.Response{
					Code:    constants.CodeInternalError,
					Message: "internal server error",
					Data:    nil,
				})
			}
		}()
		c.Next()
	}
}
