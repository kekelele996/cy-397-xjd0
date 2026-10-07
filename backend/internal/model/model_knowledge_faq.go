package model

import "time"

// KnowledgeFAQ 常见法律知识库条目。
type KnowledgeFAQ struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	Category  string    `gorm:"size:64;index;not null" json:"category"`
	Question  string    `gorm:"size:512;not null" json:"question"`
	Answer    string    `gorm:"type:text;not null" json:"answer"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (KnowledgeFAQ) TableName() string { return "knowledge_faqs" }
