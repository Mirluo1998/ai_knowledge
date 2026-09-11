package service

import (
	"context"
	"knowledge/internal/model"
	"log/slog"

	"golang.org/x/crypto/bcrypt"
)

type UserRepository interface {
	AddUser(ctx context.Context, user model.User) (bool, error)
	QueryUser(ctx context.Context, query model.UserQuery) ([]*model.User, error)
	Login(ctx context.Context, user model.User) (bool, error)
}

type UserService struct {
	repo   UserRepository
	logger *slog.Logger
}

func NewUserService(repo UserRepository, logger *slog.Logger) *UserService {
	return &UserService{repo: repo, logger: logger}
}

func (s *UserService) RegisterUser(ctx context.Context, user model.User) (bool, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return false, err
	}
	user.Password = string(hash)
	return s.repo.AddUser(ctx, user)
}

func (s *UserService) GetUser(ctx context.Context, query model.UserQuery) ([]*model.User, error) {
	return s.repo.QueryUser(ctx, query)
}

func (s *UserService) Login(ctx context.Context, user model.User) (bool, error) {
	return s.repo.Login(ctx, user)
}
