package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/Gospoduu/graphs-service/internal/domain"
)

type EdgeService struct {
	*BaseService[domain.Edge, uuid.UUID]
	repo domain.EdgeRepository
}

func NewEdgeService(
	repo domain.EdgeRepository,
) *EdgeService {
	return &EdgeService{
		BaseService: NewBaseService[domain.Edge, uuid.UUID](repo),
		repo:        repo,
	}
}

func (es *EdgeService) ToggleDirect(ctx context.Context, id uuid.UUID) (uuid.UUID, uuid.UUID, error) {
	curDirect, err := es.GetByID(ctx, id)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	newSource, newTarget := curDirect.TargetID, curDirect.SourceID
	err = es.PatchByID(ctx, id, map[string]any{"target_id": newTarget, "source_id": newSource})
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return newSource, newTarget, nil
}
func (es *EdgeService) GetAllEdgesByGraph(ctx context.Context, graph uuid.UUID, sort bool) ([]domain.Edge, error) {
	return es.repo.GetAllEdgesByGraph(ctx, graph, sort)
}
