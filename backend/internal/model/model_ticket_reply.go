package model

import "time"

// TicketReply 工单回复记录。
type TicketReply struct {
	ID          uint64      `gorm:"primaryKey" json:"id"`
	TicketID    uint64      `gorm:"index;not null" json:"ticket_id"`
	Role        string      `gorm:"size:32;not null" json:"role"`
	Content     string      `gorm:"type:text;not null" json:"content"`
	Attachments StringSlice `gorm:"type:json" json:"attachments"`
	CreatedAt   time.Time   `json:"created_at"`
}

func (TicketReply) TableName() string { return "ticket_replies" }
