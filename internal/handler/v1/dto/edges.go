package dto

import (
	"encoding/json"

	"github.com/google/uuid"
)

type EdgeResponse struct {
	ID       uuid.UUID       `json:"id"`
	GraphID  uuid.UUID       `json:"graph_id"`
	SourceID uuid.UUID       `json:"source_id"`
	TargetID uuid.UUID       `json:"target_id"`
	Metadata json.RawMessage `json:"metadata"`
	Weight   float64         `json:"weight"`
}

type CreateEdgeRequest struct {
	GraphID  uuid.UUID       `json:"graph_id" binding:"required"`
	SourceID uuid.UUID       `json:"source_id" binding:"required"`
	TargetID uuid.UUID       `json:"target_id" binding:"required"`
	Metadata json.RawMessage `json:"metadata"`
	Weight   float64         `json:"weight"`
}

type ToggleDirectionResponse struct {
	ID       uuid.UUID `json:"id"`
	SourceID uuid.UUID `json:"source_id"`
	TargetID uuid.UUID `json:"target_id"`
}

type ChangeWeightRequest struct {
	Weight float64 `json:"weight"`
}
type ChangeWeightResponse struct {
	ID uuid.UUID `json:"id"`
}

type DeleteEdgeResponse struct {
	ID uuid.UUID `json:"id"`
}
