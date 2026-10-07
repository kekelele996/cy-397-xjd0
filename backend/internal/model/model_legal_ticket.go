package model

import "time"

// LegalTicket 法律咨询工单。
type LegalTicket struct {
	ID          uint64      `gorm:"primaryKey" json:"id"`
	UserID      uint64      `gorm:"index;not null" json:"user_id"`
	Type        string      `gorm:"size:32;index;not null" json:"type"`
	Title       string      `gorm:"size:255;not null" json:"title"`
	Description string      `gorm:"type:text;not null" json:"description"`
	Attachments StringSlice `gorm:"type:json" json:"attachments"`
	Status      string      `gorm:"size:32;index;not null;default:pending" json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

func (LegalTicket) TableName() string { return "legal_tickets" }
