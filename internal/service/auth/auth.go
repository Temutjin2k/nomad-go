package auth

import (
	"context"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	"github.com/Temutjin2k/ride-hail-system/pkg/hasher"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

type AuthService struct {
	userRepo     UserRepo
	tokenService TokenProvider
	log          logger.Logger
}

func NewAuthService(UserDal UserRepo, TokenServ TokenProvider, log logger.Logger) *AuthService {
	return &AuthService{
		userRepo:     UserDal,
		tokenService: TokenServ,
		log:          log,
	}
}

// Returns (AccessToken, RefreshToken, statusCode, error message)
func (s *AuthService) Login(ctx context.Context, email, password string) (*models.TokenPair, error) {
	// Проверяем существует ли пользователь
	user, err := s.userRepo.GetUser(ctx, email)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, types.ErrUserNotFound
	}

	// Проверяем пароль
	if ok := hasher.Verify(password, user.PasswordHash); !ok {
		return nil, ErrInvalidCredentials
	}

	// Генерируем токены
	tokens, err := s.tokenService.GenerateTokens(user)
	if err != nil {
		return nil, ErrTokenGenerateFail
	}

	return tokens, nil
}

// Register creates new passenger
func (s *AuthService) Register(ctx context.Context, user *models.UserCreateRequest) (uuid.UUID, error) {
	ctx = wrap.WithAction(ctx, "passenger_register")

	// Check if user with such email already exists
	u, err := s.userRepo.GetUser(ctx, user.Email)
	if err != nil {
		return uuid.UUID{}, wrap.Error(ctx, err)
	}

	// If user exists, return error
	if u != nil {
		return uuid.UUID{}, ErrNotUniqueEmail
	}

	// Hash password
	hashPassword := hasher.Hash(user.Password)

	// Save user
	newUser := models.User{
		Email:        user.Email,
		Role:         types.RolePassenger.String(),
		PasswordHash: hashPassword,
		Status:       types.StatusUserActive.String(),
	}

	id, err := s.userRepo.CreateUser(ctx, &newUser)
	if err != nil {
		return uuid.UUID{}, wrap.Error(ctx, err)
	}

	return id, nil
}

func (s *AuthService) RoleCheck(ctx context.Context, token string) (*models.User, error) {
	// Валидируем его
	claim, err := s.tokenService.Validate(token)
	if err != nil {
		return nil, err
	}

	// Проверяем существует ли пользователь
	user, err := s.userRepo.GetUser(ctx, claim.Email)
	if err != nil {
		return nil, wrap.Error(ctx, err)
	}

	if user == nil {
		return nil, ErrUserWithEmailNotFound
	}

	return user, nil
}
