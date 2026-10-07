package model

import "time"

// ContractTemplate 合同模板，内容使用 {{.变量名}} 占位。
type ContractTemplate struct {
	ID          uint64            `gorm:"primaryKey" json:"id"`
	Code        string            `gorm:"size:64;uniqueIndex;not null" json:"code"`
	Name        string            `gorm:"size:128;not null" json:"name"`
	Category    string            `gorm:"size:64;index;not null" json:"category"`
	Description string            `gorm:"size:512" json:"description"`
	Content     string            `gorm:"type:text;not null" json:"content"`
	ContentHTML string            `gorm:"type:mediumtext" json:"content_html"`
	Variables   TemplateVariables `gorm:"type:json" json:"variables"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

func (ContractTemplate) TableName() string { return "contract_templates" }
