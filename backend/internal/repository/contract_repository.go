package repository

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/contractapi/contractapi/internal/model"
)

// ContractRepository 合同数据访问接口。
type ContractRepository interface {
	Create(contract *model.Contract) error
	FindByID(id uint64) (*model.Contract, error)
	FindByIDForUser(id, userID uint64) (*model.Contract, error)
	ListByUser(userID uint64, status string, offset, limit int) ([]model.Contract, int64, error)
	Update(contract *model.Contract) error
	AddSigner(signer *model.ContractSigner) error
	ListSigners(contractID uint64) ([]model.ContractSigner, error)
}

type contractRepository struct {
	db *gorm.DB
}

// NewContractRepository 构造合同仓储。
func NewContractRepository(db *gorm.DB) ContractRepository {
	return &contractRepository{db: db}
}

func (r *contractRepository) Create(contract *model.Contract) error {
	if err := r.db.Create(contract).Error; err != nil {
		return fmt.Errorf("create contract: %w", err)
	}
	return nil
}

func (r *contractRepository) FindByID(id uint64) (*model.Contract, error) {
	var contract model.Contract
	if err := r.db.First(&contract, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find contract by id: %w", err)
	}
	return &contract, nil
}

func (r *contractRepository) FindByIDForUser(id, userID uint64) (*model.Contract, error) {
	var contract model.Contract
	if err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&contract).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find contract by id and user: %w", err)
	}
	return &contract, nil
}

func (r *contractRepository) ListByUser(userID uint64, status string, offset, limit int) ([]model.Contract, int64, error) {
	query := r.db.Model(&model.Contract{}).Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count contracts: %w", err)
	}
	var list []model.Contract
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list contracts: %w", err)
	}
	return list, total, nil
}

func (r *contractRepository) Update(contract *model.Contract) error {
	if err := r.db.Save(contract).Error; err != nil {
		return fmt.Errorf("update contract: %w", err)
	}
	return nil
}

func (r *contractRepository) AddSigner(signer *model.ContractSigner) error {
	if err := r.db.Create(signer).Error; err != nil {
		return fmt.Errorf("add contract signer: %w", err)
	}
	return nil
}

func (r *contractRepository) ListSigners(contractID uint64) ([]model.ContractSigner, error) {
	var list []model.ContractSigner
	if err := r.db.Where("contract_id = ?", contractID).Order("id ASC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list contract signers: %w", err)
	}
	return list, nil
}
