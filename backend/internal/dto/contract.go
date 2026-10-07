package dto

// ContractListQuery 用户合同库查询参数。
type ContractListQuery struct {
	Pagination
	Status string `form:"status"`
}

// CreateContractRequest 创建并生成合同请求。
type CreateContractRequest struct {
	TemplateID uint64            `json:"template_id" binding:"required"`
	Title      string            `json:"title" binding:"required,max=255"`
	Variables  map[string]string `json:"variables"`
}

// SubmitContractRequest 提交待签署请求。
type SubmitContractRequest struct {
	Signers []SignerInput `json:"signers"`
}

// SignerInput 签署方输入。
type SignerInput struct {
	Name string `json:"name" binding:"required,max=128"`
	Role string `json:"role" binding:"required,max=64"`
}

// SignContractRequest 签署合同请求。
type SignContractRequest struct {
	SignerName string `json:"signer_name" binding:"max=128"`
	SignerRole string `json:"signer_role" binding:"max=64"`
	SignInfo   string `json:"sign_info" binding:"max=512"`
}

// ContractView 合同视图，附带模板名称。
type ContractView struct {
	ID          uint64         `json:"id"`
	UserID      uint64         `json:"user_id"`
	TemplateID  uint64         `json:"template_id"`
	TemplateName string        `json:"template_name"`
	Title       string         `json:"title"`
	ContentText string         `json:"content_text"`
	ContentHTML string         `json:"content_html"`
	Status      string         `json:"status"`
	Variables   map[string]string `json:"variables"`
	SignedAt    any            `json:"signed_at,omitempty"`
	ExpiresAt   any            `json:"expires_at,omitempty"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
}
