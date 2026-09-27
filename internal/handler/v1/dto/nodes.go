package dto

import (
	"encoding/json"

	"github.com/google/uuid"
)

type NodeResponse struct {
	ID       uuid.UUID       `json:"id"`
	GraphID  uuid.UUID       `json:"graph_id"`
	Metadata json.RawMessage `json:"meta_data"`
	X        float32         `json:"x"`
	Y        float32         `json:"y"`
}

type CreateNodeRequest struct {
	GraphID  uuid.UUID       `json:"graph_id" binding:"required"`
	Metadata json.RawMessage `json:"meta_data"`
	X        float32         `json:"x"`
	Y        float32         `json:"y"`
}

type ChangePositionRequest struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}
type ChangePositionResponse struct {
	ID uuid.UUID `json:"id"`
}

type DeleteNodeResponse struct {
	ID uuid.UUID `json:"id"`
}
