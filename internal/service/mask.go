package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Gospoduu/graphs-service/internal/domain"
)

type MaskService struct {
	*BaseService[domain.Mask, uuid.UUID]
	repo         domain.MaskRepository
	membersRepo  domain.MaskMembersRepository
	graphService *GraphService
}

func NewMaskService(
	repo domain.MaskRepository,
	membersRepo domain.MaskMembersRepository,
	graphService *GraphService,
) *MaskService {
	return &MaskService{
		BaseService:  NewBaseService[domain.Mask, uuid.UUID](repo),
		repo:         repo,
		membersRepo:  membersRepo,
		graphService: graphService,
	}
}

func (ms *MaskService) GetAllMasksByGraph(ctx context.Context, graph uuid.UUID) ([]domain.Mask, error) {
	return ms.repo.GetAllMasksByGraph(ctx, graph)
}

func (ms *MaskService) GetAllMembersByMask(ctx context.Context, mask uuid.UUID) ([]domain.MaskMember, error) {
	return ms.membersRepo.GetAllMembersByMask(ctx, mask)
}

func (ms *MaskService) createMask(
	ctx context.Context,
	graph uuid.UUID,
	name string,
	edges []uuid.UUID,
) (domain.Mask, []uuid.UUID, error) {
	mask, err := ms.repo.Create(ctx, domain.Mask{
		GraphID: graph,
		Name:    name,
	})
	zero := domain.Mask{}
	if err != nil {
		return zero, nil, err
	}

	members := make([]uuid.UUID, 0, len(edges))
	for _, edgeID := range edges {
		member, err := ms.membersRepo.Create(ctx, domain.MaskMember{
			MaskID: mask.ID,
			EdgeID: edgeID,
		})
		if err != nil {
			return zero, nil, err
		}
		members = append(members, member.EdgeID)
	}

	return mask, members, nil
}

func (ms *MaskService) CreateDFSMask(ctx context.Context, graph uuid.UUID, startNode uuid.UUID) (domain.Mask, []uuid.UUID, error) {
	_, edges, is_directed, err := ms.graphService.GetDFS(ctx, graph, startNode)
	zero := domain.Mask{}
	if err != nil {
		return zero, nil, err
	}
	node, err := ms.graphService.nodeService.GetByID(ctx, startNode)
	if err != nil {
		return zero, nil, err
	}
	name := fmt.Sprintf("DFS Mask; Start Node: %d; Is Directed: %t", node.Name, is_directed)

	return ms.createMask(ctx, graph, name, edges)
}

func (ms *MaskService) CreateBFSMask(ctx context.Context, graph uuid.UUID, startNode uuid.UUID) (domain.Mask, []uuid.UUID, error) {
	_, edges, is_directed, err := ms.graphService.GetBFS(ctx, graph, startNode)
	zero := domain.Mask{}
	if err != nil {
		return zero, nil, err
	}
	node, err := ms.graphService.nodeService.GetByID(ctx, startNode)
	if err != nil {
		return zero, nil, err
	}
	name := fmt.Sprintf("BFS Mask; Start Node: %d; Is Directed: %t", node.Name, is_directed)

	return ms.createMask(ctx, graph, name, edges)
}

func (ms *MaskService) CreateMSTMask(ctx context.Context, graph uuid.UUID) (domain.Mask, []uuid.UUID, error) {
	_, edges, err := ms.graphService.GetKruskalMST(ctx, graph)
	zero := domain.Mask{}
	if err != nil {
		return zero, nil, err
	}
	name := fmt.Sprintf("MST Mask; Start Node")

	return ms.createMask(ctx, graph, name, edges)
}
