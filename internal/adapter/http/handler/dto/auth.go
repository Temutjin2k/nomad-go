package dto

import (
	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/pkg/validator"
)

type RegisterUserRequest struct {
	Name     string         `json:"name"`
	Phone    string         `json:"phone"`
	Email    string         `json:"email"`
	Password string         `json:"password"`
	Attrs    map[string]any `json:"attrs,omitempty"`
}

func (r *RegisterUserRequest) ToModel() *models.UserCreateRequest {
	r.Attrs["name"] = r.Name
	r.Attrs["phone"] = r.Phone
	return &models.UserCreateRequest{
		Name:     r.Name,
		Email:    r.Email,
		Password: r.Password,
		Attrs:    r.Attrs,
	}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func ValidateNewUser(v *validator.Validator, user *RegisterUserRequest) {
	v.Check(user.Name != "", "name", "must be provided")
	v.Check(len(user.Name) <= 500, "name", "must not be more than 500 bytes long")

	v.Check(user.Email != "", "email", "must be provided")
	v.Check(validator.Matches(user.Email, validator.EmailRX), "email", "must be a valid email address")
	v.Check(len(user.Email) <= 500, "email", "must not be more than 500 bytes long")

	v.Check(user.Password != "", "password", "must be provided")
	v.Check(len(user.Password) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(user.Password) <= 50, "password", "must not be more than 50 bytes long")
}

func ValidateLogin(v *validator.Validator, user *LoginRequest) {
	v.Check(user.Email != "", "email", "must be provided")
	v.Check(user.Password != "", "password", "must be provided")
}

func ValidateRefreshToken(v *validator.Validator, req *RefreshTokenRequest) {
	v.Check(req.RefreshToken != "", "refresh_token", "must be provided")
}

type AuthWebSocketReq struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

type AuthWebSocketResp struct {
	Type        string `json:"type"`
	PassengerID string `json:"passenger_id"`
}
