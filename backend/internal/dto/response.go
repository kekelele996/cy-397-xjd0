package dto

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/contractapi/contractapi/internal/constants"
)

// Response 统一响应结构：{ code, message, data }。
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// PageResult 统一分页结果结构。
type PageResult struct {
	List     any   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// Success 输出成功响应。
func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{
		Code:    constants.CodeSuccess,
		Message: "ok",
		Data:    data,
	})
}

// SuccessPage 输出分页成功响应。
func SuccessPage(c *gin.Context, list any, total int64, page, pageSize int) {
	Success(c, PageResult{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}
