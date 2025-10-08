package models

import (
	"time"

	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

type UserCreateRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type User struct {
	ID         uuid.UUID `json:"ID"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	password   string    `json:"-"`
	Role       string    `json:"role"`
	Created_At time.Time `json:"created_at"`
	Updated_At time.Time `json:"updated_at,omitzero"`
}

func (u *User) GetPassword() string {
	return u.password
}

func (u *User) SetPassword(password string) {
	u.password = password
}
