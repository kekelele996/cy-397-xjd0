package service

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/contractapi/contractapi/internal/dto"
	"github.com/contractapi/contractapi/internal/model"
	"github.com/contractapi/contractapi/internal/repository"
	"github.com/contractapi/contractapi/pkg/jwtutil"
)

// AuthService 注册、登录与 JWT 签发业务。
type AuthService struct {
	userRepo  repository.UserRepository
	jwt       *jwtutil.Manager
	logger    *slog.Logger
}

// NewAuthService 构造认证服务。
func NewAuthService(userRepo repository.UserRepository, jwt *jwtutil.Manager, logger *slog.Logger) *AuthService {
	return &AuthService{userRepo: userRepo, jwt: jwt, logger: logger}
}

// Register 创建新用户。
func (s *AuthService) Register(username, password string) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("register: hash password: %w", err)
	}
	user := &model.User{Username: username, PasswordHash: string(hash)}
	if err := s.userRepo.Create(user); err != nil {
		if errors.Is(err, repository.ErrDuplicateKey) {
			return nil, dto.ConflictError("username already exists")
		}
		return nil, fmt.Errorf("register: create user: %w", err)
	}
	s.logger.Info("user registered", "user_id", user.ID, "username", username)
	return user, nil
}

// Login 校验密码并签发 JWT。
func (s *AuthService) Login(username, password string) (string, time.Time, *model.User, error) {
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", time.Time{}, nil, dto.UnauthorizedError("invalid username or password")
		}
		return "", time.Time{}, nil, fmt.Errorf("login: find user: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", time.Time{}, nil, dto.UnauthorizedError("invalid username or password")
	}
	token, expiresAt, err := s.jwt.Generate(user.ID, user.Username)
	if err != nil {
		return "", time.Time{}, nil, fmt.Errorf("login: generate token: %w", err)
	}
	s.logger.Info("user logged in", "user_id", user.ID, "username", username)
	return token, expiresAt, user, nil
}
