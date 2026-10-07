package dto

// TicketListQuery 工单列表查询参数。
type TicketListQuery struct {
	Pagination
	Status string `form:"status"`
}

// CreateTicketRequest 创建法律咨询工单请求。
type CreateTicketRequest struct {
	Type        string   `json:"type" binding:"required"`
	Title       string   `json:"title" binding:"required,max=255"`
	Description string   `json:"description" binding:"required"`
	Attachments []string `json:"attachments"`
	// ContractID 关联合同编号，可空；填写时必须存在且归属当前用户，
	// 工单会登记该合同此刻的状态与签署方快照。
	ContractID uint64 `json:"contract_id"`
}

// AddTicketReplyRequest 添加工单回复请求。
type AddTicketReplyRequest struct {
	Role        string   `json:"role" binding:"omitempty,oneof=user lawyer"`
	Content     string   `json:"content" binding:"required"`
	Attachments []string `json:"attachments"`
}

// UpdateTicketStatusRequest 工单流转请求。
// 关单（closed）必须走对账接口；pending_review/on_hold 只由系统联动/对账设置，不在此枚举中。
type UpdateTicketStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=pending processing replied"`
}

// ReviewTicketRequest 复核确认请求：把待复核/挂起工单登记的合同快照刷新为合同库现状。
type ReviewTicketRequest struct {
	// Note 复核备注，可空，仅用于日志留痕。
	Note string `json:"note" binding:"max=512"`
}

// BatchCloseTicketsRequest 工单组批量关单请求，逐笔对账。
type BatchCloseTicketsRequest struct {
	TicketIDs []uint64 `json:"ticket_ids" binding:"required,min=1,max=100,dive,gt=0"`
}

// BackfillTicketContractRequest 旧数据合同编号回填请求（可限定工单范围，默认扫全部未关联工单）。
type BackfillTicketContractRequest struct {
	TicketIDs []uint64 `json:"ticket_ids" binding:"omitempty,max=500,dive,gt=0"`
}
