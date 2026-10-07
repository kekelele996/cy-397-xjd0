package service

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/contractapi/contractapi/internal/dto"
	"github.com/contractapi/contractapi/internal/model"
	"github.com/contractapi/contractapi/internal/repository"
)

// KnowledgeService 法律 FAQ 检索与维护业务。
type KnowledgeService struct {
	knowledgeRepo repository.KnowledgeRepository
	logger        *slog.Logger
}

// NewKnowledgeService 构造 FAQ 服务。
func NewKnowledgeService(knowledgeRepo repository.KnowledgeRepository, logger *slog.Logger) *KnowledgeService {
	return &KnowledgeService{knowledgeRepo: knowledgeRepo, logger: logger}
}

// Search 按分类与关键词分页检索 FAQ。
func (s *KnowledgeService) Search(category, keyword string, page, pageSize int) ([]model.KnowledgeFAQ, int64, error) {
	offset := (page - 1) * pageSize
	list, total, err := s.knowledgeRepo.List(category, keyword, offset, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("search faqs: %w", err)
	}
	return list, total, nil
}

// Get 查询单个 FAQ。
func (s *KnowledgeService) Get(id uint64) (*model.KnowledgeFAQ, error) {
	faq, err := s.knowledgeRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, dto.NotFoundError("faq not found")
		}
		return nil, fmt.Errorf("get faq: %w", err)
	}
	return faq, nil
}

// Create 创建 FAQ。
func (s *KnowledgeService) Create(req dto.SaveKnowledgeRequest) (*model.KnowledgeFAQ, error) {
	faq := &model.KnowledgeFAQ{
		Category: req.Category,
		Question: req.Question,
		Answer:   req.Answer,
	}
	if err := s.knowledgeRepo.Create(faq); err != nil {
		return nil, fmt.Errorf("create faq: %w", err)
	}
	s.logger.Info("faq created", "faq_id", faq.ID)
	return faq, nil
}

// Update 更新 FAQ。
func (s *KnowledgeService) Update(id uint64, req dto.SaveKnowledgeRequest) (*model.KnowledgeFAQ, error) {
	faq, err := s.knowledgeRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, dto.NotFoundError("faq not found")
		}
		return nil, fmt.Errorf("update faq: find: %w", err)
	}
	faq.Category = req.Category
	faq.Question = req.Question
	faq.Answer = req.Answer
	if err := s.knowledgeRepo.Update(faq); err != nil {
		return nil, fmt.Errorf("update faq: save: %w", err)
	}
	s.logger.Info("faq updated", "faq_id", faq.ID)
	return faq, nil
}

// Delete 删除 FAQ。
func (s *KnowledgeService) Delete(id uint64) error {
	if _, err := s.knowledgeRepo.FindByID(id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return dto.NotFoundError("faq not found")
		}
		return fmt.Errorf("delete faq: find: %w", err)
	}
	if err := s.knowledgeRepo.Delete(id); err != nil {
		return fmt.Errorf("delete faq: %w", err)
	}
	s.logger.Info("faq deleted", "faq_id", id)
	return nil
}
