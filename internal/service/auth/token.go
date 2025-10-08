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
	UserDal    UserRepo
	RefreshTTL time.Duration
	AccessTTL  time.Duration
	log        logger.Logger
	secret     string
}

func NewTokenService(secret string, UserDal UserRepo, RefreshTTL time.Duration, AccessTTL time.Duration, log logger.Logger) *TokenService {
	return &TokenService{
		UserDal:    UserDal,
		RefreshTTL: RefreshTTL,
		AccessTTL:  AccessTTL,
		secret:     secret,
		log:        log,
	}
}

func (s *TokenService) getSecret() string {
	return s.secret
}

func (s *TokenService) GenerateTokens(user models.User) (models.TokenPair, error) {
	var signed []string
	for _, claim := range []jwt.Claims{NewAccessClaim(user, s.AccessTTL), NewRefreshClaim(user, s.RefreshTTL)} {
		// Подпись каждого jwt токена
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)
		signedToken, err := token.SignedString([]byte(s.getSecret()))
		if err != nil {
			return models.TokenPair{}, err
		}
		signed = append(signed, signedToken)
	}
	return models.TokenPair{
		AccessExpiresAt:  time.Now().Add(s.AccessTTL),
		RefreshExpiresAt: time.Now().Add(s.RefreshTTL),
		AccessToken:      signed[0],
		RefreshToken:     signed[1],
	}, nil
}

func NewAccessClaim(user models.User, accessTTL time.Duration) jwt.Claims {
	return jwt.MapClaims{
		"ID":         user.ID,
		"name":       user.Name,
		"email":      user.Email,
		"is_refresh": false,
		"role":       user.Role,
		"exp":        time.Now().Add(accessTTL).Unix(),
	}
}

func NewRefreshClaim(user models.User, refreshTTL time.Duration) jwt.Claims {
	return jwt.MapClaims{
		"ID":         user.ID,
		"name":       user.Name,
		"email":      user.Email,
		"is_refresh": true,
		"role":       user.Role,
		"exp":        time.Now().Add(refreshTTL).Unix(),
	}
}

func (s *TokenService) Refresh(refreshToken string) (models.TokenPair, error) {
	ctx := wrap.WithAction(context.Background(), "refresh_token")

	claims, err := s.Validate(refreshToken)
	if err != nil {
		s.log.Error(ctx, "Refresh token is invalid", err)
		return models.TokenPair{}, ErrInvalidToken
	}

	// Проверяем существует ли пользователь
	user, err := s.UserDal.GetUser(claims.Email)
	if err != nil {
		s.log.Error(ctx, "Failed to check user uniqueness", err)
		return models.TokenPair{}, ErrUnexpected
	}

	pair, err := s.GenerateTokens(*user)
	if err != nil {
		s.log.Error(ctx, "Failed to generate tokens", err)
		return models.TokenPair{}, ErrUnexpected
	}

	return pair, nil
}

func (s *TokenService) Validate(token string) (models.CustomClaims, error) {
	parsedToken, err := jwt.ParseWithClaims(token, jwt.MapClaims{}, func(t *jwt.Token) (any, error) {
		return []byte(s.getSecret()), nil
	})
	if err != nil {
		return models.CustomClaims{}, ErrInvalidToken
	}

	if !parsedToken.Valid {
		return models.CustomClaims{}, ErrInvalidToken
	}

	mapClaims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return models.CustomClaims{}, ErrInvalidToken
	}

	var claims models.CustomClaims
	var invOrMissingForm string = "invalid or missing '%s' in token claims"

	// Извлекаем Name
	if id, ok := mapClaims["ID"].(uuid.UUID); ok {
		claims.ID = id
	} else {
		return models.CustomClaims{}, fmt.Errorf(invOrMissingForm, "ID")
	}

	if role, ok := mapClaims["role"].(string); ok {
		claims.Role = role
	} else {
		return models.CustomClaims{}, fmt.Errorf(invOrMissingForm, "role")
	}

	// Извлекаем Email
	if email, ok := mapClaims["email"].(string); ok {
		claims.Email = email
	} else {
		return models.CustomClaims{}, fmt.Errorf(invOrMissingForm, "email")
	}

	// Извлекаем Name
	if name, ok := mapClaims["name"].(string); ok {
		claims.Name = name
	} else {
		return models.CustomClaims{}, fmt.Errorf(invOrMissingForm, "name")
	}

	if isRefresh, ok := mapClaims["is_refresh"].(bool); ok {
		claims.IsRefresh = isRefresh
	} else {
		return models.CustomClaims{}, fmt.Errorf(invOrMissingForm, "is_refresh")
	}

	// Извлекаем exp и проверяем время
	expFloat, ok := mapClaims["exp"].(float64)
	if !ok {
		return models.CustomClaims{}, fmt.Errorf(invOrMissingForm, "exp")
	}
	expTime := time.Unix(int64(expFloat), 0)
	claims.ExpiresAt = jwt.NewNumericDate(expTime)

	if time.Now().After(expTime) {
		return models.CustomClaims{}, ErrExpToken
	}

	return claims, nil
}
