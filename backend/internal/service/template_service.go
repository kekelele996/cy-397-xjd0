package service

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/contractapi/contractapi/internal/dto"
	"github.com/contractapi/contractapi/internal/model"
	"github.com/contractapi/contractapi/internal/repository"
)

// TemplateService 合同模板与收藏业务。
type TemplateService struct {
	templateRepo repository.TemplateRepository
	favoriteRepo repository.FavoriteRepository
	logger       *slog.Logger
}

// NewTemplateService 构造模板服务。
func NewTemplateService(
	templateRepo repository.TemplateRepository,
	favoriteRepo repository.FavoriteRepository,
	logger *slog.Logger,
) *TemplateService {
	return &TemplateService{templateRepo: templateRepo, favoriteRepo: favoriteRepo, logger: logger}
}

// List 分页查询模板。
func (s *TemplateService) List(category, keyword string, page, pageSize int) ([]model.ContractTemplate, int64, error) {
	offset := (page - 1) * pageSize
	list, total, err := s.templateRepo.List(category, keyword, offset, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("list templates: %w", err)
	}
	return list, total, nil
}

// Get 查询模板详情。
func (s *TemplateService) Get(id uint64) (*model.ContractTemplate, error) {
	template, err := s.templateRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, dto.NotFoundError("template not found")
		}
		return nil, fmt.Errorf("get template: %w", err)
	}
	return template, nil
}

// Create 创建模板。
func (s *TemplateService) Create(req dto.SaveTemplateRequest) (*model.ContractTemplate, error) {
	template := &model.ContractTemplate{
		Code:        req.Code,
		Name:        req.Name,
		Category:    req.Category,
		Description: req.Description,
		Content:     req.Content,
		ContentHTML: req.ContentHTML,
		Variables:   req.Variables,
	}
	if err := s.templateRepo.Create(template); err != nil {
		if errors.Is(err, repository.ErrDuplicateKey) {
			return nil, dto.ConflictError("template code already exists")
		}
		return nil, fmt.Errorf("create template: %w", err)
	}
	s.logger.Info("template created", "template_id", template.ID, "code", template.Code)
	return template, nil
}

// Update 更新模板。
func (s *TemplateService) Update(id uint64, req dto.SaveTemplateRequest) (*model.ContractTemplate, error) {
	template, err := s.templateRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, dto.NotFoundError("template not found")
		}
		return nil, fmt.Errorf("update template: find: %w", err)
	}
	template.Code = req.Code
	template.Name = req.Name
	template.Category = req.Category
	template.Description = req.Description
	template.Content = req.Content
	template.ContentHTML = req.ContentHTML
	template.Variables = req.Variables
	if err := s.templateRepo.Update(template); err != nil {
		if errors.Is(err, repository.ErrDuplicateKey) {
			return nil, dto.ConflictError("template code already exists")
		}
		return nil, fmt.Errorf("update template: save: %w", err)
	}
	s.logger.Info("template updated", "template_id", template.ID)
	return template, nil
}

// Delete 删除模板。
func (s *TemplateService) Delete(id uint64) error {
	if _, err := s.templateRepo.FindByID(id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return dto.NotFoundError("template not found")
		}
		return fmt.Errorf("delete template: find: %w", err)
	}
	if err := s.templateRepo.Delete(id); err != nil {
		return fmt.Errorf("delete template: %w", err)
	}
	s.logger.Info("template deleted", "template_id", id)
	return nil
}

// Favorite 收藏模板。
func (s *TemplateService) Favorite(userID, templateID uint64) error {
	if _, err := s.templateRepo.FindByID(templateID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return dto.NotFoundError("template not found")
		}
		return fmt.Errorf("favorite template: find: %w", err)
	}
	if err := s.favoriteRepo.Create(userID, templateID); err != nil {
		if errors.Is(err, repository.ErrDuplicateKey) {
			return dto.ConflictError("template already favorited")
		}
		return fmt.Errorf("favorite template: %w", err)
	}
	return nil
}

// Unfavorite 取消收藏模板。
func (s *TemplateService) Unfavorite(userID, templateID uint64) error {
	if err := s.favoriteRepo.Delete(userID, templateID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return dto.NotFoundError("favorite not found")
		}
		return fmt.Errorf("unfavorite template: %w", err)
	}
	return nil
}

// Favorites 查询用户收藏模板。
func (s *TemplateService) Favorites(userID uint64, page, pageSize int) ([]model.ContractTemplate, int64, error) {
	offset := (page - 1) * pageSize
	list, total, err := s.favoriteRepo.ListByUser(userID, offset, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("list favorites: %w", err)
	}
	return list, total, nil
}
