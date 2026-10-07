package constants

// ContractStatus 合同签署状态：草稿 -> 待签署 -> 已签署 -> 已过期。
const (
	ContractStatusDraft        = "draft"
	ContractStatusPendingSign  = "pending_signed"
	ContractStatusSigned       = "signed"
	ContractStatusExpired      = "expired"
)

// TicketType 法律工单问题类型。
const (
	TicketTypeLabor      = "labor"
	TicketTypeContract   = "contract"
	TicketTypeProperty   = "property"
	TicketTypeIP         = "ip"
	TicketTypeOther      = "other"
)

// TicketStatus 工单状态：待处理 -> 处理中 -> 已回复 -> 已关闭。
const (
	TicketStatusPending   = "pending"
	TicketStatusProcessing = "processing"
	TicketStatusReplied   = "replied"
	TicketStatusClosed    = "closed"
)

// TicketReplyRole 回复角色。
const (
	TicketReplyRoleUser   = "user"
	TicketReplyRoleLawyer = "lawyer"
)

// TemplateCategory 合同模板分类。
const (
	TemplateCategoryLease     = "lease"
	TemplateCategoryLabor     = "labor"
	TemplateCategoryLoan      = "loan"
	TemplateCategoryCooperation = "cooperation"
	TemplateCategoryNDA       = "nda"
)

// contractTransitions 定义合同状态合法流转方向。
var contractTransitions = map[string]map[string]bool{
	ContractStatusDraft: {
		ContractStatusPendingSign: true,
		ContractStatusExpired:     true,
	},
	ContractStatusPendingSign: {
		ContractStatusSigned:  true,
		ContractStatusExpired: true,
	},
	ContractStatusSigned: {
		ContractStatusExpired: true,
	},
}

// ticketTransitions 定义工单状态合法流转方向。
var ticketTransitions = map[string]map[string]bool{
	TicketStatusPending: {
		TicketStatusProcessing: true,
		TicketStatusClosed:     true,
	},
	TicketStatusProcessing: {
		TicketStatusReplied: true,
		TicketStatusClosed:  true,
	},
	TicketStatusReplied: {
		TicketStatusClosed: true,
	},
}

// CanTransitionContract 判断合同状态能否从 from 流转到 to。
func CanTransitionContract(from, to string) bool {
	return contractTransitions[from][to]
}

// CanTransitionTicket 判断工单状态能否从 from 流转到 to。
func CanTransitionTicket(from, to string) bool {
	return ticketTransitions[from][to]
}

// IsValidContractStatus 判断合同状态是否合法。
func IsValidContractStatus(status string) bool {
	switch status {
	case ContractStatusDraft, ContractStatusPendingSign, ContractStatusSigned, ContractStatusExpired:
		return true
	default:
		return false
	}
}

// IsValidTicketType 判断工单类型是否合法。
func IsValidTicketType(ticketType string) bool {
	switch ticketType {
	case TicketTypeLabor, TicketTypeContract, TicketTypeProperty, TicketTypeIP, TicketTypeOther:
		return true
	default:
		return false
	}
}
