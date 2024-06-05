// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"encoding/base64"
	"slices"

	"github.com/yourbasic/graph"
	"github.com/zeebo/blake3"
)

// func copyVertex[K comparable, T any](vertex K, from *graph.Graph[K, T], to *graph.Graph[K, T]) (err error) {
// 	val, attrMap, err := (*from).VertexWithProperties(vertex)
// 	if err != nil {
// 		return
// 	}
//
// 	err = (*to).AddVertex(val, copyVertexProperties(attrMap))
// 	return
// }
//
// func copyVertexProperties(source graph.VertexProperties) func(*graph.VertexProperties) {
// 	return func(p *graph.VertexProperties) {
// 		for k, v := range source.Attributes {
// 			p.Attributes[k] = v
// 		}
// 		p.Weight = source.Weight
// 	}
// }

func TieredTopSort(g graph.Iterator) ([][]int, bool) {
	indegree := make([]int, g.Order())
	for v := range indegree {
		g.Visit(v, func(w int, _ int64) (skip bool) {
			indegree[w]++
			return
		})
	}

	var res [][]int
	// Invariant: this queue holds all vertices with indegree 0.
	var queue []int
	for v, degree := range indegree {
		if degree == 0 {
			queue = append(queue, v)
		}
	}

	vertexCount := 0

	for len(queue) > 0 {
		slices.Sort(queue)
		res = append(res, queue)

		l := len(queue)
		for i := 0; i < l; i++ {
			v := queue[0]
			queue = queue[1:]

			vertexCount++
			g.Visit(v, func(w int, _ int64) (skip bool) {
				indegree[w]--
				if indegree[w] == 0 {
					queue = append(queue, w)
				}
				return false
			})
		}
	}

	return res, vertexCount == g.Order()
}

func LiftGraph(g graph.Iterator, choose func(int) bool) (res *graph.Mutable) {
	visited := make(map[int]bool)
	res = graph.New(g.Order())

	// // The deterministic node traversal
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

	// // The non-deterministic node traversal
	// for actualNode := range adjMap {
	// 	if choose(actualNode) {
	// 		for adj := range adjMap[actualNode] {
	// 			// println("Starting with", actualNode, "going to", adj)
	// 			if err = liftDfs(adj, actualNode, choose, adjMap, visited, &res); err != nil {
	// 				return
	// 			}
	// 		}
	// 	}
	// }

	for node := 0; node < g.Order(); node++ {
		if choose(node) {
			g.Visit(node, func(adj int, _ int64) (skip bool) {
				liftDfs(adj, node, choose, g, visited, res)
				return false
			})
		}
	}

	return
}

func liftDfs(node int, parent int, choose func(int) bool, g graph.Iterator, visited map[int]bool, res *graph.Mutable) {
	if node == parent {
		return
	}

	nextp := parent
	if choose(node) {
		// println(parent, "->", node)
		res.Add(parent, node)
		nextp = node
	}

	if visited[node] {
		return
	}
	visited[node] = true

	g.Visit(node, func(adj int, _ int64) (skip bool) {
		liftDfs(adj, nextp, choose, g, visited, res)
		return false
	})
}

