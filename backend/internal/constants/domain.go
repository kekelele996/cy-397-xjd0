package constants

import "regexp"

// ContractStatus 合同签署状态：草稿 -> 待签署 -> 已签署 -> 已过期。
const (
	ContractStatusDraft       = "draft"
	ContractStatusPendingSign = "pending_signed"
	ContractStatusSigned      = "signed"
	ContractStatusExpired     = "expired"
)

// ContractNoPattern 合同业务编号格式：HT-2026-000001。
const ContractNoPattern = `HT-\d{4}-\d{6}`

// contractNoRegexp 用于从工单问题描述中提取合同编号。
var contractNoRegexp = regexp.MustCompile(ContractNoPattern)

// TicketType 法律工单问题类型。
const (
	TicketTypeLabor    = "labor"
	TicketTypeContract = "contract"
	TicketTypeProperty = "property"
	TicketTypeIP       = "ip"
	TicketTypeOther    = "other"
)

// TicketStatus 工单状态：待处理 -> 处理中 -> 已回复 -> 已关闭；review_pending 为待复核（挂起）。
const (
	TicketStatusPending       = "pending"
	TicketStatusProcessing    = "processing"
	TicketStatusReplied       = "replied"
	TicketStatusClosed        = "closed"
	TicketStatusReviewPending = "review_pending"
)

// TicketReplyRole 回复角色。
const (
	TicketReplyRoleUser   = "user"
	TicketReplyRoleLawyer = "lawyer"
)

// TemplateCategory 合同模板分类。
const (
	TemplateCategoryLease       = "lease"
	TemplateCategoryLabor       = "labor"
	TemplateCategoryLoan        = "loan"
	TemplateCategoryCooperation = "cooperation"
	TemplateCategoryNDA         = "nda"
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
		TicketStatusProcessing:    true,
		TicketStatusClosed:        true,
		TicketStatusReviewPending: true,
	},
	TicketStatusProcessing: {
		TicketStatusReplied:       true,
		TicketStatusClosed:        true,
		TicketStatusReviewPending: true,
	},
	TicketStatusReplied: {
		TicketStatusClosed:        true,
		TicketStatusReviewPending: true,
	},
	// 待复核（挂起）：人工核对后可恢复处理或关单。
	TicketStatusReviewPending: {
		TicketStatusProcessing: true,
		TicketStatusClosed:     true,
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

// IsValidTicketStatus 判断工单状态是否合法。
func IsValidTicketStatus(status string) bool {
	switch status {
	case TicketStatusPending, TicketStatusProcessing, TicketStatusReplied,
		TicketStatusClosed, TicketStatusReviewPending:
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

// IsValidContractNo 判断字符串是否为合法合同编号。
func IsValidContractNo(no string) bool {
	if no == "" || len(no) > 32 {
		return false
	}
	return contractNoRegexp.MatchString(no)
}

// ExtractContractNo 从文本中提取第一个合同编号，提取不到返回空串。
func ExtractContractNo(text string) string {
	return contractNoRegexp.FindString(text)
}
