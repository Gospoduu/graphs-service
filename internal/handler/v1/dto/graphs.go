package dto

import (
	"github.com/google/uuid"
)

type GraphResponse struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	UserID     uuid.UUID `json:"user_id"`
	IsDirected bool      `json:"is_directed"`
}

type CreateGraphRequest struct {
	Name       string    `json:"name" binding:"required"`
	UserID     uuid.UUID `json:"user_id" binding:"required"`
	IsDirected bool      `json:"is_directed"`
}

type ToggleIsDirectedResponse struct {
	ID         uuid.UUID `json:"id"`
	IsDirected bool      `json:"is_directed"`
}

type RenameGraphRequest struct {
	NewName string `json:"new_name" binding:"required"`
}
type RenameGraphResponse struct {
	ID      uuid.UUID `json:"id"`
	NewName string    `json:"new_name"`
}

type DeleteGraphResponse struct {
	ID uuid.UUID `json:"id"`
}
