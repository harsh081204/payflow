package service

import (
	"context"
	"errors"
	"time"
	"user-service/models"
	"user-service/repository"
	"user-service/util"

	"github.com/google/uuid"
)

var (
	ErrUserExists   = errors.New("user already exists")
	ErrInvalidCreds = errors.New("invalid email or password")
	ErrUserNotFound = errors.New("user not found")
)

type UserService interface {
	Register(ctx context.Context, req *models.RegisterRequest) (*models.User, error)
	Login(ctx context.Context, req *models.LoginRequest, jwtSecret string) (string, *models.User, error)
	GetUser(ctx context.Context, id uuid.UUID) (*models.User, error)
}

type userService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return &userService{repo: repo}
}

func (s *userService) Register(ctx context.Context, req *models.RegisterRequest) (*models.User, error) {
	existingUser, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrUserExists
	}

	hashed, err := util.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		ID:        uuid.New(),
		Email:     req.Email,
		Password:  hashed,
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userService) Login(ctx context.Context, req *models.LoginRequest, jwtSecret string) (string, *models.User, error) {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return "", nil, err
	}
	if user == nil {
		return "", nil, ErrInvalidCreds
	}

	if !util.CheckPasswordHash(req.Password, user.Password) {
		return "", nil, ErrInvalidCreds
	}

	token, err := util.GenerateJWT(user.ID, user.Role, jwtSecret)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil
}

func (s *userService) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}
