package repository

import (
	"context"

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
func (nr *NodeRepository) GetAllNodesByGraph(ctx context.Context, graph uuid.UUID) ([]domain.Node, error) {
	nodes := make([]domain.Node, 0)
	err := nr.db.WithContext(ctx).Model(new(domain.Node)).Where("graph_id = ?", graph).Find(&nodes).Error
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

type EdgeRepository struct {
	*BaseRepository[domain.Edge, uuid.UUID]
}

func NewEdgeRepository(db *gorm.DB) *EdgeRepository {
	return &EdgeRepository{
		BaseRepository: NewBaseRepository[domain.Edge, uuid.UUID](db),
	}
}

func (er *EdgeRepository) GetAllEdgesByGraph(ctx context.Context, graph uuid.UUID, sort bool) ([]domain.Edge, error) {
	edges := make([]domain.Edge, 0)
	query := er.db.WithContext(ctx).Model(new(domain.Edge)).Where("graph_id = ?", graph)
	if sort {
		query = query.Order("weight")
	}
	err := query.Find(&edges).Error
	if err != nil {
		return nil, err
	}
	return edges, nil
}

type MaskRepository struct {
	*BaseRepository[domain.Mask, uuid.UUID]
}

func NewMaskRepository(db *gorm.DB) *MaskRepository {
	return &MaskRepository{
		BaseRepository: NewBaseRepository[domain.Mask, uuid.UUID](db),
	}
}

func (mr *MaskRepository) GetAllMasksByGraph(ctx context.Context, graph uuid.UUID) ([]domain.Mask, error) {
	masks := make([]domain.Mask, 0)
	err := mr.db.WithContext(ctx).Model(new(domain.Mask)).Where("graph_id = ?", graph).Find(&masks).Error
	if err != nil {
		return nil, err
	}
	return masks, nil
}

type MaskMemberRepository struct {
	*BaseRepository[domain.MaskMember, uuid.UUID]
}

func NewMaskMembersRepository(db *gorm.DB) *MaskMemberRepository {
	return &MaskMemberRepository{
		BaseRepository: NewBaseRepository[domain.MaskMember, uuid.UUID](db),
	}
}
func (mmr *MaskMemberRepository) GetAllMembersByMask(ctx context.Context, mask uuid.UUID) ([]domain.MaskMember, error) {
	members := make([]domain.MaskMember, 0)
	err := mmr.db.WithContext(ctx).Model(new(domain.MaskMember)).Where("mask_id = ?", mask).Find(&members).Error
	if err != nil {
		return nil, err
	}
	return members, nil
}
