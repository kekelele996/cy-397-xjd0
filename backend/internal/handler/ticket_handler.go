package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/contractapi/contractapi/internal/dto"
	"github.com/contractapi/contractapi/internal/middleware"
	"github.com/contractapi/contractapi/internal/service"
)

// TicketHandler 法律工单处理。
type TicketHandler struct {
	ticketService *service.TicketService
	logger        *slog.Logger
}

// NewTicketHandler 构造工单处理器。
func NewTicketHandler(ticketService *service.TicketService, logger *slog.Logger) *TicketHandler {
	return &TicketHandler{ticketService: ticketService, logger: logger}
}

// Create POST /api/v1/tickets
func (h *TicketHandler) Create(c *gin.Context) {
	var req dto.CreateTicketRequest
	if !bindJSON(c, &req) {
		return
	}
	ticket, err := h.ticketService.Create(middleware.CurrentUserID(c), req)
	if err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, ticket)
}

// List GET /api/v1/tickets
func (h *TicketHandler) List(c *gin.Context) {
	var req dto.TicketListQuery
	if !bindQuery(c, &req) {
		return
	}
	page, pageSize := req.Pagination.Normalize()
	list, total, err := h.ticketService.ListForUser(middleware.CurrentUserID(c), req.Status, page, pageSize)
	if err != nil {
		fail(c, err)
		return
	}
	dto.SuccessPage(c, list, total, page, pageSize)
}

// Get GET /api/v1/tickets/:id —— 工单详情带出合同库此刻状态与签署方。
func (h *TicketHandler) Get(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	detail, err := h.ticketService.GetDetailForUser(middleware.CurrentUserID(c), id)
	if err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, detail)
}

// AddReply POST /api/v1/tickets/:id/replies
func (h *TicketHandler) AddReply(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req dto.AddTicketReplyRequest
	if !bindJSON(c, &req) {
		return
	}
	reply, err := h.ticketService.AddReply(
		middleware.CurrentUserID(c),
		id,
		req.Role,
		req.Content,
		req.Attachments,
	)
	if err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, reply)
}

// UpdateStatus PATCH /api/v1/tickets/:id/status —— 关单前强制对账。
func (h *TicketHandler) UpdateStatus(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateTicketStatusRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := h.ticketService.Transition(middleware.CurrentUserID(c), id, req.Status); err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, gin.H{"ticket_id": id, "status": req.Status})
}

// ResumeReview POST /api/v1/tickets/:id/resume-review —— 复核通过，刷新快照并恢复处理。
func (h *TicketHandler) ResumeReview(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := h.ticketService.ResumeAfterReview(middleware.CurrentUserID(c), id); err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, gin.H{"ticket_id": id, "status": "processing"})
}

// BatchClose POST /api/v1/admin/tickets/batch-close —— 工单组批量关单，逐笔对账。
func (h *TicketHandler) BatchClose(c *gin.Context) {
	var req dto.BatchCloseTicketsRequest
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.ticketService.BatchClose(middleware.CurrentUserID(c), req.TicketIDs)
	if err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, result)
}

// BackfillContractNo POST /api/v1/admin/tickets/backfill-contract —— 旧数据回填合同编号。
func (h *TicketHandler) BackfillContractNo(c *gin.Context) {
	result, err := h.ticketService.BackfillContractNo()
	if err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, result)
}

// Backfails GET /api/v1/admin/tickets/backfails —— 回填不出的异常清单。
func (h *TicketHandler) Backfails(c *gin.Context) {
	var req struct {
		dto.Pagination
		Resolved *bool `form:"resolved"`
	}
	if !bindQuery(c, &req) {
		return
	}
	page, pageSize := req.Pagination.Normalize()
	list, total, err := h.ticketService.ListBackfails(req.Resolved, page, pageSize)
	if err != nil {
		fail(c, err)
		return
	}
	dto.SuccessPage(c, list, total, page, pageSize)
}

// Replies GET /api/v1/tickets/:id/replies
func (h *TicketHandler) Replies(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	replies, err := h.ticketService.ListReplies(middleware.CurrentUserID(c), id)
	if err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, replies)
}
