package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	"github.com/Temutjin2k/ride-hail-system/pkg/hasher"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/trm"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	"github.com/golang-jwt/jwt/v5"
)

type TokenService struct {
	userRepo    UserRepo
	refreshRepo RefreshTokenRepo
	txManager   trm.TxManager
	RefreshTTL  time.Duration
	AccessTTL   time.Duration
	secret      string
	log         logger.Logger
}

func NewTokenService(secret string, userRepo UserRepo, refreshRepo RefreshTokenRepo, txManager trm.TxManager, RefreshTTL time.Duration, AccessTTL time.Duration, log logger.Logger) *TokenService {
	return &TokenService{
		userRepo:    userRepo,
		refreshRepo: refreshRepo,
		txManager:   txManager,
		RefreshTTL:  RefreshTTL,
		AccessTTL:   AccessTTL,
		secret:      secret,
		log:         log,
	}
}

func (s *TokenService) getSecret() string {
	return s.secret
}

// GenerateTokens creates a new pair of access and refresh tokens for the given user.
// The refresh token is stored in the database
// along with its hash, expiration time, and associated user ID.
func (s *TokenService) GenerateTokens(ctx context.Context, user *models.User) (*models.TokenPair, error) {
	ctx = wrap.WithAction(ctx, "generate_tokens")
	if user == nil {
		return nil, wrap.Error(ctx, errors.New("user is nil"))
	}

	issuedAt := time.Now().UTC()
	accessID := uuid.New()
	refreshID := uuid.New()

	accessExp := issuedAt.Add(s.AccessTTL)
	refreshExp := issuedAt.Add(s.RefreshTTL)

	accessClaims := NewAccessClaim(user, issuedAt, s.AccessTTL, accessID)
	accessToken, err := s.signClaims(accessClaims)
	if err != nil {
		return nil, wrap.Error(ctx, err)
	}

	refreshClaims := NewRefreshClaim(user, issuedAt, s.RefreshTTL, refreshID)
	refreshToken, err := s.signClaims(refreshClaims)
	if err != nil {
		return nil, wrap.Error(ctx, err)
	}

	if s.refreshRepo != nil {
		record := &models.RefreshTokenRecord{
			ID:        refreshID,
			UserID:    user.ID,
			TokenHash: hasher.Hash(refreshToken),
			ExpiresAt: refreshExp,
			Revoked:   false,
			CreatedAt: issuedAt,
		}

		if err := s.refreshRepo.Save(ctx, record); err != nil {
			return nil, wrap.Error(ctx, fmt.Errorf("failed to persist refresh token: %w", err))
		}
	}

	return &models.TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresAt:  accessExp,
		RefreshExpiresAt: refreshExp,
	}, nil
}

// Refresh refreshes the token pair using the provided refresh token.
// It validates the refresh token, checks its status in the database,
// marks it as used, and generates a new pair of tokens.
func (s *TokenService) Refresh(ctx context.Context, refreshToken string) (*models.TokenPair, error) {
	ctx = wrap.WithAction(ctx, "refresh_token")

	claims, err := s.Validate(ctx, refreshToken)
	if err != nil {
		return nil, wrap.Error(ctx, ErrInvalidToken)
	}

	if claims.TokenType != models.RefreshToken {
		return nil, wrap.Error(ctx, ErrInvalidToken)
	}

	var pair *models.TokenPair

	// Use a transaction to ensure atomicity of operations
	txErr := s.txManager.Do(ctx, func(txCtx context.Context) error {
		record, err := s.refreshRepo.Get(txCtx, claims.TokenID)
		if err != nil {
			return fmt.Errorf("failed to load refresh token record: %w", err)
		}

		if record == nil {
			return ErrInvalidToken
		}

		if record.Revoked {
			return ErrInvalidToken
		}

		now := time.Now().UTC()
		if now.After(record.ExpiresAt) {
			if err := s.refreshRepo.MarkUsed(txCtx, record.ID); err != nil {
				return fmt.Errorf("failed to revoke expired refresh token: %w", err)
			}
			return ErrExpToken
		}

		expectedHash := record.TokenHash
		actualHash := hasher.Hash(refreshToken)
		if expectedHash != actualHash {
			if err := s.refreshRepo.MarkUsed(txCtx, record.ID); err != nil {
				return fmt.Errorf("failed to revoke mismatched refresh token: %w", err)
			}
			return ErrInvalidToken
		}

		if err := s.refreshRepo.MarkUsed(txCtx, record.ID); err != nil {
			return fmt.Errorf("failed to mark refresh token as used: %w", err)
		}

		user, err := s.userRepo.GetUserByID(txCtx, claims.UserID)
		if err != nil {
			return fmt.Errorf("failed to load user for refresh token: %w", err)
		}

		if user == nil {
			return types.ErrUserNotFound
		}

		pair, err = s.GenerateTokens(txCtx, user)
		if err != nil {
			return err
		}

		return nil
	})

	if txErr != nil {
		return nil, wrap.Error(ctx, txErr)
	}

	return pair, nil
}

