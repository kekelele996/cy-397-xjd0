package model

import "time"

// TicketContractBackfail 记录旧工单合同编号回填失败的清单，等待人工处理。
type TicketContractBackfail struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	TicketID    uint64    `gorm:"uniqueIndex;not null" json:"ticket_id"`
	Reason      string    `gorm:"size:255;not null" json:"reason"`
	ExtractedNo string    `gorm:"column:extracted_no;size:32" json:"extracted_no,omitempty"`
	Resolved    bool      `gorm:"index;not null;default:false" json:"resolved"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (TicketContractBackfail) TableName() string { return "ticket_contract_backfails" }
