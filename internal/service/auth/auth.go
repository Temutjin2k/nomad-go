package auth

import (
	"context"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/passhash"
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
	user, err := s.userRepo.GetUser(email)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, ErrUserWithEmailNotFound
	}

	// Проверяем пароль
	if ok, err := passhash.VerifyPassword(password, user.GetPassword()); err != nil || !ok {
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
	u, err := s.userRepo.GetUser(user.Email)
	if err != nil {
		return uuid.UUID{}, ErrUnexpected
	}

	// If user exists, return error
	if u != nil {
		return uuid.UUID{}, ErrNotUniqueEmail
	}

	// Hash password
	hashPassword, err := passhash.HashPassword(user.Password)
	if err != nil {
		s.log.Error(ctx, "Failed to generate hash from password", err)
		return uuid.UUID{}, ErrUnexpected
	}

	// Save user
	newUser := models.User{
		Name:  user.Name,
		Email: user.Email,
		Role:  types.PassengerRole.String(),
	}
	newUser.SetPassword(hashPassword)

	id, err := s.userRepo.CreateUser(&newUser)
	if err != nil {
		s.log.Error(ctx, "Failed to save user", err)
		return uuid.UUID{}, ErrUnexpected
	}

	return id, nil
}

func (s *AuthService) RoleCheck(token string) (*models.User, error) {
	// Валидируем его
	claim, err := s.tokenService.Validate(token)
	if err != nil {
		s.log.Error(context.Background(), "Access token is invalid", err)
		return nil, ErrInvalidToken
	}

	// Проверяем существует ли пользователь
	existUser, err := s.userRepo.GetUser(claim.Email)
	if err != nil {
		return nil, ErrUnexpected
	}

	// Читаем админ ли он
	return existUser, nil
}
