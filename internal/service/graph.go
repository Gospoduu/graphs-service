package service

import (
	"context"
	"errors"
	"sort"

	"github.com/google/uuid"

	"github.com/Gospoduu/graphs-service/internal/domain"
)

var OnlyForNoDirectedGraphError = errors.New("This action only for not oriented graph")
var NonConnectedGraphError = errors.New("This action only for connected graph")

type AdjacencyList map[uuid.UUID][]uuid.UUID

func toAdjacencyList(
	nodes []domain.Node, edges []domain.Edge, is_directed bool,
) (AdjacencyList, []domain.Edge, bool, error) {
	al := make(AdjacencyList, len(nodes))
	for _, edge := range edges {
		al[edge.SourceID] = append(al[edge.SourceID], edge.TargetID)
		if !is_directed {
			al[edge.TargetID] = append(al[edge.TargetID], edge.SourceID)
		} else {
			_, ok := al[edge.TargetID]
			if !ok {
				al[edge.TargetID] = []uuid.UUID{}
			}
		}
	}
	for _, node := range nodes {
		_, ok := al[node.ID]
		if !ok {
			return nil, nil, false, NonConnectedGraphError
		}
	}
	return al, edges, is_directed, nil
}

func toMapEdges(edges []domain.Edge) map[string]uuid.UUID {
	edgeMap := make(map[string]uuid.UUID)
	for _, edge := range edges {
		edgeMap[edge.SourceID.String()+":"+edge.TargetID.String()] = edge.ID
	}
	return edgeMap
}

type GraphService struct {
	*BaseService[domain.Graph, uuid.UUID]
	nodeService *NodeService
	edgeService *EdgeService
	repo        domain.GraphRepository
}

