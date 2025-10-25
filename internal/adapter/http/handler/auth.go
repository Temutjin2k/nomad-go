package handler

import (
	"context"
	"net/http"

	"github.com/Temutjin2k/ride-hail-system/internal/adapter/http/handler/dto"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	"github.com/Temutjin2k/ride-hail-system/pkg/validator"
)

type AuthService interface {
	Register(ctx context.Context, newUser *models.UserCreateRequest) (uuid.UUID, error)
	Login(ctx context.Context, email, password string) (*models.TokenPair, error)
	Refresh(ctx context.Context, refreshToken string) (*models.TokenPair, error)
	RoleCheck(ctx context.Context, token string) (*models.User, error)
}

type Auth struct {
	auth AuthService
	l    logger.Logger
}

func NewAuth(service AuthService, l logger.Logger) *Auth {
	return &Auth{
		auth: service,
		l:    l,
	}
}

func (h *Auth) Register(w http.ResponseWriter, r *http.Request) {
	ctx := wrap.WithAction(r.Context(), "register_user")

	req := &dto.RegisterUserRequest{}
	if err := readJSON(w, r, req); err != nil {
		h.l.Error(ctx, "failed to read request JSON data", err)
		badRequestResponse(w, err.Error())
		return
	}

	v := validator.New()
	dto.ValidateNewUser(v, req)

	if !v.Valid() {
		failedValidationResponse(w, v.Errors)
		return
	}

	id, err := h.auth.Register(ctx, req.ToModel())
	if err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to register a new user", err)
		errorResponse(w, GetCode(err), err.Error())
		return
	}

	response := envelope{"id": id}
	if err := writeJSON(w, http.StatusCreated, response, nil); err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to write JSON response", err)
		internalErrorResponse(w, "failed to write JSON response")
	}
}

func (h *Auth) Login(w http.ResponseWriter, r *http.Request) {
	ctx := wrap.WithAction(r.Context(), "login_user")

	req := &dto.LoginRequest{}
	if err := readJSON(w, r, req); err != nil {
		badRequestResponse(w, err.Error())
		return
	}

	v := validator.New()
	dto.ValidateLogin(v, req)
	if !v.Valid() {
		failedValidationResponse(w, v.Errors)
		return
	}

	tokens, err := h.auth.Login(ctx, req.Email, req.Password)
	if err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to login user", err)
		errorResponse(w, GetCode(err), err.Error())
		return
	}

	response := envelope{
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
	}

	if err := writeJSON(w, http.StatusOK, response, nil); err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to write JSON response", err)
		internalErrorResponse(w, "failed to write JSON response")
	}
}

func (h *Auth) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx := wrap.WithAction(r.Context(), "refresh_token")

	req := &dto.RefreshTokenRequest{}
	if err := readJSON(w, r, req); err != nil {
		badRequestResponse(w, err.Error())
		return
	}

	v := validator.New()
	dto.ValidateRefreshToken(v, req)
	if !v.Valid() {
		failedValidationResponse(w, v.Errors)
		return
	}

	tokens, err := h.auth.Refresh(ctx, req.RefreshToken)
	if err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to refresh token pair", err)
		errorResponse(w, GetCode(err), err.Error())
		return
	}

	response := envelope{
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
	}

	if err := writeJSON(w, http.StatusOK, response, nil); err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to write JSON response", err)
		internalErrorResponse(w, "failed to write JSON response")
	}
}

func (h *Auth) Profile(w http.ResponseWriter, r *http.Request) {
	ctx := wrap.WithAction(r.Context(), "get_profile")

	user := models.UserFromContext(ctx)
	if user == nil {
		h.l.Warn(ctx, "failed to get profile")
		errorResponse(w, http.StatusNotFound, "failed to get profile")
		return
	}

	response := envelope{
		"user": user,
	}

	if err := writeJSON(w, http.StatusOK, response, nil); err != nil {
		h.l.Error(ctx, "failed to write JSON response", err)
		internalErrorResponse(w, "failed to write JSON response")
	}
}