// func RemoveVertexWithEdges[K comparable, T any](g *graph.Graph[K, T], node K) error {
// 	adjMap, err := (*g).AdjacencyMap()
// 	if err != nil {
// 		return err
// 	}
//
// 	for adj := range adjMap[node] {
// 		err = (*g).RemoveEdge(node, adj)
// 		if err != nil {
// 			return err
// 		}
// 	}
//
// 	return (*g).RemoveVertex(node)
// }
//
// // I think I accidentally implemented
// // https://github.com/dominikbraun/graph/issues/39. Just need to change some
// // indeg initial values and conditions and I think that's all XD
// //
// // Obviously, this is not used, because I didn't realize I'm working with directed
// // graphs until I finished...
// func SimpleCycles[K comparable, T any](g graph.Graph[K, T]) ([][]K, error) {
// 	res := make([][]K, 0)
//
// 	gm, err := g.AdjacencyMap()
// 	if err != nil {
// 		return res, err
// 	}
//
// 	indeg := make(map[K]int)
// 	for _, adjM := range gm {
// 		for adj := range adjM {
// 			indeg[adj]++
// 		}
// 	}
//
// 	queue := []K{}
// 	for node, deg := range indeg {
// 		if deg == 0 {
// 			queue = append(queue, node)
// 		}
// 	}
//
// 	for len(queue) > 0 {
// 		node := queue[0]
// 		queue = queue[1:]
//
// 		for adj := range gm[node] {
// 			indeg[adj]--
//
// 			if indeg[adj] == 0 {
// 				queue = append(queue, adj)
// 			}
// 		}
// 	}
//
// 	visited := make(map[K]bool)
// 	for node, deg := range indeg {
// 		if deg == 0 || visited[node] {
// 			continue
// 		}
//
// 		cycle := []K{}
// 		queue := []K{node}
// 		for len(queue) > 0 {
// 			node := queue[0]
// 			queue = queue[1:]
//
// 			if visited[node] {
// 				continue
// 			}
// 			visited[node] = true
// 			cycle = append(cycle, node)
//
// 			for adj := range gm[node] {
// 				if indeg[adj] != 0 && !visited[adj] {
// 					queue = append(queue, adj)
// 				}
// 			}
// 		}
//
// 		res = append(res, cycle)
// 	}
//
// 	return res, err
// }
//
// // Warning: this doesn't preserve any extra information on the edges/nodes!
// func ReverseGraph(g *graph.Graph[int, int]) (*graph.Graph[int, int], error) {
// 	res := graph.New(graph.IntHash, graph.Directed(), graph.Acyclic())
// 	adjMap, err := (*g).AdjacencyMap()
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	for node := range adjMap {
// 		if err := copyVertex(node, g, &res); err != nil {
// 			return nil, err
// 		}
// 	}
//
// 	for node, edges := range adjMap {
// 		for adj := range edges {
// 			if err := res.AddEdge(adj, node); err != nil {
// 				return nil, err
// 			}
// 		}
// 	}
//
// 	return &res, nil
// }

// BFSWithDepth works just as BFS and performs a breadth-first search on the graph, but its
// visit function is passed the current depth level as a second argument. Consequently, the
// current depth can be used for deciding whether or not to proceed past a certain depth.
//
//	_ = utils.BFSWithDepth(g, 1, func(value int, depth int) bool {
//		fmt.Println(value)
//		return depth > 3
//	})
//
// With the visit function from the example, the BFS traversal will stop once a depth greater
// than 3 is reached. Note that depth is calculated by treating the start node
// as having depth 0.
func BFSWithDepth(g *graph.Immutable, start int, visit func(int, int) bool) {
	queue := make([]int, 0)
	visited := make(map[int]bool)
	depths := make(map[int]int)

	visited[start] = true
	queue = append(queue, start)
	depths[start] = 0

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		// Stop traversing the graph if the visit function returns true.
		if stop := visit(node, depths[node]); stop {
			break
		}

		g.Visit(node, func(adj int, _ int64) bool {
			if _, ok := visited[adj]; !ok {
				visited[adj] = true
				depths[adj] = depths[node] + 1
				queue = append(queue, adj)
			}
			return false
		})
	}
}

func GraphHash(g *graph.Immutable) string {
	hashBytes := blake3.Sum256([]byte(g.String()))
	return base64.StdEncoding.EncodeToString(hashBytes[:])
}

func LongerShortestPath(g graph.Iterator, u int, v int) []int {
	path1, dist1 := graph.ShortestPath(g, u, v)
	path2, dist2 := graph.ShortestPath(g, v, u)

	if dist1 == -1 && dist2 == -1 {
		return []int{}
	} else if len(path1) < len(path2) {
		return path2
	} else {
		return path1
	}
}
