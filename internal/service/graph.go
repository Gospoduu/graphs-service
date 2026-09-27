package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/Gospoduu/graphs-service/internal/domain"
)

type GraphService struct {
	*BaseService[domain.Graph, uuid.UUID]
}

func NewGraphService(repo domain.Repository[domain.Graph, uuid.UUID]) *GraphService {
	return &GraphService{
		BaseService: NewBaseService[domain.Graph, uuid.UUID](repo),
	}
}

func (gs *GraphService) ToggleIsDirected(ctx context.Context, id uuid.UUID) (bool, error) {
	curIsDirected, err := gs.GetByID(ctx, id)
	if err != nil {
		return false, err
	}
	newIsDirected := !curIsDirected.IsDirected
	err = gs.PatchByID(ctx, id, map[string]any{"is_directed": newIsDirected})
	if err != nil {
		return false, err
	}
	return newIsDirected, nil
}
