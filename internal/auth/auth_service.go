package auth

import (
	"context"
	"errors"
	"uptime/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo domain.UserRepository
	jwtSvc   *JWTService
}

func NewAuthService(userRepo domain.UserRepository, jwtSvc *JWTService) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		jwtSvc:   jwtSvc,
	}
}

func (s *AuthService) Register(ctx context.Context, email, password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	user, err := s.userRepo.Create(ctx, email, string(hash))
	if err != nil {
		return "", err
	}

	return s.jwtSvc.GenerateToken(user.ID)
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", errors.New("invalid credentials")
	}

	return s.jwtSvc.GenerateToken(user.ID)
}
