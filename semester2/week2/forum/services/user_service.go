package services

import (
	"context"

	domainErrors "forum/errors"
	"forum/models"
	"forum/repositories"

	"golang.org/x/crypto/bcrypt"
)

type userService struct {
	userRepo   repositories.UserRepository
	jwtService JWTService
}

func NewUserService(userRepo repositories.UserRepository, jwtService JWTService) UserService {
	return &userService{
		userRepo:   userRepo,
		jwtService: jwtService,
	}
}

func (s *userService) Register(username, email, password string) (*models.AuthResponse, error) {
	ctx := context.Background()

	existingUser, _ := s.userRepo.FindByEmail(ctx, email)
	if existingUser != nil {
		return nil, domainErrors.ErrUserAlreadyExists
	}

	existingUser, _ = s.userRepo.FindByUsername(ctx, username)
	if existingUser != nil {
		return nil, domainErrors.ErrUserAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Username: username,
		Email:    email,
		Password: string(hashedPassword),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	token, err := s.jwtService.GenerateToken(user.ID.Hex())
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		Token: token,
		User:  *user,
	}, nil
}

func (s *userService) Login(email, password string) (*models.AuthResponse, error) {
	ctx := context.Background()

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, domainErrors.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, domainErrors.ErrInvalidCredentials
	}

	token, err := s.jwtService.GenerateToken(user.ID.Hex())
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		Token: token,
		User:  *user,
	}, nil
}
