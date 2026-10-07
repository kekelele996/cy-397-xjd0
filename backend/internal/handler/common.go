package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/contractapi/contractapi/internal/dto"
)

// fail 将 service 层错误交给统一错误处理中间件。
func fail(c *gin.Context, err error) {
	if err == nil {
		return
	}
	if _, ok := dto.IsAppError(err); ok {
		c.Error(err)
		return
	}
	c.Error(dto.InternalError(err))
}

// bindJSON 绑定并校验请求体，校验失败直接返回。
func bindJSON(c *gin.Context, obj any) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		c.Error(dto.ValidationError(err.Error()))
		return false
	}
	return true
}

// bindQuery 绑定并校验查询参数，校验失败直接返回。
func bindQuery(c *gin.Context, obj any) bool {
	if err := c.ShouldBindQuery(obj); err != nil {
		c.Error(dto.ValidationError(err.Error()))
		return false
	}
	return true
}

// parseUintParam 解析路径中的 uint64 参数。
func parseUintParam(c *gin.Context, name string) (uint64, bool) {
	value, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || value == 0 {
		c.Error(dto.ValidationError("invalid " + name))
		return 0, false
	}
	return value, true
}

