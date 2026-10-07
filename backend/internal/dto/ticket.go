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
	// ContractNo 关联的合同编号，需为合同库中存在的合同；历史兼容允许留空。
	ContractNo string `json:"contract_no" binding:"omitempty,max=32"`
}

// AddTicketReplyRequest 添加工单回复请求。
type AddTicketReplyRequest struct {
	Role        string   `json:"role" binding:"omitempty,oneof=user lawyer"`
	Content     string   `json:"content" binding:"required"`
	Attachments []string `json:"attachments"`
}

// UpdateTicketStatusRequest 工单流转请求；关单（closed）必须通过对账。
type UpdateTicketStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=pending processing replied closed review_pending"`
}

// BatchCloseTicketsRequest 工单组批量关单，逐笔先对账再关单。
type BatchCloseTicketsRequest struct {
	TicketIDs []uint64 `json:"ticket_ids" binding:"required,min=1,max=100,dive,gt=0"`
}

// ReconcileMismatch 单笔工单关单对账差异。
type ReconcileMismatch struct {
	TicketID             uint64 `json:"ticket_id"`
	ContractNo           string `json:"contract_no"`
	RegisteredStatus     string `json:"registered_status"`
	CurrentStatus        string `json:"current_status"`
	CurrentContractFound bool   `json:"current_contract_found"`
	Reason               string `json:"reason"`
}

// BatchCloseResult 批量关单结果。
type BatchCloseResult struct {
	Closed    []uint64            `json:"closed"`
	Suspended []ReconcileMismatch `json:"suspended"`
}

// ContractSnapshotView 工单详情中带出的合同库此刻状态与签署方。
type ContractSnapshotView struct {
	ContractNo string       `json:"contract_no"`
	Status     string       `json:"status"`
	Title      string       `json:"title"`
	Found      bool         `json:"found"`
	Signers    []SignerView `json:"signers"`
}

// SignerView 签署方视图。
type SignerView struct {
	Name     string `json:"name"`
	Role     string `json:"role"`
	SignInfo string `json:"sign_info"`
}

// TicketDetailView 工单详情：工单 + 回复 + 关联合同现状。
type TicketDetailView struct {
	Ticket                   any                   `json:"ticket"`
	Replies                  any                   `json:"replies"`
	RegisteredContractNo     string                `json:"registered_contract_no,omitempty"`
	RegisteredContractStatus string                `json:"registered_contract_status,omitempty"`
	Contract                 *ContractSnapshotView `json:"contract,omitempty"`
}

// BackfillResult 旧工单合同编号回填结果。
type BackfillResult struct {
	Total      int      `json:"total"`
	Backfilled int      `json:"backfilled"`
	Failed     int      `json:"failed"`
	FailedIDs  []uint64 `json:"failed_ticket_ids"`
}
