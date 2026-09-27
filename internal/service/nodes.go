package service

import (
	"github.com/Gospoduu/graphs-service/internal/domain"
	"github.com/google/uuid"
)

type NodeService struct {
	*BaseService[domain.Node, uuid.UUID]
}

func NewNodeService(repo domain.Repository[domain.Node, uuid.UUID]) *NodeService {
	return &NodeService{
		BaseService: NewBaseService[domain.Node, uuid.UUID](repo),
	}
}
