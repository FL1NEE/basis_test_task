package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/FL1NEE/basis_test_task/internal/auth"
	"github.com/FL1NEE/basis_test_task/internal/domain"
	"github.com/FL1NEE/basis_test_task/internal/repository"
	"github.com/go-sql-driver/mysql"
)

type AuthService struct {
	users  *repository.UserRepo
	tokens *auth.TokenIssuer
}

func NewAuthService(users *repository.UserRepo, tokens *auth.TokenIssuer) *AuthService {
	return &AuthService{users: users, tokens: tokens}
}

func (s *AuthService) Register(ctx context.Context, email, password, name string) (int64, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" || name == "" {
		return 0, domain.ErrValidation
	}
	if len(password) < 8 {
		return 0, fmt.Errorf("%w: password must be at least 8 characters", domain.ErrValidation)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return 0, fmt.Errorf("hash password: %w", err)
	}

	id, err := s.users.Create(ctx, email, hash, name)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return 0, domain.ErrEmailTaken
		}
		return 0, err
	}
	return id, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, sql.ErrNoRows) {
			return "", domain.ErrInvalidCredentials
		}
		return "", err
	}

	if !auth.CheckPassword(user.PasswordHash, password) {
		return "", domain.ErrInvalidCredentials
	}

	token, err := s.tokens.Issue(user.ID)
	if err != nil {
		return "", fmt.Errorf("issue token: %w", err)
	}
	return token, nil
}
