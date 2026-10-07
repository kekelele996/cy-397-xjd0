package repository

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/contractapi/contractapi/internal/model"
)

// KnowledgeRepository 法律 FAQ 数据访问接口。
type KnowledgeRepository interface {
	List(category, keyword string, offset, limit int) ([]model.KnowledgeFAQ, int64, error)
	FindByID(id uint64) (*model.KnowledgeFAQ, error)
	Create(faq *model.KnowledgeFAQ) error
	Update(faq *model.KnowledgeFAQ) error
	Delete(id uint64) error
}

type knowledgeRepository struct {
	db *gorm.DB
}

// NewKnowledgeRepository 构造 FAQ 仓储。
func NewKnowledgeRepository(db *gorm.DB) KnowledgeRepository {
	return &knowledgeRepository{db: db}
}

func (r *knowledgeRepository) List(category, keyword string, offset, limit int) ([]model.KnowledgeFAQ, int64, error) {
	query := r.db.Model(&model.KnowledgeFAQ{})
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("question LIKE ? OR answer LIKE ?", like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count faqs: %w", err)
	}
	var list []model.KnowledgeFAQ
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list faqs: %w", err)
	}
	return list, total, nil
}

func (r *knowledgeRepository) FindByID(id uint64) (*model.KnowledgeFAQ, error) {
	var faq model.KnowledgeFAQ
	if err := r.db.First(&faq, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find faq by id: %w", err)
	}
	return &faq, nil
}

func (r *knowledgeRepository) Create(faq *model.KnowledgeFAQ) error {
	if err := r.db.Create(faq).Error; err != nil {
		return fmt.Errorf("create faq: %w", err)
	}
	return nil
}

func (r *knowledgeRepository) Update(faq *model.KnowledgeFAQ) error {
	if err := r.db.Save(faq).Error; err != nil {
		return fmt.Errorf("update faq: %w", err)
	}
	return nil
}

func (r *knowledgeRepository) Delete(id uint64) error {
	if err := r.db.Delete(&model.KnowledgeFAQ{}, id).Error; err != nil {
		return fmt.Errorf("delete faq: %w", err)
	}
	return nil
}
