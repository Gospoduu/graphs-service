package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Gospoduu/graphs-service/internal/domain"
)

type GraphRepository struct {
	*BaseRepository[domain.Graph, uuid.UUID]
}

func NewGraphRepository(db *gorm.DB) *GraphRepository {
	return &GraphRepository{
		BaseRepository: NewBaseRepository[domain.Graph, uuid.UUID](db),
	}
}

type NodeRepository struct {
	*BaseRepository[domain.Node, uuid.UUID]
}

func NewNodeRepository(db *gorm.DB) *NodeRepository {
	return &NodeRepository{
		BaseRepository: NewBaseRepository[domain.Node, uuid.UUID](db),
	}
}

type EdgeRepository struct {
	*BaseRepository[domain.Edge, uuid.UUID]
}

func NewEdgeRepository(db *gorm.DB) *EdgeRepository {
	return &EdgeRepository{
		BaseRepository: NewBaseRepository[domain.Edge, uuid.UUID](db),
	}
}
