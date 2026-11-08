package service

import (
	"context"
	"errors"
	"slices"

	"github.com/google/uuid"

	"github.com/Gospoduu/graphs-service/internal/base"
)

func (gs *GraphService) FordBelmanPathfinding(ctx context.Context, graph, start, end uuid.UUID) ([]uuid.UUID, error) {
	// start: uuid.UUID - start node; end uuid.UUID - end node -> []uuid.UUID - slice edges uuid
	al, edges, isDirected, err := gs.getAdjacencyList(ctx, graph, false)
	if err != nil {
		return nil, err
	}
	if _, ok := al[start]; !ok {
		return nil, errors.New("Unavalible start")
	}
	if _, ok := al[end]; !ok {
		return nil, errors.New("Unavalible end")
	}
	dist := make(map[uuid.UUID]float64)
	type step struct{ node, edge uuid.UUID }
	parent := make(map[uuid.UUID]step)
	dist[start] = 0

	for i := 0; i <= len(al)-1; i++ {
		changed := false
		for _, edge := range edges {
			u := edge.Source.ID
			v := edge.Target.ID
			w := edge.Weight

			dv, okv := dist[v]
			du, oku := dist[u]

			if oku && (!okv || du+w < dv) {
				dist[v] = du + w
				parent[v] = step{node: u, edge: edge.ID}
				dv, okv = dist[v], true
				changed = true
			}

			if !isDirected && okv && (!oku || dv+w < du) {
				dist[u] = dv + w
				parent[u] = step{node: v, edge: edge.ID}
				changed = true
			}
		}
		if !changed {
			break
		}
		if i == len(al)-1 {
			return nil, errors.New("graph contains a negative cycle")
		}
	}

	if _, ok := dist[end]; !ok {
		return nil, errors.New("end is unreachable from start")
	}

	path := []uuid.UUID{}
	for cur := end; cur != start; {
		s := parent[cur]
		path = append(path, s.edge)
		cur = s.node
	}
	slices.Reverse(path)
	return path, nil
}

type pqItem struct {
	node uuid.UUID
	dist float64
}

func (gs *GraphService) DijkstraPathfinding(ctx context.Context, graph, start, end uuid.UUID) ([]uuid.UUID, error) {
	// start: uuid.UUID - start node; end uuid.UUID - end node -> []uuid.UUID - slice edges uuid
	al, edges, _, err := gs.getAdjacencyListV2(ctx, graph, false)
	if err != nil {
		return nil, err
	}
	if _, ok := al[start]; !ok {
		return nil, errors.New("Unavalible start")
	}
	if _, ok := al[end]; !ok {
		return nil, errors.New("Unavalible end")
	}
	for _, e := range edges {
		if e.Weight < 0 {
			return nil, errors.New("graph contains a negative edge weight")
		}
	}

	type step struct{ node, edge uuid.UUID }
	parent := make(map[uuid.UUID]step)
	dist := map[uuid.UUID]float64{start: 0}
	pq := base.NewHeap(func(a, b pqItem) bool { return a.dist < b.dist })
	pq.Push(pqItem{node: start, dist: 0})

	for pq.Len() > 0 {
		it, _ := pq.Pop()
		u := it.node
		if it.dist > dist[u] {
			continue
		}
		if u == end {
			break
		}
		for _, edge := range al[u] {
			v := edge.TargetID
			if v == u {
				v = edge.SourceID
			}
			nd := it.dist + edge.Weight
			if dv, okv := dist[v]; !okv || nd < dv {
				dist[v] = nd
				parent[v] = step{node: u, edge: edge.ID}
				pq.Push(pqItem{node: v, dist: nd})
			}
		}
	}

	if _, ok := dist[end]; !ok {
		return nil, errors.New("end is unreachable from start")
	}

	path := []uuid.UUID{}
	for cur := end; cur != start; {
		s := parent[cur]
		path = append(path, s.edge)
		cur = s.node
	}
	slices.Reverse(path)
	return path, nil
}
