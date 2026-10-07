package repository

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/contractapi/contractapi/internal/model"
)

// FavoriteRepository 模板收藏数据访问接口。
type FavoriteRepository interface {
	Create(userID, templateID uint64) error
	Delete(userID, templateID uint64) error
	Exists(userID, templateID uint64) (bool, error)
	ListByUser(userID uint64, offset, limit int) ([]model.ContractTemplate, int64, error)
}

type favoriteRepository struct {
	db *gorm.DB
}

// NewFavoriteRepository 构造收藏仓储。
func NewFavoriteRepository(db *gorm.DB) FavoriteRepository {
	return &favoriteRepository{db: db}
}

func (r *favoriteRepository) Create(userID, templateID uint64) error {
	fav := model.TemplateFavorite{UserID: userID, TemplateID: templateID}
	if err := r.db.Create(&fav).Error; err != nil {
		if isDuplicate(err) {
			return ErrDuplicateKey
		}
		return fmt.Errorf("create favorite: %w", err)
	}
	return nil
}

func (r *favoriteRepository) Delete(userID, templateID uint64) error {
	res := r.db.Where("user_id = ? AND template_id = ?", userID, templateID).
		Delete(&model.TemplateFavorite{})
	if res.Error != nil {
		return fmt.Errorf("delete favorite: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *favoriteRepository) Exists(userID, templateID uint64) (bool, error) {
	var count int64
	if err := r.db.Model(&model.TemplateFavorite{}).
		Where("user_id = ? AND template_id = ?", userID, templateID).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("check favorite exists: %w", err)
	}
	return count > 0, nil
}

func (r *favoriteRepository) ListByUser(userID uint64, offset, limit int) ([]model.ContractTemplate, int64, error) {
	var total int64
	if err := r.db.Model(&model.TemplateFavorite{}).
		Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count favorites: %w", err)
	}
	var list []model.ContractTemplate
	err := r.db.Table("contract_templates").
		Joins("JOIN template_favorites ON template_favorites.template_id = contract_templates.id").
		Where("template_favorites.user_id = ?", userID).
		Order("template_favorites.id DESC").
		Offset(offset).Limit(limit).
		Find(&list).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list favorite templates: %w", err)
	}
	return list, total, nil
}
