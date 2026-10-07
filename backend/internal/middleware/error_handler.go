package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/contractapi/contractapi/internal/constants"
	"github.com/contractapi/contractapi/internal/dto"
)

// ErrorHandler 统一捕获 handler 通过 c.Error 写入的错误并转换为标准响应。
func ErrorHandler(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 {
			return
		}
		err := c.Errors[0].Err
		if appErr, ok := dto.IsAppError(err); ok {
			c.AbortWithStatusJSON(appErr.Status, dto.Response{
				Code:    appErr.Code,
				Message: appErr.Message,
				Data:    nil,
			})
			return
		}
		logger.Error("unhandled request error", "request_id", c.GetString(requestIDKey), "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, dto.Response{
			Code:    constants.CodeInternalError,
			Message: "internal server error",
			Data:    nil,
		})
	}
}
