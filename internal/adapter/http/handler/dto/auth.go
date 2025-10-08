package dto

import (
	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/pkg/validator"
)

func ValidateNewUser(v *validator.Validator, user *models.UserCreateRequest) {
	v.Check(user.Name != "", "name", "must be provided")
	v.Check(len(user.Name) <= 500, "name", "must not be more than 500 bytes long")

	v.Check(user.Email != "", "email", "must be provided")
	v.Check(validator.Matches(user.Email, validator.EmailRX), "email", "must be a valid email address")
	v.Check(len(user.Email) <= 500, "email", "must not be more than 500 bytes long")

	v.Check(user.Password != "", "password", "must be provided")
	v.Check(len(user.Password) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(user.Password) <= 50, "password", "must not be more than 50 bytes long")
}
