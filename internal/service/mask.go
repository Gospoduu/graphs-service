package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/datatypes"

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

func (ms *MaskService) createShortestPathMask(
	ctx context.Context,
	graph, startNode uuid.UUID,
	name string,
	edges []uuid.UUID,
) (domain.Mask, []uuid.UUID, error) {
	zero := domain.Mask{}
	if len(edges) == 0 {
		return zero, nil, errors.New("path is empty: start and end coincide")
	}
	maskMeta, err := json.Marshal(struct {
		IsPath      bool      `json:"is_path"`
		StartNodeID uuid.UUID `json:"start_node_id"`
	}{true, startNode})
	if err != nil {
		return zero, nil, err
	}
	mask, err := ms.repo.Create(ctx, domain.Mask{
		GraphID:  graph,
		Name:     name,
		Metadata: datatypes.JSON(maskMeta),
	})
	if err != nil {
		return zero, nil, err
	}
	for serial, edgeID := range edges {
		metadata, err := json.Marshal(struct {
			Serial int `json:"serial"`
		}{Serial: serial})
		if err != nil {
			_ = ms.repo.DeleteByID(ctx, mask.ID)
			return zero, nil, err
		}
		_, err = ms.membersRepo.Create(ctx, domain.MaskMember{
			MaskID:   mask.ID,
			EdgeID:   edgeID,
			Metadata: datatypes.JSON(metadata),
		})
		if err != nil {
			_ = ms.repo.DeleteByID(ctx, mask.ID)
			return zero, nil, err
		}
	}
	return mask, edges, nil
}

func (ms *MaskService) CreateDijkstraPathMask(ctx context.Context, graph, from, to uuid.UUID) (domain.Mask, []uuid.UUID, error) {
	edges, err := ms.graphService.DijkstraPathfinding(ctx, graph, from, to)
	zero := domain.Mask{}
	if err != nil {
		return zero, nil, err
	}
	name := fmt.Sprintf("Dijkstra Path Mask; From: %s; To: %s", from.String(), to.String())
	return ms.createShortestPathMask(ctx, graph, from, name, edges)
}

func (ms *MaskService) CreateFordBellmanPathMask(ctx context.Context, graph, from, to uuid.UUID) (domain.Mask, []uuid.UUID, error) {
	edges, err := ms.graphService.FordBellmanPathfinding(ctx, graph, from, to)
	zero := domain.Mask{}
	if err != nil {
		return zero, nil, err
	}
	name := fmt.Sprintf("Ford-Bellman Path Mask; From: %s; To: %s", from.String(), to.String())
	return ms.createShortestPathMask(ctx, graph, from, name, edges)
}
