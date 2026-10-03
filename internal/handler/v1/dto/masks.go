package dto

import (
	"github.com/google/uuid"
)

type MaskResponse struct {
	ID      uuid.UUID   `json:"id"`
	GraphID uuid.UUID   `json:"graph_id"`
	Name    string      `json:"name"`
	Members []uuid.UUID `json:"members"`
}

type CreateMaskRequest struct {
	GraphID uuid.UUID `json:"graph_id" binding:"required"`
	StartID uuid.UUID `json:"start_id" binding:"required"`
}
type CreateMSTMaskRequest struct {
	GraphID uuid.UUID `json:"graph_id" binding:"required"`
}

type DeleteMaskResponse struct {
	ID uuid.UUID `json:"id"`
}
