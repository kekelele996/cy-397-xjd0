package repository

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/contractapi/contractapi/internal/model"
)

// UserRepository 用户数据访问接口。
type UserRepository interface {
	Create(user *model.User) error
	FindByUsername(username string) (*model.User, error)
	FindByID(id uint64) (*model.User, error)
}

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 构造用户仓储。
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *model.User) error {
	if err := r.db.Create(user).Error; err != nil {
		if isDuplicate(err) {
			return ErrDuplicateKey
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *userRepository) FindByUsername(username string) (*model.User, error) {
	var user model.User
	if err := r.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find user by username: %w", err)
	}
	return &user, nil
}

func (r *userRepository) FindByID(id uint64) (*model.User, error) {
	var user model.User
	if err := r.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return &user, nil
}