func NewGraphService(repo domain.GraphRepository, nodeService *NodeService, edgeService *EdgeService) *GraphService {
	return &GraphService{
		BaseService: NewBaseService[domain.Graph, uuid.UUID](repo),
		repo:        repo,
		nodeService: nodeService,
		edgeService: edgeService,
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

func (gs *GraphService) GetAllGraphsByUser(ctx context.Context, user uuid.UUID) ([]domain.Graph, error) {
	return gs.repo.GetByFilter(ctx, map[string]any{"user_id": user})
}

func (gs *GraphService) getAdjacencyList(ctx context.Context, graph uuid.UUID, sort bool) (AdjacencyList, []domain.Edge, bool, error) {
	graphData, err := gs.GetByID(ctx, graph)
	if err != nil {
		return nil, nil, false, err
	}
	chErr := make(chan error, 1)

	var nodes []domain.Node
	var edges []domain.Edge

	go func() {
		gettedNodes, nodeErr := gs.nodeService.GetAllNodesByGraph(ctx, graph)
		if gettedNodes != nil {
			nodes = gettedNodes
		}
		chErr <- nodeErr

	}()
	go func() {
		gettedEdges, edgeErr := gs.edgeService.GetAllEdgesByGraph(ctx, graph, sort)
		if gettedEdges != nil {
			edges = gettedEdges
		}
		chErr <- edgeErr

	}()
	if err := <-chErr; err != nil {
		<-chErr
		return nil, nil, false, err
	}
	if err := <-chErr; err != nil {
		return nil, nil, false, err
	}
	if nodes == nil || edges == nil {
		return nil, nil, false, errors.New("node or edge service returned a nil pointer")
	}
	return toAdjacencyList(nodes, edges, graphData.IsDirected)
}

func (gs *GraphService) GetDFS(ctx context.Context, graph uuid.UUID, startNode uuid.UUID) (AdjacencyList, []uuid.UUID, bool, error) {

	al, edges, is_directed, err := gs.getAdjacencyList(ctx, graph, false)
	if err != nil {
		return nil, nil, false, err
	}
	mapEdges := toMapEdges(edges)
	treeEdges := make([]uuid.UUID, 0, len(edges))
	if _, ok := al[startNode]; !ok {
		return nil, nil, false, errors.New("start node does not belong to graph")
	}
	var visited = make(map[uuid.UUID]struct{}, len(al))
	var DFSTree = make(AdjacencyList, len(al))
	var f func(curNode uuid.UUID)
	f = func(curNode uuid.UUID) {
		visited[curNode] = struct{}{}
		for _, i := range al[curNode] {
			_, ok := visited[i]
			if ok {
				continue
			}
			DFSTree[curNode] = append(DFSTree[curNode], i)
			edge, ok := mapEdges[curNode.String()+":"+i.String()]
			if !ok {
				edge = mapEdges[i.String()+":"+curNode.String()]
			}
			treeEdges = append(treeEdges, edge)
			f(i)
		}
	}
	f(startNode)
	return DFSTree, treeEdges, is_directed, nil
}

func (gs *GraphService) GetBFS(ctx context.Context, graph uuid.UUID, startNode uuid.UUID) (AdjacencyList, []uuid.UUID, bool, error) {
	al, edges, is_directed, err := gs.getAdjacencyList(ctx, graph, false)
	if err != nil {
		return nil, nil, false, err
	}
	mapEdges := toMapEdges(edges)
	treeEdges := make([]uuid.UUID, 0, len(edges))
	if _, ok := al[startNode]; !ok {
		return nil, nil, false, errors.New("start node does not belong to graph")
	}
	var visited = make(map[uuid.UUID]struct{}, len(al))
	visited[startNode] = struct{}{}
	queue := []uuid.UUID{startNode}
	var BFSTree = make(AdjacencyList, len(al))
	for len(queue) > 0 {
		curNode := queue[0]
		queue = queue[1:]
		for _, i := range al[curNode] {
			_, ok := visited[i]
			if ok {
				continue
			}
			visited[i] = struct{}{}
			BFSTree[curNode] = append(BFSTree[curNode], i)
			edge, ok := mapEdges[curNode.String()+":"+i.String()]
			if !ok {
				edge = mapEdges[i.String()+":"+curNode.String()]
			}
			treeEdges = append(treeEdges, edge)

			queue = append(queue, i)
		}
	}
	return BFSTree, treeEdges, is_directed, nil
}

func (gs *GraphService) GetKruskalMST(ctx context.Context, graph uuid.UUID) (AdjacencyList, []uuid.UUID, error) {
	graphData, err := gs.GetByID(ctx, graph)
	if err != nil {
		return nil, nil, err
	}
	if graphData.IsDirected {
		return nil, nil, OnlyForNoDirectedGraphError
	}
	edges, err := gs.edgeService.GetAllEdgesByGraph(ctx, graph, true)
	if err != nil {
		return nil, nil, err
	}
	nodes, err := gs.nodeService.GetAllNodesByGraph(ctx, graph)
	if err != nil {
		return nil, nil, err
	}
	if len(nodes) == 1 {
		return AdjacencyList{
			nodes[0].ID: []uuid.UUID{},
		}, []uuid.UUID{}, nil
	}
	var MSTree = make(AdjacencyList, len(nodes))
	components := make(map[uuid.UUID]*int, 0)
	lastComponentID := 0
	visited := make(map[uuid.UUID]struct{}, len(edges))
	treeEdges := make([]uuid.UUID, 0, len(edges))
	for _, edge := range edges {
		_, ok1 := MSTree[edge.SourceID]
		_, ok2 := MSTree[edge.TargetID]

		if ok1 && ok2 {
			continue
		} else if ok1 {
			components[edge.TargetID] = components[edge.SourceID]
		} else if ok2 {
			components[edge.SourceID] = components[edge.TargetID]
		} else {
			copyLastComponentID := lastComponentID
			components[edge.SourceID] = &copyLastComponentID
			components[edge.TargetID] = &copyLastComponentID
			lastComponentID++
		}
		MSTree[edge.SourceID] = append(MSTree[edge.SourceID], edge.TargetID)
		MSTree[edge.TargetID] = append(MSTree[edge.TargetID], edge.SourceID)
		treeEdges = append(treeEdges, edge.ID)

		visited[edge.ID] = struct{}{}
	}
	if len(components) != len(nodes) {
		return nil, nil, NonConnectedGraphError
	}
	priorityComponents := make(map[int][]*int, 0)
	for _, edge := range edges {
		if _, ok := visited[edge.ID]; ok {
			continue
		}
		targetLink := components[edge.TargetID]
		sourceLink := components[edge.SourceID]
		targetValue := *targetLink
		sourceValue := *sourceLink

		if targetValue == sourceValue {
			continue
		}

		_, ok1 := priorityComponents[sourceValue]
		_, ok2 := priorityComponents[targetValue]
		if !ok1 && !ok2 {
			priorityComponents[sourceValue] = []*int{sourceLink, targetLink}
			*targetLink = sourceValue
			MSTree[edge.SourceID] = append(MSTree[edge.SourceID], edge.TargetID)
			MSTree[edge.TargetID] = append(MSTree[edge.TargetID], edge.SourceID)
			treeEdges = append(treeEdges, edge.ID)
			continue

		} else if ok1 && !ok2 {
			priorityComponents[sourceValue] = append(priorityComponents[sourceValue], targetLink)
			*targetLink = sourceValue
			MSTree[edge.SourceID] = append(MSTree[edge.SourceID], edge.TargetID)
			MSTree[edge.TargetID] = append(MSTree[edge.TargetID], edge.SourceID)
			treeEdges = append(treeEdges, edge.ID)
			continue

		} else if !ok1 && ok2 {
			priorityComponents[targetValue] = append(priorityComponents[targetValue], sourceLink)
			*sourceLink = targetValue
			MSTree[edge.SourceID] = append(MSTree[edge.SourceID], edge.TargetID)
			MSTree[edge.TargetID] = append(MSTree[edge.TargetID], edge.SourceID)
			treeEdges = append(treeEdges, edge.ID)
			continue
		}

		if sourcePriority, targetPriority := len(priorityComponents[sourceValue]), len(priorityComponents[targetValue]); sourcePriority >= targetPriority {
			priorityComponents[sourceValue] = append(priorityComponents[sourceValue], priorityComponents[targetValue]...)
			for _, link := range priorityComponents[targetValue] {
				*link = sourceValue
			}
			delete(priorityComponents, targetValue)
			MSTree[edge.SourceID] = append(MSTree[edge.SourceID], edge.TargetID)
			MSTree[edge.TargetID] = append(MSTree[edge.TargetID], edge.SourceID)
		} else {
			priorityComponents[targetValue] = append(priorityComponents[targetValue], priorityComponents[sourceValue]...)
			for _, link := range priorityComponents[sourceValue] {
				*link = targetValue
			}
			delete(priorityComponents, sourceValue)
			MSTree[edge.SourceID] = append(MSTree[edge.SourceID], edge.TargetID)
			MSTree[edge.TargetID] = append(MSTree[edge.TargetID], edge.SourceID)
		}
		treeEdges = append(treeEdges, edge.ID)
	}
	if lastComponentID > 1 {
		if len(priorityComponents) != 1 {
			return nil, nil, NonConnectedGraphError
		}

		for _, links := range priorityComponents {
			if len(links) != lastComponentID {
				return nil, nil, NonConnectedGraphError
			}
		}
	}
	return MSTree, treeEdges, nil
}

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

func (gs *GraphService) GetAdjacencyMatrix(
	ctx context.Context,
	graph uuid.UUID,
) ([]AdjacencyMatrixRow, error) {
	graphData, err := gs.GetByID(ctx, graph)
	if err != nil {
		return nil, err
	}

	nodes, err := gs.nodeService.GetAllNodesByGraph(ctx, graph)
	if err != nil {
		return nil, err
	}

	edges, err := gs.edgeService.GetAllEdgesByGraph(ctx, graph, false)
	if err != nil {
		return nil, err
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Name < nodes[j].Name
	})

	matrix := make([]AdjacencyMatrixRow, len(nodes))
	indices := make(map[uuid.UUID]int, len(nodes))

	// Строки и столбцы используют один порядок вершин.
	for i, node := range nodes {
		indices[node.ID] = i

		matrix[i] = AdjacencyMatrixRow{
			NodeID: node.ID,
			Name:   node.Name,
			Cells:  make([]AdjacencyMatrixCell, len(nodes)),
		}

		for j, target := range nodes {
			matrix[i].Cells[j] = AdjacencyMatrixCell{
				NodeID: target.ID,
				Name:   target.Name,
				// Weight: 0, HasEdge: false — нулевые значения.
			}
		}
	}

	for _, edge := range edges {
		source, sourceOK := indices[edge.SourceID]
		target, targetOK := indices[edge.TargetID]
		if !sourceOK || !targetOK {
			return nil, errors.New("edge endpoint does not belong to graph")
		}

		matrix[source].Cells[target].Weight = float64(edge.Weight)
		matrix[source].Cells[target].HasEdge = true

		if !graphData.IsDirected {
			matrix[target].Cells[source].Weight = float64(edge.Weight)
			matrix[target].Cells[source].HasEdge = true
		}
	}

	return matrix, nil
}
