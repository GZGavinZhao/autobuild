// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package main

import (
	// "github.com/DataDrake/waterlog"
	"errors"
	"fmt"

	"github.com/dominikbraun/graph"
)

func subgraph(g *graph.Graph[int, int], startingNodes []int) (res graph.Graph[int, int], err error) {
	res = graph.New(graph.IntHash, graph.Directed(), graph.Acyclic())

	visited := make([]bool, len(srcPkgs))
	for _, startingNode := range startingNodes {
		err = dfsSubGraph(startingNode, g, visited[:], &res)
		if err != nil {
			return
		}
	}

	return
}

func dfsSubGraph(node int, g *graph.Graph[int, int], visited []bool, res *graph.Graph[int, int]) error {
	if visited[node] {
		// waterlog.Printf("Already visited %s, skipping\n", srcPkgs[node].Name)
		return nil
	}
	copyVertex(node, g, res)
	visited[node] = true
	// waterlog.Printf("Visiting %s\n", srcPkgs[node].Name)

	adjMap, err := (*g).AdjacencyMap()
	if err != nil {
		return err
	}

	for adj := range adjMap[node] {
		copyVertex(adj, g, res)
		err := (*res).AddEdge(node, adj)
		if err != nil {
			return err
		}

		err = dfsSubGraph(adj, g, visited, res)
		if err != nil {
			return err
		}
	}

	return nil
}

// Note: this is not a pure and general isolate function. I need to assign
// special properties to nodes depending on their value in the `srcPkgs` array
// (if they're updated or not).
func isolate(g *graph.Graph[int, int], containingNodes []int) (res graph.Graph[int, int], err error) {
	visited := make([]bool, len(srcPkgs))
	sizes := make([]int, len(srcPkgs))
	res = graph.New(graph.IntHash, graph.Directed(), graph.PreventCycles())

	for _, containNode := range containingNodes {
		sizes[containNode] = 1
	}

	for _, containNode := range containingNodes {
		err = dfsMark(containNode, g, visited, sizes)
		if err != nil {
			return
		}
	}

	for node := range visited {
		visited[node] = false
	}
	for _, containNode := range containingNodes {
		err = dfsIsolate(containNode, g, visited, sizes, &res)
		if err != nil {
			return
		}
	}

	return
}

func dfsIsolate[T any](node int, g *graph.Graph[int, T], visited []bool, sizes []int, res *graph.Graph[int, T]) (err error) {
	if visited[node] {
		return
	}
	copyVertex(node, g, res)
	visited[node] = true

	adjMap, err := (*g).AdjacencyMap()
	if err != nil {
		return
	}

	for adj := range adjMap[node] {
		if sizes[adj] == 0 {
			continue
		}

		copyVertex(adj, g, res)
		err = (*res).AddEdge(node, adj)
		if err != nil {
			return errors.New(fmt.Sprintf("Failed to construct edge from %s to %s: %s", srcPkgs[node].Name, srcPkgs[adj].Name, err))
		}

		err = dfsIsolate(adj, g, visited, sizes, res)
		if err != nil {
			return
		}
	}

	return
}

func dfsMark[T any](node int, g *graph.Graph[int, T], visited []bool, sizes []int) error {
	if visited[node] {
		return nil
	}
	visited[node] = true

	adjMap, err := (*g).AdjacencyMap()
	if err != nil {
		return err
	}

	for adj := range adjMap[node] {
		err := dfsMark(adj, g, visited, sizes)
		if err != nil {
			return err
		}

		sizes[node] += sizes[adj]
	}

	return nil
}

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

func liftgraph(g *graph.Graph[int, int], choose func(int) bool) (res graph.Graph[int, int], err error) {
	visited := make([]bool, len(srcPkgs))
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
		err = liftDfs(node, node, choose, g, visited[:], &res)
		if err != nil {
			return
		}
	}

	return
}

func liftDfs(node int, parent int, choose func(int) bool, g *graph.Graph[int, int] /* gm *map[int]map[int]graph.Edge[int] */, visited []bool, res *graph.Graph[int, int]) (err error) {
	if visited[node] {
		return
	}
	visited[node] = true

	// chosen := choose(node)
	// if chosen && parent != -1 {
	// 	err = copyVertex(node, g, res)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	err = (*res).AddEdge(parent, node)
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	// newp := parent
	// if chosen {
	// 	newp = node
	// }

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

// BFS
// indeg := make([]int, len(srcPkgs))
// adjMap, err := (*g).AdjacencyMap()
// if err != nil {
// 	return err
// }

// for _, edges := range adjMap {
// 	for adj := range edges {
// 		indeg[adj]++
// 	}
// }

// queue := make([]int, 0)
// for node := range adjMap {
// 	if indeg[node] == 0 {
// 		queue = append(queue, node)
// 	}
// }

// for len(queue) > 0 {
// 	node := queue[0]
// 	queue = queue[1:]

// 	for adj := range adjMap[node] {
// 		indeg[adj]--

// 		if indeg[adj] == 0 {
// 			queue = append(queue, adj)
// 		}
// 	}
// }
