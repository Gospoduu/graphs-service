package domain

import (
	"context"

	"github.com/google/uuid"
)

type Service[T any, ID IDConstraint] interface {
	GetByID(ctx context.Context, id ID) (*T, error)
	Create(ctx context.Context, entity T) (*T, error)
	DeleteByID(ctx context.Context, id ID) error
	UpdateByID(ctx context.Context, id ID, entity T) error
	PatchByID(ctx context.Context, id ID, fields map[string]any) error
}

type AdjacencyList map[uuid.UUID][]uuid.UUID

type AdjacencyMatrixCell struct {
	NodeID  uuid.UUID `json:"node_id"`
	Name    int       `json:"name"`
	Weight  float64   `json:"weight"`
	HasEdge bool      `json:"has_edge"`
}
type AdjacencyMatrixRow struct {
	NodeID uuid.UUID             `json:"node_id"`
	Name   int                   `json:"name"`
	Cells  []AdjacencyMatrixCell `json:"cells"`
}
type GraphService interface {
	Service[Graph, uuid.UUID]

	GetAllGraphsByUser(
		ctx context.Context,
		user uuid.UUID,
	) ([]Graph, error)

	GetBFS(ctx context.Context, graph uuid.UUID, startNode uuid.UUID) (AdjacencyList, []uuid.UUID, error)
	GetKruskalMST(ctx context.Context, graph uuid.UUID) (AdjacencyList, []uuid.UUID, error)
	GetDFS(ctx context.Context, graph uuid.UUID, startNode uuid.UUID) (AdjacencyList, []uuid.UUID, error)
	GetAdjacencyMatrix(ctx context.Context, graph uuid.UUID) ([]AdjacencyMatrixRow, error)
}

type NodeService interface {
	Service[Node, uuid.UUID]

	GetAllNodesByGraph(
		ctx context.Context,
		graph uuid.UUID,
	) ([]Node, error)
}
type EdgeService interface {
	Service[Edge, uuid.UUID]

	GetAllEdgesByGraph(
		ctx context.Context,
		graph uuid.UUID,
		sort bool,
	) ([]Edge, error)
}
