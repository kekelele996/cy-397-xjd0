package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/contractapi/contractapi/internal/dto"
	"github.com/contractapi/contractapi/internal/service"
)

// KnowledgeHandler 法律 FAQ 处理。
type KnowledgeHandler struct {
	knowledgeService *service.KnowledgeService
	logger           *slog.Logger
}

// NewKnowledgeHandler 构造 FAQ 处理器。
func NewKnowledgeHandler(knowledgeService *service.KnowledgeService, logger *slog.Logger) *KnowledgeHandler {
	return &KnowledgeHandler{knowledgeService: knowledgeService, logger: logger}
}

// Search GET /api/v1/faqs
func (h *KnowledgeHandler) Search(c *gin.Context) {
	var req dto.KnowledgeListQuery
	if !bindQuery(c, &req) {
		return
	}
	page, pageSize := req.Pagination.Normalize()
	list, total, err := h.knowledgeService.Search(req.Category, req.Keyword, page, pageSize)
	if err != nil {
		fail(c, err)
		return
	}
	dto.SuccessPage(c, list, total, page, pageSize)
}

// Get GET /api/v1/faqs/:id
func (h *KnowledgeHandler) Get(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	faq, err := h.knowledgeService.Get(id)
	if err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, faq)
}

// Create POST /api/v1/admin/faqs
func (h *KnowledgeHandler) Create(c *gin.Context) {
	var req dto.SaveKnowledgeRequest
	if !bindJSON(c, &req) {
		return
	}
	faq, err := h.knowledgeService.Create(req)
	if err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, faq)
}

// Update PUT /api/v1/admin/faqs/:id
func (h *KnowledgeHandler) Update(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req dto.SaveKnowledgeRequest
	if !bindJSON(c, &req) {
		return
	}
	faq, err := h.knowledgeService.Update(id, req)
	if err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, faq)
}

// Delete DELETE /api/v1/admin/faqs/:id
func (h *KnowledgeHandler) Delete(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := h.knowledgeService.Delete(id); err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, nil)
}
