package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/Gospoduu/graphs-service/internal/domain"
)

type NodeService struct {
	*BaseService[domain.Node, uuid.UUID]
	repo domain.NodeRepository
}

func NewNodeService(
	repo domain.NodeRepository,
) *NodeService {
	return &NodeService{
		BaseService: NewBaseService[domain.Node, uuid.UUID](repo),
		repo:        repo,
	}
}

func (ns *NodeService) GetAllNodesByGraph(ctx context.Context, graph uuid.UUID) ([]domain.Node, error) {
	return ns.repo.GetAllNodesByGraph(ctx, graph)
}
