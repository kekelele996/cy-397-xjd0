package model

import "time"

// TemplateFavorite 用户收藏的合同模板。
type TemplateFavorite struct {
	ID         uint64    `gorm:"primaryKey" json:"id"`
	UserID     uint64    `gorm:"uniqueIndex:uk_user_template;not null" json:"user_id"`
	TemplateID uint64    `gorm:"uniqueIndex:uk_user_template;not null" json:"template_id"`
	CreatedAt  time.Time `json:"created_at"`
}

func (TemplateFavorite) TableName() string { return "template_favorites" }
