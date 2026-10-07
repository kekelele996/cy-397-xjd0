package dto

import "github.com/contractapi/contractapi/internal/model"

// TemplateListQuery 模板列表查询参数。
type TemplateListQuery struct {
	Pagination
	Category string `form:"category"`
	Keyword  string `form:"keyword"`
}

// SaveTemplateRequest 创建/更新模板请求。
type SaveTemplateRequest struct {
	Code        string                   `json:"code" binding:"required,max=64"`
	Name        string                   `json:"name" binding:"required,max=128"`
	Category    string                   `json:"category" binding:"required,max=64"`
	Description string                   `json:"description" binding:"max=512"`
	Content     string                   `json:"content" binding:"required"`
	ContentHTML string                   `json:"content_html"`
	Variables   model.TemplateVariables `json:"variables"`
}

// FavoriteRequest 收藏/取消收藏请求。
type FavoriteRequest struct {
	TemplateID uint64 `json:"template_id" binding:"required"`
}