// Validate validates the given JWT token string, returning the custom claims if valid.
func (s *TokenService) Validate(ctx context.Context, token string) (*models.CustomClaims, error) {
	ctx = wrap.WithAction(ctx, "validate_token")

	parsedToken, err := jwt.ParseWithClaims(token, jwt.MapClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}
		return []byte(s.getSecret()), nil
	})
	if err != nil || !parsedToken.Valid {
		return nil, wrap.Error(ctx, ErrInvalidToken)
	}

	mc, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, wrap.Error(ctx, ErrInvalidToken)
	}

	typ, _ := mc["typ"].(string)
	if !models.IsValidTokenType(typ) {
		return nil, wrap.Error(ctx, ErrInvalidToken)
	}

	userIDStr, _ := mc["user_id"].(string)
	if userIDStr == "" {
		return nil, wrap.Error(ctx, fmt.Errorf("invalid or missing 'user_id' in token claims"))
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, wrap.Error(ctx, fmt.Errorf("invalid 'user_id' in token claims"))
	}

	tokenIDStr, _ := mc["jti"].(string)
	if tokenIDStr == "" {
		return nil, wrap.Error(ctx, fmt.Errorf("invalid or missing 'jti' in token claims"))
	}
	tokenID, err := uuid.Parse(tokenIDStr)
	if err != nil {
		return nil, wrap.Error(ctx, fmt.Errorf("invalid 'jti' in token claims"))
	}

	email, _ := mc["email"].(string)
	role, _ := mc["role"].(string)

	expFloat, ok := mc["exp"].(float64)
	if !ok {
		return nil, wrap.Error(ctx, fmt.Errorf("invalid or missing 'exp' in token claims"))
	}

	expTime := time.Unix(int64(expFloat), 0)
	if time.Now().UTC().After(expTime) {
		return nil, wrap.Error(ctx, ErrExpToken)
	}

	claims := &models.CustomClaims{
		UserID:    userID,
		TokenID:   tokenID,
		TokenType: typ,
		Email:     email,
		Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expTime),
		},
	}

	return claims, nil
}

func (s *TokenService) signClaims(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.getSecret()))
}

func NewAccessClaim(user *models.User, issuedAt time.Time, accessTTL time.Duration, tokenID uuid.UUID) jwt.Claims {
	return jwt.MapClaims{
		"typ":     models.AccessToken,
		"jti":     tokenID.String(),
		"user_id": user.ID.String(),
		"email":   user.Email,
		"role":    user.Role,
		"iat":     issuedAt.Unix(),
		"exp":     issuedAt.Add(accessTTL).Unix(),
	}
}

func NewRefreshClaim(user *models.User, issuedAt time.Time, refreshTTL time.Duration, tokenID uuid.UUID) jwt.Claims {
	return jwt.MapClaims{
		"typ":     models.RefreshToken,
		"jti":     tokenID.String(),
		"user_id": user.ID.String(),
		"iat":     issuedAt.Unix(),
		"exp":     issuedAt.Add(refreshTTL).Unix(),
	}
}
