package repository

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/contractapi/contractapi/internal/model"
)

// TemplateRepository 合同模板数据访问接口。
type TemplateRepository interface {
	List(category, keyword string, offset, limit int) ([]model.ContractTemplate, int64, error)
	FindByID(id uint64) (*model.ContractTemplate, error)
	Create(template *model.ContractTemplate) error
	Update(template *model.ContractTemplate) error
	Delete(id uint64) error
}

type templateRepository struct {
	db *gorm.DB
}

// NewTemplateRepository 构造合同模板仓储。
func NewTemplateRepository(db *gorm.DB) TemplateRepository {
	return &templateRepository{db: db}
}

func (r *templateRepository) List(category, keyword string, offset, limit int) ([]model.ContractTemplate, int64, error) {
	query := r.db.Model(&model.ContractTemplate{})
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR description LIKE ? OR code LIKE ?", like, like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count templates: %w", err)
	}
	var list []model.ContractTemplate
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list templates: %w", err)
	}
	return list, total, nil
}

func (r *templateRepository) FindByID(id uint64) (*model.ContractTemplate, error) {
	var template model.ContractTemplate
	if err := r.db.First(&template, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find template by id: %w", err)
	}
	return &template, nil
}

func (r *templateRepository) Create(template *model.ContractTemplate) error {
	if err := r.db.Create(template).Error; err != nil {
		if isDuplicate(err) {
			return ErrDuplicateKey
		}
		return fmt.Errorf("create template: %w", err)
	}
	return nil
}

func (r *templateRepository) Update(template *model.ContractTemplate) error {
	if err := r.db.Save(template).Error; err != nil {
		if isDuplicate(err) {
			return ErrDuplicateKey
		}
		return fmt.Errorf("update template: %w", err)
	}
	return nil
}

func (r *templateRepository) Delete(id uint64) error {
	if err := r.db.Delete(&model.ContractTemplate{}, id).Error; err != nil {
		return fmt.Errorf("delete template: %w", err)
	}
	return nil
}
