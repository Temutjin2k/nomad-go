package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	"github.com/golang-jwt/jwt/v5"
)

type TokenService struct {
	userRepo   UserRepo
	RefreshTTL time.Duration
	AccessTTL  time.Duration
	log        logger.Logger
	secret     string
}

func NewTokenService(secret string, userRepo UserRepo, RefreshTTL time.Duration, AccessTTL time.Duration, log logger.Logger) *TokenService {
	return &TokenService{
		userRepo:   userRepo,
		RefreshTTL: RefreshTTL,
		AccessTTL:  AccessTTL,
		secret:     secret,
		log:        log,
	}
}

func (s *TokenService) getSecret() string {
	return s.secret
}

func (s *TokenService) GenerateTokens(user *models.User) (*models.TokenPair, error) {
	var signed []string

	claims := []jwt.Claims{
		NewAccessClaim(user, s.AccessTTL),
		NewRefreshClaim(user, s.RefreshTTL),
	}

	for _, claim := range claims {
		// Подпись каждого jwt токена
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)
		signedToken, err := token.SignedString([]byte(s.getSecret()))
		if err != nil {
			return nil, err
		}
		signed = append(signed, signedToken)
	}
	return &models.TokenPair{
		AccessExpiresAt:  time.Now().Add(s.AccessTTL),
		RefreshExpiresAt: time.Now().Add(s.RefreshTTL),
		AccessToken:      signed[0],
		RefreshToken:     signed[1],
	}, nil
}

func NewAccessClaim(user *models.User, accessTTL time.Duration) jwt.Claims {
	return jwt.MapClaims{
		"typ":     models.AccessToken,
		"user_id": user.ID.String(),
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(accessTTL).Unix(),
	}
}

func NewRefreshClaim(user *models.User, refreshTTL time.Duration) jwt.Claims {
	return jwt.MapClaims{
		"typ":     models.RefreshToken,
		"user_id": user.ID.String(),
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(refreshTTL).Unix(),
	}
}

func (s *TokenService) Refresh(refreshToken string) (*models.TokenPair, error) {
	ctx := wrap.WithAction(context.Background(), "refresh_token")

	claims, err := s.Validate(refreshToken)
	if err != nil {
		s.log.Error(ctx, "Refresh token is invalid", err)
		return nil, ErrInvalidToken
	}

	// Проверяем существует ли пользователь
	user, err := s.userRepo.GetUser(ctx, claims.Email)
	if err != nil {
		s.log.Error(ctx, "Failed to check user uniqueness", err)
		return nil, ErrUnexpected
	}

	if user == nil {
		return nil, ErrUserWithEmailNotFound
	}

	pair, err := s.GenerateTokens(user)
	if err != nil {
		s.log.Error(ctx, "Failed to generate tokens", err)
		return nil, ErrUnexpected
	}

	return pair, nil
}

func (s *TokenService) Validate(token string) (*models.CustomClaims, error) {
	parsedToken, err := jwt.ParseWithClaims(token, jwt.MapClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}
		return []byte(s.getSecret()), nil
	})
	if err != nil || !parsedToken.Valid {
		return nil, ErrInvalidToken
	}

	mc, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	if typ, _ := mc["typ"].(string); typ == "" {
		return nil, fmt.Errorf("invalid or missing 'typ' in token claims")
	}

	userIDStr, _ := mc["user_id"].(string)
	if userIDStr == "" {
		return nil, fmt.Errorf("invalid or missing 'user_id' in token claims")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid 'user_id' in token claims")
	}

	email, _ := mc["email"].(string)
	role, _ := mc["role"].(string)

	// exp → time.Time и проверка истечения
	expFloat, ok := mc["exp"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid or missing 'exp' in token claims")
	}

	expTime := time.Unix(int64(expFloat), 0)
	if time.Now().After(expTime) {
		return nil, ErrExpToken
	}

	claims := &models.CustomClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expTime),
		},
	}

	return claims, nil
}
