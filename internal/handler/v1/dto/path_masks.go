package dto

import (
	"encoding/json"

	"github.com/google/uuid"
)

type CreateEulerianMasksRequest struct {
	GraphID uuid.UUID `json:"graph_id" binding:"required"`
}
type CreateHamiltonianMasksRequest struct {
	GraphID uuid.UUID `json:"graph_id" binding:"required"`
}
type CreateDijkstraMasksRequest struct {
	GraphID uuid.UUID `json:"graph_id" binding:"required"`
	From    uuid.UUID `json:"from" binding:"required"`
	To      uuid.UUID `json:"to" binding:"required"`
}
type CreateFordBellmanMasksRequest struct {
	GraphID uuid.UUID `json:"graph_id" binding:"required"`
	From    uuid.UUID `json:"from" binding:"required"`
	To      uuid.UUID `json:"to" binding:"required"`
}

// Предложение ещё не сохранено в БД, поэтому у него нет ID.
type SuggestedEdgeResponse struct {
	GraphID  uuid.UUID `json:"graph_id"`
	SourceID uuid.UUID `json:"source_id"`
	TargetID uuid.UUID `json:"target_id"`
	Weight   float64   `json:"weight"`
}

type EulerianAnalysisResponse struct {
	// 0 — эйлеровый, 1 — полуэйлеровый, 2 — неэйлеровый.
	Kind          int                     `json:"kind"`
	StartNodeID   uuid.UUID               `json:"start_node_id"`
	PathEdges     []SuggestedEdgeResponse `json:"path_edges"`
	CycleEdges    []SuggestedEdgeResponse `json:"cycle_edges"`
	NeedMoreNodes bool                    `json:"need_more_nodes"`
}

type HamiltonianAnalysisResponse struct {
	Kind        int                     `json:"kind"`
	StartNodeID uuid.UUID               `json:"start_node_id"`
	PathEdges   []SuggestedEdgeResponse `json:"path_edges"`
	CycleEdges  []SuggestedEdgeResponse `json:"cycle_edges"`
}

type PathFolderResponse struct {
	ID       uuid.UUID       `json:"id"`
	GraphID  uuid.UUID       `json:"graph_id"`
	Name     string          `json:"name"`
	Metadata json.RawMessage `json:"metadata"`
}

type CreatePathMasksResponse[Analysis EulerianAnalysisResponse | HamiltonianAnalysisResponse] struct {
	Folder       *PathFolderResponse `json:"folder"`
	MasksCreated int                 `json:"masks_created"`
	Analysis     Analysis            `json:"analysis"`
	Error        string              `json:"error,omitempty"`
}
