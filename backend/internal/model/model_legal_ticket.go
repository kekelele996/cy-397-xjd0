package model

import "time"

// LegalTicket 法律咨询工单。
//
// 工单与合同通过 ContractID 关联：建单时把合同库当时的状态与签署方
// （ContractStatusSnapshot/ContractSignersSnapshot）登记到工单上，
// 合同后续流转不会改写快照；关单前以合同库现状与快照做对账。
// ContractID 为 0 表示历史遗留、无法关联合同的工单（回填后单列）。
type LegalTicket struct {
	ID          uint64      `gorm:"primaryKey" json:"id"`
	UserID      uint64      `gorm:"index;not null" json:"user_id"`
	Type        string      `gorm:"size:32;index;not null" json:"type"`
	Title       string      `gorm:"size:255;not null" json:"title"`
	Description string      `gorm:"type:text;not null" json:"description"`
	Attachments StringSlice `gorm:"type:json" json:"attachments"`
	Status      string      `gorm:"size:32;index;not null;default:pending" json:"status"`

	// ContractID 关联合同编号（合同库主键）。
	ContractID uint64 `gorm:"index;not null;default:0" json:"contract_id"`
	// ContractStatusSnapshot 建单（或最近一次复核确认）时登记的合同状态。
	ContractStatusSnapshot string `gorm:"size:32;not null;default:''" json:"contract_status_snapshot"`
	// ContractSignersSnapshot 建单（或最近一次复核确认）时登记的签署方，JSON：[{"name","role"}]。
	ContractSignersSnapshot SignerSnapshots `gorm:"type:json" json:"contract_signers_snapshot"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (LegalTicket) TableName() string { return "legal_tickets" }
