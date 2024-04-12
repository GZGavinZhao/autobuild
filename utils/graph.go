// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"cmp"
	"errors"
	_ "fmt"
	"slices"

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

func TopologicalSort[K cmp.Ordered, T any](g graph.Graph[K, T]) ([][]K, error) {
	res := [][]K{}

	adjMap, err := g.AdjacencyMap()
	if err != nil {
		return res, err
	}

	indeg := make(map[K]int)
	occupied := make(map[K]bool)
	for node, edges := range adjMap {
		if _, ok := indeg[node]; !ok {
			indeg[node] = 0
		}
		for adj := range edges {
			indeg[adj]++
		}
	}

	queue := []K{}
	for node, deg := range indeg {
		if deg == 0 {
			queue = append(queue, node)
		}
	}

	for len(queue) > 0 {
		slices.Sort(queue)
		res = append(res, queue)

		l := len(queue)
		for i := 0; i < l; i++ {
			node := queue[0]
			for adj := range adjMap[node] {
				indeg[adj]--
				if indeg[adj] == 0 && !occupied[adj] {
					occupied[adj] = true
					queue = append(queue, adj)
				}
			}

			queue = queue[1:]
		}
	}

	for _, deg := range indeg {
		if deg > 0 {
			return res, errors.New("topological sort cannot be computed on graph with cycles")
		}
	}

	return res, nil
}

func LiftGraph(g *graph.Graph[int, int], choose func(int) bool) (res graph.Graph[int, int], err error) {
	visited := make(map[int]bool)
	res = graph.New(graph.IntHash, graph.Directed(), graph.Acyclic())
	adjMap, err := (*g).AdjacencyMap()

	if err != nil {
		return
	}

	for node := range adjMap {
		if choose(node) {
			if err = copyVertex(node, g, &res); err != nil {
				return
			}
		}
	}

	// The deterministic node traversal
	// for node := 0; node < len(adjMap); node++ {
	// 	actualNode := node
	// 	if choose(actualNode) {
	// 		for adj := range adjMap[actualNode] {
	// 			// println("Starting with", actualNode, "going to", adj)
	// 			if err = liftDfs(adj, actualNode, choose, adjMap, visited, &res); err != nil {
	// 				return
	// 			}
	// 		}
	// 	}
	// }

	// The non-deterministic node traversal
	for actualNode := range adjMap {
		if choose(actualNode) {
			for adj := range adjMap[actualNode] {
				// println("Starting with", actualNode, "going to", adj)
				if err = liftDfs(adj, actualNode, choose, adjMap, visited, &res); err != nil {
					return
				}
			}
		}
	}

	return
}

func liftDfs(node int, parent int, choose func(int) bool, gm map[int]map[int]graph.Edge[int], visited map[int]bool, res *graph.Graph[int, int]) error {
	if node == parent {
		// err = errors.New("wtf node is parent???")
		return nil
	}

	// for adj := range gm[node] {
	// 	// if adj == parent {
	// 	// 	continue
	// 	// }

	// 	nextp := parent

	// 	if choose(adj) {
	// 		nextp = adj

	// 		if err = (*res).AddEdge(parent, adj, graph.EdgeWeight(1)); err != nil && /* !errors.Is(err, graph.ErrEdgeCreatesCycle) && */ !errors.Is(err, graph.ErrEdgeAlreadyExists) {
	// 			return
	// 		} else {
	// 			err = nil
	// 		}
	// 	}

	// 	if err = liftDfs(adj, nextp, choose, gm, visited, res); err != nil {
	// 		return
	// 	}
	// }

	nextp := parent
	if choose(node) {
		// println(parent, "->", node)
		if err := (*res).AddEdge(parent, node, graph.EdgeWeight(1)); err != nil && !errors.Is(err, graph.ErrEdgeAlreadyExists) {
			return err
		}
		nextp = node
	}

	if visited[node] {
		return nil
	}
	visited[node] = true

	for adj := range gm[node] {
		if err := liftDfs(adj, nextp, choose, gm, visited, res); err != nil {
			return err
		}
	}

	return nil
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

// I think I accidentally implemented
// https://github.com/dominikbraun/graph/issues/39. Just need to change some
// indeg initial values and conditions and I think that's all XD
//
// Obviously, this is not used, because I didn't realize I'm working with directed
// graphs until I finished...
func SimpleCycles[K comparable, T any](g graph.Graph[K, T]) ([][]K, error) {
	res := make([][]K, 0)

	gm, err := g.AdjacencyMap()
	if err != nil {
		return res, err
	}

	indeg := make(map[K]int)
	for _, adjM := range gm {
		for adj := range adjM {
			indeg[adj]++
		}
	}

	queue := []K{}
	for node, deg := range indeg {
		if deg == 0 {
			queue = append(queue, node)
		}
	}

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		for adj := range gm[node] {
			indeg[adj]--

			if indeg[adj] == 0 {
				queue = append(queue, adj)
			}
		}
	}

	visited := make(map[K]bool)
	for node, deg := range indeg {
		if deg == 0 || visited[node] {
			continue
		}

		cycle := []K{}
		queue := []K{node}
		for len(queue) > 0 {
			node := queue[0]
			queue = queue[1:]

			if visited[node] {
				continue
			}
			visited[node] = true
			cycle = append(cycle, node)

			for adj := range gm[node] {
				if indeg[adj] != 0 && !visited[adj] {
					queue = append(queue, adj)
				}
			}
		}

		res = append(res, cycle)
	}

	return res, err
}
