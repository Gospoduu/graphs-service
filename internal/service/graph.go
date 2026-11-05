package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Gospoduu/graphs-service/internal/domain"
)

var OnlyForNoDirectedGraphError = errors.New("This action only for not oriented graph")
var NonConnectedGraphError = errors.New("This action only for connected graph")

func toAdjacencyList(
	nodes []domain.Node, edges []domain.Edge, isDirected bool,
) (domain.AdjacencyList, []domain.Edge, bool, error) {
	al := make(domain.AdjacencyList, len(nodes))
	for _, edge := range edges {
		al[edge.SourceID] = append(al[edge.SourceID], edge.TargetID)
		if !isDirected {
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
	return al, edges, isDirected, nil
}

func isGraphConnected(nodes []domain.Node, edges []domain.Edge) bool {
	if len(nodes) == 0 {
		return false
	}

	// Каждая вершина сначала образует отдельную компоненту.
	positions := make(map[uuid.UUID]int, len(nodes))
	parent := make([]int, len(nodes))
	size := make([]int, len(nodes))
	for i, node := range nodes {
		if _, exists := positions[node.ID]; exists {
			return false
		}
		positions[node.ID] = i
		parent[i] = i
		size[i] = 1
	}
	components := len(nodes)

	// Находим представителя компоненты. Одновременно укорачиваем цепочку
	// родителей, чтобы следующие поиски были быстрее (сжатие путей).
	find := func(vertex int) int {
		for parent[vertex] != vertex {
			parent[vertex] = parent[parent[vertex]]
			vertex = parent[vertex]
		}
		return vertex
	}

	for _, edge := range edges {
		source, sourceExists := positions[edge.SourceID]
		target, targetExists := positions[edge.TargetID]
		if !sourceExists || !targetExists {
			return false
		}

		sourceRoot := find(source)
		targetRoot := find(target)
		if sourceRoot == targetRoot {
			// Ребро внутри одной компоненты связность не меняет.
			continue
		}

		// Присоединяем меньшую компоненту к большей.
		// Меняем только родителя корня, а не все вершины компоненты.
		if size[sourceRoot] < size[targetRoot] {
			sourceRoot, targetRoot = targetRoot, sourceRoot
		}
		parent[targetRoot] = sourceRoot
		size[sourceRoot] += size[targetRoot]
		components--
	}

	return components == 1
}

func getKeyForMapEdges(source uuid.UUID, target uuid.UUID) string {
	return source.String() + ":" + target.String()
}
func toMapEdges(edges []domain.Edge, isDirected bool) map[string]uuid.UUID {
	edgeMap := make(map[string]uuid.UUID)
	for _, edge := range edges {
		edgeMap[getKeyForMapEdges(edge.SourceID, edge.TargetID)] = edge.ID
		if !isDirected {
			edgeMap[getKeyForMapEdges(edge.TargetID, edge.SourceID)] = edge.ID
		}
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
func (gs *GraphService) getNodesAndEdges(
	ctx context.Context, graph uuid.UUID, sort bool,
) ([]domain.Node, []domain.Edge, bool, error) {
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
	return nodes, edges, graphData.IsDirected, nil
}

func (gs *GraphService) getAdjacencyList(
	ctx context.Context, graph uuid.UUID, sort bool,
) (domain.AdjacencyList, []domain.Edge, bool, error) {
	nodes, edges, isDirected, err := gs.getNodesAndEdges(ctx, graph, sort)
	if err != nil {
		return nil, nil, false, err
	}
	return toAdjacencyList(nodes, edges, isDirected)
}

func (gs *GraphService) GetDFS(ctx context.Context, graph uuid.UUID, startNode uuid.UUID) (domain.AdjacencyList, []uuid.UUID, bool, error) {

	al, edges, isDirected, err := gs.getAdjacencyList(ctx, graph, false)
	if err != nil {
		return nil, nil, false, err
	}
	mapEdges := toMapEdges(edges, isDirected)
	treeEdges := make([]uuid.UUID, 0, len(edges))
	if _, ok := al[startNode]; !ok {
		return nil, nil, false, errors.New("start node does not belong to graph")
	}
	var visited = make(map[uuid.UUID]struct{}, len(al))
	var DFSTree = make(domain.AdjacencyList, len(al))
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
	return DFSTree, treeEdges, isDirected, nil
}

func (gs *GraphService) GetBFS(ctx context.Context, graph uuid.UUID, startNode uuid.UUID) (domain.AdjacencyList, []uuid.UUID, bool, error) {
	al, edges, isDirected, err := gs.getAdjacencyList(ctx, graph, false)
	if err != nil {
		return nil, nil, false, err
	}
	mapEdges := toMapEdges(edges, isDirected)
	treeEdges := make([]uuid.UUID, 0, len(edges))
	if _, ok := al[startNode]; !ok {
		return nil, nil, false, errors.New("start node does not belong to graph")
	}
	var visited = make(map[uuid.UUID]struct{}, len(al))
	visited[startNode] = struct{}{}
	queue := []uuid.UUID{startNode}
	var BFSTree = make(domain.AdjacencyList, len(al))
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
	return BFSTree, treeEdges, isDirected, nil
}

func (gs *GraphService) GetKruskalMST(ctx context.Context, graph uuid.UUID) (domain.AdjacencyList, []uuid.UUID, error) {
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
		return domain.AdjacencyList{
			nodes[0].ID: []uuid.UUID{},
		}, []uuid.UUID{}, nil
	}
	var MSTree = make(domain.AdjacencyList, len(nodes))
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

const (
	isEulirian int = iota
	isHalfEulirian
	isNotEulirian
)

func (gs *GraphService) checkEulirian(edges []domain.Edge, isDirected bool) (int, uuid.UUID) {
	sourceCnt := make(map[uuid.UUID]int)
	targetCnt := make(map[uuid.UUID]int)

	if len(edges) == 0 {
		return isEulirian, uuid.Nil
	}

	for _, edge := range edges {

		_, ok := targetCnt[edge.SourceID]
		if !ok {
			targetCnt[edge.SourceID] = 0
		}
		_, ok = sourceCnt[edge.TargetID]
		if !ok {
			sourceCnt[edge.TargetID] = 0
		}

		sourceCnt[edge.SourceID] += 1
		targetCnt[edge.TargetID] += 1
	}
	cnt := 0
	start := uuid.Nil
	if isDirected {
		p := false
		n := false
		for key, source := range sourceCnt {
			target := targetCnt[key]
			d := source - target
			if d == 0 {
				continue
			} else if d == 1 && p {
				return isNotEulirian, uuid.Nil
			} else if d == 1 {
				start = key
				p = true
			} else if d == -1 && n {
				return isNotEulirian, uuid.Nil
			} else if d == -1 {
				n = true
			} else {
				return isNotEulirian, uuid.Nil
			}
		}
		if p && n {
			return isHalfEulirian, start
		} else if !p && !n {
			return isEulirian, edges[0].SourceID
		}
		return isNotEulirian, uuid.Nil
	}
	for vertex, source := range sourceCnt {
		if (source+targetCnt[vertex])%2 != 0 {
			cnt++
			start = vertex

			if cnt > 2 {
				return isNotEulirian, uuid.Nil
			}
		}
	}
	switch cnt {
	case 0:
		return isEulirian, edges[0].SourceID
	case 2:
		return isHalfEulirian, start
	default:
		return isNotEulirian, uuid.Nil
	}
}

var ErrNotEulerian = errors.New("graph has no Eulerian trail")
var ErrInvalidEulerianStart = errors.New("invalid start node for an Eulerian trail")
var ErrNilChan = errors.New("Path-channel is nil")

func (gs *GraphService) findAllEulirianPathes(
	ctx context.Context,
	graph uuid.UUID,
	startNode uuid.UUID,
	out chan<- []uuid.UUID,
	limit int,
) error {

	if out == nil {
		return ErrNilChan
	}
	defer close(out)

	searchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	nodes, edges, isDirected, err := gs.getNodesAndEdges(searchCtx, graph, false)
	if err != nil {
		return err
	}

	startExists := false
	for _, n := range nodes {
		if n.ID == startNode {
			startExists = true
			break
		}
	}
	if !startExists {
		return ErrInvalidEulerianStart
	}

	type halfEdge struct {
		id uuid.UUID
		to uuid.UUID
	}
	adj := make(map[uuid.UUID][]halfEdge, len(nodes))
	for _, e := range edges {
		adj[e.SourceID] = append(adj[e.SourceID], halfEdge{e.ID, e.TargetID})
		if !isDirected {
			adj[e.TargetID] = append(adj[e.TargetID], halfEdge{e.ID, e.SourceID})
		}
	}

	used := make(map[uuid.UUID]struct{}, len(edges))
	path := make([]uuid.UUID, 0, len(edges))
	cnt := 0
	limitReached := false

	var f func(curNode uuid.UUID) error

	f = func(curNode uuid.UUID) error {
		if err := searchCtx.Err(); err != nil {
			if limitReached {
				return nil
			}
			return err
		}
		if len(path) == len(edges) {
			select {
			case out <- append([]uuid.UUID(nil), path...):
				cnt++
				if limit > 0 && cnt >= limit {
					limitReached = true
					cancel()
				}
				return nil
			case <-searchCtx.Done():
				if limitReached {
					return nil
				}
				return searchCtx.Err()
			}
		}
		for _, h := range adj[curNode] {
			if _, ok := used[h.id]; ok {
				continue
			}
			used[h.id] = struct{}{}
			path = append(path, h.id)
			if err := f(h.to); err != nil {
				return err
			}
			path = path[:len(path)-1]
			delete(used, h.id)
		}
		return nil
	}

	return f(startNode)
}

var ErrNeedMoreNodes = errors.New("Need more nodes")

func (gs *GraphService) suggestEulerianEdges(
	ctx context.Context,
	graphID uuid.UUID,
	edges []domain.Edge,
	isDirected bool,
) ([]domain.Edge, []domain.Edge, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	kind, start := gs.checkEulirian(edges, isDirected)
	if kind == isEulirian {
		return []domain.Edge{}, []domain.Edge{}, nil
	}

	// Записи A:B и B:A различаем даже для неориентированного графа.
	type pair struct {
		from uuid.UUID
		to   uuid.UUID
	}

	existing := make(map[pair]bool, len(edges))

	// Для неориентированного графа — степень.
	// Для ориентированного — исходящие минус входящие.
	degrees := make(map[uuid.UUID]int)
	vertices := make([]uuid.UUID, 0)

	for _, edge := range edges {
		for _, id := range [2]uuid.UUID{
			edge.SourceID,
			edge.TargetID,
		} {
			if _, exists := degrees[id]; !exists {
				degrees[id] = 0
				vertices = append(vertices, id)
			}
		}

		existing[pair{edge.SourceID, edge.TargetID}] = true

		degrees[edge.SourceID]++
		if isDirected {
			degrees[edge.TargetID]--
		} else {
			degrees[edge.TargetID]++
		}
	}

	cycleEdges := make([]domain.Edge, 0)

	// Добавляем предложение только в локальное состояние.
	// В БД здесь ничего не сохраняем.
	add := func(from, to uuid.UUID) {
		cycleEdges = append(cycleEdges, domain.Edge{
			GraphID:  graphID,
			SourceID: from,
			TargetID: to,
			Weight:   1,
		})

		existing[pair{from, to}] = true

		degrees[from]++
		if isDirected {
			degrees[to]--
		} else {
			degrees[to]++
		}
	}

	// Полуэйлеровый граф: по нашему контракту добавляем
	// только одно ребро из конца пути в начало.
	if kind == isHalfEulirian {
		var end uuid.UUID

		for _, vertex := range vertices {
			if isDirected {
				if degrees[vertex] == -1 {
					end = vertex
					break
				}
			} else if vertex != start && degrees[vertex]%2 != 0 {
				end = vertex
				break
			}
		}

		from, to := end, start

		if existing[pair{from, to}] {
			if isDirected || existing[pair{to, from}] {
				return nil, nil, ErrNeedMoreNodes
			}

			from, to = to, from
		}

		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}

		add(from, to)
		return []domain.Edge{}, cycleEdges, nil
	}

	balanced := func() bool {
		for _, degree := range degrees {
			if isDirected {
				if degree != 0 {
					return false
				}
			} else if degree%2 != 0 {
				return false
			}
		}
		return true
	}

	// Ищем ребро, которое исправляет степени обоих концов.
	findCandidate := func() (pair, bool, error) {
		for priority := 0; priority < 2; priority++ {
			for _, from := range vertices {
				for _, to := range vertices {
					if err := ctx.Err(); err != nil {
						return pair{}, false, err
					}

					if from == to || existing[pair{from, to}] {
						continue
					}

					hasReverse := existing[pair{to, from}]
					if priority == 0 && hasReverse {
						continue
					}
					if priority == 1 && !hasReverse {
						continue
					}

					if isDirected {
						if degrees[from] >= 0 || degrees[to] <= 0 {
							continue
						}
					} else {
						if degrees[from]%2 == 0 ||
							degrees[to]%2 == 0 {
							continue
						}
					}

					return pair{from, to}, true, nil
				}
			}
		}

		return pair{}, false, nil
	}

	for !balanced() {
		candidate, found, err := findCandidate()
		if err != nil {
			return nil, nil, err
		}
		if !found {
			break
		}

		add(candidate.from, candidate.to)
	}

	if !balanced() {
		allEdges := make([]domain.Edge, 0, len(edges)+len(cycleEdges))
		allEdges = append(allEdges, edges...)
		allEdges = append(allEdges, cycleEdges...)

		for _, edge := range allEdges {
			if err := ctx.Err(); err != nil {
				return nil, nil, err
			}

			from, to := edge.TargetID, edge.SourceID
			if !existing[pair{from, to}] {
				add(from, to)
			}
		}
	}

	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	pathEdges := make([]domain.Edge, len(cycleEdges)-1)
	copy(pathEdges, cycleEdges[:len(cycleEdges)-1])

	return pathEdges, cycleEdges, nil
}

type EulerianResult struct {
	Kind          int
	StartNodeID   uuid.UUID
	PathEdges     []domain.Edge // Добавления для открытого пути.
	CycleEdges    []domain.Edge // Добавления для цикла.
	NeedMoreNodes bool
}

func (gs *GraphService) RunEulerian(
	ctx context.Context,
	graphID uuid.UUID,
	out chan<- []uuid.UUID,
	limit int,
) (EulerianResult, error) {
	result := EulerianResult{
		Kind:       isNotEulirian,
		PathEdges:  []domain.Edge{},
		CycleEdges: []domain.Edge{},
	}

	if out == nil {
		return result, errors.New("output channel is nil")
	}
	closeHere := true
	defer func() {
		if closeHere {
			close(out)
		}
	}()

	if err := ctx.Err(); err != nil {
		return result, err
	}

	graph, err := gs.GetByID(ctx, graphID)
	if err != nil {
		return result, err
	}

	nodes, err := gs.nodeService.GetAllNodesByGraph(ctx, graphID)
	if err != nil {
		return result, err
	}

	edges, err := gs.edgeService.GetAllEdgesByGraph(
		ctx,
		graphID,
		false,
	)
	if err != nil {
		return result, err
	}

	if err := ctx.Err(); err != nil {
		return result, err
	}

	// Для ориентированного графа здесь проверяется слабая связность.
	// Вместе с условиями на степени её достаточно.
	if !isGraphConnected(nodes, edges) {
		return result, NonConnectedGraphError
	}

	result.Kind, result.StartNodeID = gs.checkEulirian(
		edges,
		graph.IsDirected,
	)

	if len(edges) == 0 {
		result.StartNodeID = nodes[0].ID

		select {
		case out <- []uuid.UUID{}:
			return result, ctx.Err()
		case <-ctx.Done():
			return result, ctx.Err()
		}
	}

	if result.Kind != isEulirian {
		pathEdges, cycleEdges, err := gs.suggestEulerianEdges(
			ctx,
			graphID,
			edges,
			graph.IsDirected,
		)

		if err != nil {
			if !errors.Is(err, ErrNeedMoreNodes) {
				return result, err
			}
			result.NeedMoreNodes = true
		} else {
			result.PathEdges = pathEdges
			result.CycleEdges = cycleEdges
		}
	}

	if result.Kind == isNotEulirian {
		return result, ctx.Err()
	}

	// Передаём ответственность за закрытие канала функции поиска.
	closeHere = false

	err = gs.findAllEulirianPathes(
		ctx,
		graphID,
		result.StartNodeID,
		out,
		limit,
	)
	return result, err
}

type GamiltonPath struct {
	Path    []uuid.UUID
	IsCycle bool
}

const (
	isHamiltonian int = iota
	isHalfHamiltonian
	isNotHamiltonian
	unknownHamiltonian
)

var ErrInvalidHamiltonianStart = errors.New("invalid start node for an Eulerian trail")

func (gs *GraphService) findAllHamiltonianPathes(
	ctx context.Context,
	al domain.AdjacencyList,
	edges []domain.Edge,
	isDirected bool,
	startNode uuid.UUID,
	out chan<- GamiltonPath,
	maxPathes int,
	timeout time.Duration,
) (int, error) {
	if out == nil {
		return unknownHamiltonian, ErrNilChan
	}

	var gCtx context.Context
	var cancel context.CancelFunc
	if timeout > 0 {
		gCtx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		gCtx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	if len(al) == 0 {
		return unknownHamiltonian, fmt.Errorf("Graph is empty")
	}

	mapEdges := toMapEdges(edges, isDirected)
	if _, ok := al[startNode]; !ok {
		return unknownHamiltonian, ErrInvalidHamiltonianStart
	}
	if len(al) == 1 {
		return isHamiltonian, nil
	}
	if maxPathes < 0 {
		return unknownHamiltonian, fmt.Errorf("maxPathes must be >= 0, got %d", maxPathes)
	}
	unlimited := maxPathes == 0
	counter := 0
	var visited = make(map[uuid.UUID]struct{}, len(al))
	var path = make([]uuid.UUID, 0, len(al)-1)
	kind := isNotHamiltonian
	var f func(curNode uuid.UUID) error
	f = func(curNode uuid.UUID) error {
		if gCtx.Err() != nil {
			return gCtx.Err()
		}
		if !unlimited && counter >= maxPathes {
			return nil
		}
		visited[curNode] = struct{}{}

		if len(path) == len(al)-1 {
			res := make([]uuid.UUID, len(path))
			copy(res, path)
			cycleKey := getKeyForMapEdges(curNode, startNode)
			isCycle := false
			if closingEdgeID, ok := mapEdges[cycleKey]; ok {
				res = append(res, closingEdgeID)
				isCycle = true
				kind = isHamiltonian
			} else if kind != isHamiltonian {
				kind = isHalfHamiltonian
			}
			select {
			case out <- GamiltonPath{Path: res, IsCycle: isCycle}:
			case <-gCtx.Done():
				return gCtx.Err()
			}
			counter++
			return nil
		}
		for _, i := range al[curNode] {
			if gCtx.Err() != nil {
				return gCtx.Err()
			}
			_, ok := visited[i]
			if ok {
				continue
			}
			key := getKeyForMapEdges(curNode, i)

			edge, ok := mapEdges[key]
			if !ok {
				return fmt.Errorf("edge not found: %s -> %s", curNode, i)
			}
			path = append(path, edge)
			err := f(i)
			if err != nil {
				return err
			}
			path = path[:len(path)-1]
			delete(visited, i)
		}
		return nil
	}
	return kind, f(startNode)
}

func findStartNodesForHamilton(al domain.AdjacencyList) ([]uuid.UUID, error) {
	if len(al) == 0 {
		return nil, errors.New("nil al")
	}
	nodes := make([]uuid.UUID, 0, len(al))
	for n := range al {
		nodes = append(nodes, n)
	}
	sort.Slice(nodes, func(i, j int) bool {
		li, lj := len(al[nodes[i]]), len(al[nodes[j]])
		if li != lj {
			return li < lj
		}
		return bytes.Compare(nodes[i][:], nodes[j][:]) < 0 // детерминированный порядок при равной степени
	})
	return nodes, nil
}

func (gs *GraphService) suggestHamiltonianEdges(
	ctx context.Context,
	graphID uuid.UUID,
	al domain.AdjacencyList,
	isDirected bool,
	startNode uuid.UUID,
) (pathEdges, cycleEdges []domain.Edge, err error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if len(al) == 0 {
		return nil, nil, errors.New("graph is empty")
	}
	if _, ok := al[startNode]; !ok {
		return nil, nil, errors.New("start node does not belong to graph")
	}

	// 1. startNode — первым, остальные — по UUID.
	rest := make([]uuid.UUID, 0, len(al)-1)
	for v := range al {
		if v != startNode {
			rest = append(rest, v)
		}
	}
	sort.Slice(rest, func(i, j int) bool {
		return bytes.Compare(rest[i][:], rest[j][:]) < 0
	})
	vertices := make([]uuid.UUID, 0, len(al))
	vertices = append(vertices, startNode)
	vertices = append(vertices, rest...)

	// 2. Ключ для set'а рёбер. Для undirected нормализуем (min, max),
	//    чтобы (u, v) и (v, u) считались одним ребром.
	type pair struct{ from, to uuid.UUID }
	makeKey := func(from, to uuid.UUID) pair {
		if !isDirected && bytes.Compare(from[:], to[:]) > 0 {
			from, to = to, from
		}
		return pair{from, to}
	}

	existing := make(map[pair]struct{}, len(vertices))
	for from, neighbors := range al {
		for _, to := range neighbors {
			if from == to {
				continue // self-loop не участвует в гамильтоновом пути
			}
			existing[makeKey(from, to)] = struct{}{}
		}
	}

	add := func(slice *[]domain.Edge, from, to uuid.UUID) {
		*slice = append(*slice, domain.Edge{
			GraphID:  graphID,
			SourceID: from,
			TargetID: to,
			Weight:   1,
		})
		existing[makeKey(from, to)] = struct{}{}
	}

	n := len(vertices)
	if n == 1 {
		return nil, nil, nil
	}

	for i := 0; i < n-1; i++ {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		from, to := vertices[i], vertices[i+1]
		if _, ok := existing[makeKey(from, to)]; !ok {
			add(&pathEdges, from, to)
		}
	}

	cycleEdges = make([]domain.Edge, len(pathEdges))
	copy(cycleEdges, pathEdges)

	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	from, to := vertices[n-1], vertices[0] // vertices[0] == startNode
	if _, ok := existing[makeKey(from, to)]; !ok {
		add(&cycleEdges, from, to)
	}

	return pathEdges, cycleEdges, nil
}

type HamiltonianResult struct {
	StartNodeID uuid.UUID
	PathEdges   []domain.Edge // достройка до полугамильтонова (существует путь)
	CycleEdges  []domain.Edge // достройка до гамильтонова  (существует цикл)
	Kind        int
}

func pathKey(ids []uuid.UUID) string {
	sorted := append([]uuid.UUID(nil), ids...)
	sort.Slice(sorted, func(i, j int) bool {
		return bytes.Compare(sorted[i][:], sorted[j][:]) < 0
	})
	var sb strings.Builder
	for _, id := range sorted {
		sb.WriteString(id.String())
	}
	return sb.String()
}

func (gs *GraphService) RunHamiltonian(
	ctx context.Context,
	graphID uuid.UUID,
	out chan<- GamiltonPath,
	limit int,
) (HamiltonianResult, error) {
	result := HamiltonianResult{
		PathEdges:  []domain.Edge{},
		CycleEdges: []domain.Edge{},
		Kind:       isNotHamiltonian,
	}

	if out == nil {
		return result, ErrNilChan
	}
	defer close(out)

	if err := ctx.Err(); err != nil {
		return result, err
	}
	if limit < 0 {
		return result, fmt.Errorf("limit must be >= 0, got %d", limit)
	}

	// 1. Connectivity check — отсекает несвязные до запуска воркеров.
	nodes, edges, isDirected, err := gs.getNodesAndEdges(ctx, graphID, false)
	if err != nil {
		return result, err
	}
	if len(nodes) == 0 {
		return result, errors.New("graph is empty")
	}
	if !isGraphConnected(nodes, edges) {
		return result, NonConnectedGraphError
	}
	al, _, _, err := toAdjacencyList(nodes, edges, isDirected)
	if err != nil {
		return result, err
	}

	startNodes, err := findStartNodesForHamilton(al)
	if err != nil {
		return result, err
	}
	result.StartNodeID = startNodes[0]

	if err := ctx.Err(); err != nil {
		return result, err
	}

	searchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	internal := make(chan GamiltonPath)
	type outcome struct {
		kind int
		err  error
	}
	outcomes := make(chan outcome, len(startNodes))

	var wg sync.WaitGroup
	for _, startNode := range startNodes {
		wg.Add(1)
		go func(sn uuid.UUID) {
			defer wg.Done()
			kind, werr := gs.findAllHamiltonianPathes(
				searchCtx, al, edges, isDirected, sn, internal, limit, -1,
			)
			outcomes <- outcome{kind, werr}
		}(startNode)
	}
	go func() {
		wg.Wait()
		close(internal)
		close(outcomes)
	}()

	// 2. Dedup по множеству рёбер + сбор bestKind.
	unlimited := limit == 0
	seen := make(map[string]struct{}, 8)
	total := 0
	limitReached := false

	for gp := range internal {
		if limitReached {
			continue // дочитываем, чтобы воркеры не зависли на отправке
		}

		key := pathKey(gp.Path)
		if _, dup := seen[key]; dup {
			continue // дуп: не растим total, не форвардим
		}
		seen[key] = struct{}{}

		select {
		case out <- gp:
		case <-ctx.Done():
			cancel()
			limitReached = true
			continue
		}
		total++
		if !unlimited && total >= limit {
			limitReached = true
			cancel()
		}
	}

	bestKind := isNotHamiltonian
	var firstErr error
	for res := range outcomes {
		if res.kind == isHamiltonian {
			bestKind = isHamiltonian
		} else if res.kind == isHalfHamiltonian && bestKind != isHamiltonian {
			bestKind = isHalfHamiltonian
		}
		if res.err != nil && !errors.Is(res.err, context.Canceled) && firstErr == nil {
			firstErr = res.err
		}
	}
	if firstErr == nil {
		firstErr = ctx.Err()
	}

	result.Kind = bestKind

	// 3. Suggestions — только если ничего не нашли.
	if bestKind == isNotHamiltonian {
		pathEdges, cycleEdges, serr := gs.suggestHamiltonianEdges(
			ctx, graphID, al, isDirected, startNodes[0],
		)
		if serr != nil {
			return result, serr
		}
		result.PathEdges = pathEdges
		result.CycleEdges = cycleEdges
	}

	return result, firstErr
}
