package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/contractapi/contractapi/internal/dto"
	"github.com/contractapi/contractapi/internal/middleware"
	"github.com/contractapi/contractapi/internal/service"
)

// TemplateHandler 合同模板与收藏处理。
type TemplateHandler struct {
	templateService *service.TemplateService
	logger          *slog.Logger
}

// NewTemplateHandler 构造模板处理器。
func NewTemplateHandler(templateService *service.TemplateService, logger *slog.Logger) *TemplateHandler {
	return &TemplateHandler{templateService: templateService, logger: logger}
}

// List GET /api/v1/templates
func (h *TemplateHandler) List(c *gin.Context) {
	var req dto.TemplateListQuery
	if !bindQuery(c, &req) {
		return
	}
	page, pageSize := req.Pagination.Normalize()
	list, total, err := h.templateService.List(req.Category, req.Keyword, page, pageSize)
	if err != nil {
		fail(c, err)
		return
	}
	dto.SuccessPage(c, list, total, page, pageSize)
}

// Get GET /api/v1/templates/:id
func (h *TemplateHandler) Get(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	template, err := h.templateService.Get(id)
	if err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, template)
}

// Favorites GET /api/v1/templates/favorites
func (h *TemplateHandler) Favorites(c *gin.Context) {
	var req dto.Pagination
	if !bindQuery(c, &req) {
		return
	}
	page, pageSize := req.Normalize()
	list, total, err := h.templateService.Favorites(middleware.CurrentUserID(c), page, pageSize)
	if err != nil {
		fail(c, err)
		return
	}
	dto.SuccessPage(c, list, total, page, pageSize)
}

// Favorite POST /api/v1/templates/:id/favorite
func (h *TemplateHandler) Favorite(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := h.templateService.Favorite(middleware.CurrentUserID(c), id); err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, gin.H{"template_id": id, "favorited": true})
}

// Unfavorite DELETE /api/v1/templates/:id/favorite
func (h *TemplateHandler) Unfavorite(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := h.templateService.Unfavorite(middleware.CurrentUserID(c), id); err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, gin.H{"template_id": id, "favorited": false})
}

// Create POST /api/v1/admin/templates
func (h *TemplateHandler) Create(c *gin.Context) {
	var req dto.SaveTemplateRequest
	if !bindJSON(c, &req) {
		return
	}
	template, err := h.templateService.Create(req)
	if err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, template)
}

// Update PUT /api/v1/admin/templates/:id
func (h *TemplateHandler) Update(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req dto.SaveTemplateRequest
	if !bindJSON(c, &req) {
		return
	}
	template, err := h.templateService.Update(id, req)
	if err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, template)
}

// Delete DELETE /api/v1/admin/templates/:id
func (h *TemplateHandler) Delete(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := h.templateService.Delete(id); err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, nil)
}
