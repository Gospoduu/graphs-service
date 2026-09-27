package dto

import (
	"github.com/google/uuid"
)

type UserResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type CreateUserRequest struct {
	Name string `json:"name" binding:"required"`
}
