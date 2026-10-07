package dto

import "time"

// SignerView 签署方视图。
type SignerView struct {
	Name     string     `json:"name"`
	Role     string     `json:"role"`
	SignedAt *time.Time `json:"signed_at,omitempty"`
	SignInfo string     `json:"sign_info,omitempty"`
}

// ContractSnapshotView 工单详情中带出的合同库现状与登记快照。
type ContractSnapshotView struct {
	ContractID uint64 `json:"contract_id"`
	// RegisteredStatus 工单登记时的合同状态。
	RegisteredStatus string `json:"registered_status"`
	// RegisteredSigners 工单登记时的签署方。
	RegisteredSigners []SignerSnapshotItem `json:"registered_signers"`
	// CurrentStatus 合同库此刻状态；合同已删除时为空。
	CurrentStatus string `json:"current_status"`
	// CurrentSigners 合同库此刻签署方。
	CurrentSigners []SignerView `json:"current_signers"`
	// ContractExists 合同库中是否还存在这份合同。
	ContractExists bool `json:"contract_exists"`
	// Reconciled 现状与登记快照是否一致（关单对账口径）。
	Reconciled bool `json:"reconciled"`
}

// SignerSnapshotItem 工单登记的签署方快照项。
type SignerSnapshotItem struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

// TicketDetailView 工单详情：工单本身 + 回复 + 合同库现状。
type TicketDetailView struct {
	Ticket   any                   `json:"ticket"`
	Replies  any                   `json:"replies"`
	Contract *ContractSnapshotView `json:"contract"`
}

// FieldMismatch 单笔对账差异。
type FieldMismatch struct {
	Field      string `json:"field"`
	Registered any    `json:"registered"`
	Current    any    `json:"current"`
}

// TicketMismatch 关单对不上的单笔工单明细。
type TicketMismatch struct {
	TicketID    uint64          `json:"ticket_id"`
	ContractID  uint64          `json:"contract_id"`
	Reason      string          `json:"reason"`
	Differences []FieldMismatch `json:"differences"`
}

// CloseTicketResult 单笔关单对账结果。
type CloseTicketResult struct {
	TicketID uint64 `json:"ticket_id"`
	Status   string `json:"status"`
	Closed   bool   `json:"closed"`
}

// BatchCloseTicketResult 批量关单结果：关成的与挂起/对不上的分开列出。
type BatchCloseTicketResult struct {
	Closed   []uint64         `json:"closed"`
	Mismatch []TicketMismatch `json:"mismatch"`
}

// ReviewTicketResult 复核确认结果。
type ReviewTicketResult struct {
	TicketID      uint64 `json:"ticket_id"`
	Status        string `json:"status"`
	ContractID    uint64 `json:"contract_id"`
	CurrentStatus string `json:"current_status"`
}

// ContractTicketSyncResult 合同签署/过期后联动工单的结果。
type ContractTicketSyncResult struct {
	ContractID       uint64   `json:"contract_id"`
	ContractStatus   string   `json:"contract_status"`
	PendingReviewIDs []uint64 `json:"pending_review_ids"`
	Affected         int      `json:"affected"`
}

// BackfillTicketResult 旧数据回填结果。
type BackfillTicketResult struct {
	// Backfilled 成功按描述中的编号回填的工单。
	Backfilled []BackfilledTicket `json:"backfilled"`
	// Unresolved 描述中识别不出有效合同编号的工单，单列人工处理。
	Unresolved []UnresolvedTicket `json:"unresolved"`
}

// BackfilledTicket 回填成功的工单。
type BackfilledTicket struct {
	TicketID   uint64 `json:"ticket_id"`
	ContractID uint64 `json:"contract_id"`
}

// UnresolvedTicket 回填不出的工单。
type UnresolvedTicket struct {
	TicketID uint64 `json:"ticket_id"`
	Reason   string `json:"reason"`
}
