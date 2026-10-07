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
	ticketService   *service.TicketService
	ticketContract  *service.TicketContractService
	backfillService *service.BackfillService
	logger          *slog.Logger
}

// NewTicketHandler 构造工单处理器。
func NewTicketHandler(
	ticketService *service.TicketService,
	ticketContract *service.TicketContractService,
	backfillService *service.BackfillService,
	logger *slog.Logger,
) *TicketHandler {
	return &TicketHandler{
		ticketService:   ticketService,
		ticketContract:  ticketContract,
		backfillService: backfillService,
		logger:          logger,
	}
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

// Get GET /api/v1/tickets/:id
// 工单详情除工单与回复外，还带出合同库此刻的状态与签署方。
func (h *TicketHandler) Get(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	ticket, replies, err := h.ticketService.GetForUser(middleware.CurrentUserID(c), id)
	if err != nil {
		fail(c, err)
		return
	}
	contractView, err := h.ticketContract.BuildSnapshotView(ticket)
	if err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, dto.TicketDetailView{
		Ticket:   ticket,
		Replies:  replies,
		Contract: contractView,
	})
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

// UpdateStatus PATCH /api/v1/tickets/:id/status （普通流转，不含关单）
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

// Close POST /api/v1/tickets/:id/close
// 关单前拿合同库现状与工单登记快照对账，对得上才准关；对不上挂起并列出差异。
func (h *TicketHandler) Close(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	result, err := h.ticketContract.CloseWithReconcile(middleware.CurrentUserID(c), id)
	if err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, result)
}

// BatchClose POST /api/v1/tickets/batch-close
// 工单组批量关单：逐笔对账，closed 与 mismatch 分开返回。
func (h *TicketHandler) BatchClose(c *gin.Context) {
	var req dto.BatchCloseTicketsRequest
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.ticketContract.BatchClose(middleware.CurrentUserID(c), req.TicketIDs)
	if err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, result)
}

// Review POST /api/v1/tickets/:id/review
// 复核确认：把待复核/挂起工单的合同快照刷新为合同库现状并重新投入处理。
func (h *TicketHandler) Review(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req dto.ReviewTicketRequest
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.ticketContract.Review(middleware.CurrentUserID(c), id, req.Note)
	if err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, result)
}

// SyncContractTickets POST /api/v1/contracts/:id/sync-tickets
// 合同签署/过期后工单联动失败时，只重试这份合同下的这批工单。
func (h *TicketHandler) SyncContractTickets(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	result, err := h.ticketContract.SyncContractTickets(id)
	if err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, result)
}

// Backfill POST /api/v1/admin/tickets/backfill-contract
// 旧数据回填：按问题描述里的合同编号回填，回填不出的单列。
func (h *TicketHandler) Backfill(c *gin.Context) {
	var req dto.BackfillTicketContractRequest
	// 允许空 body：默认扫描全部未关联工单。
	if c.Request.ContentLength > 0 {
		if !bindJSON(c, &req) {
			return
		}
	}
	result, err := h.backfillService.Run(req.TicketIDs)
	if err != nil {
		fail(c, err)
		return
	}
	dto.Success(c, result)
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
