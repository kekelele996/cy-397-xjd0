package model

import "time"

// Contract 用户生成的合同实例。
type Contract struct {
	ID            uint64    `gorm:"primaryKey" json:"id"`
	UserID        uint64    `gorm:"index;not null" json:"user_id"`
	TemplateID    uint64    `gorm:"index;not null" json:"template_id"`
	Title         string    `gorm:"size:255;not null" json:"title"`
	ContentText   string    `gorm:"type:mediumtext" json:"content_text"`
	ContentHTML   string    `gorm:"type:mediumtext" json:"content_html"`
	Status        string    `gorm:"size:32;index;not null;default:draft" json:"status"`
	Variables     JSONMap   `gorm:"type:json" json:"variables"`
	SignedAt      *time.Time `json:"signed_at,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (Contract) TableName() string { return "contracts" }
