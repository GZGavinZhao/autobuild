package utils

import (
	"github.com/dominikbraun/graph"
)

func copyVertex[K comparable, T any](vertex K, from *graph.Graph[K, T], to *graph.Graph[K, T]) (err error) {
	val, attrMap, err := (*from).VertexWithProperties(vertex)
	if err != nil {
		return
	}

	err = (*to).AddVertex(val, copyVertexProperties(attrMap))
	return
}

func copyVertexProperties(source graph.VertexProperties) func(*graph.VertexProperties) {
	return func(p *graph.VertexProperties) {
		for k, v := range source.Attributes {
			p.Attributes[k] = v
		}
		p.Weight = source.Weight
	}
}

func LiftGraph(g *graph.Graph[int, int], choose func(int) bool) (res graph.Graph[int, int], err error) {
	visited := make(map[int]bool)
	res = graph.New(graph.IntHash, graph.Directed(), graph.PreventCycles())
	adjMap, err := (*g).AdjacencyMap()

	if err != nil {
		return
	}

	for node := range adjMap {
		if choose(node) {
			err = copyVertex(node, g, &res)
			if err != nil {
				return
			}
		}
	}

	for node := range adjMap {
		err = liftDfs(node, node, choose, g, visited, &res)
		if err != nil {
			return
		}
	}

	return
}

func liftDfs(node int, parent int, choose func(int) bool, g *graph.Graph[int, int] /* gm *map[int]map[int]graph.Edge[int] */, visited map[int]bool, res *graph.Graph[int, int]) (err error) {
	if visited[node] {
		return
	}
	visited[node] = true

	adjMap, err := (*g).AdjacencyMap()
	if err != nil {
		return err
	}
	for adj := range adjMap[node] {
		nextp := parent
		if choose(adj) {
			nextp = adj
			(*res).AddEdge(parent, adj)
		}

		err = liftDfs(adj, nextp, choose, g, visited, res)
		if err != nil {
			return
		}
	}

	return
}

func RemoveVertexWithEdges[K comparable, T any](g *graph.Graph[K, T], node K) error {
	adjMap, err := (*g).AdjacencyMap()
	if err != nil {
		return err
	}

	for adj := range adjMap[node] {
		err = (*g).RemoveEdge(node, adj)
		if err != nil {
			return err
		}
	}

	return (*g).RemoveVertex(node)
}
