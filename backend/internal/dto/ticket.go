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
}

// AddTicketReplyRequest 添加工单回复请求。
type AddTicketReplyRequest struct {
	Role        string   `json:"role" binding:"omitempty,oneof=user lawyer"`
	Content     string   `json:"content" binding:"required"`
	Attachments []string `json:"attachments"`
}

// UpdateTicketStatusRequest 工单流转请求。
type UpdateTicketStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=pending processing replied closed"`
}
