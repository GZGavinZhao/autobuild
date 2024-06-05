// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"errors"
	"fmt"

	"github.com/DataDrake/waterlog"
	st "github.com/GZGavinZhao/autobuild/state"
	"github.com/GZGavinZhao/autobuild/utils"
	"github.com/spf13/cobra"
	"github.com/yourbasic/graph"
)

var (
	dotPath  string
	tiers    bool
	forward  int
	reverse  int
	detailed bool

	cmdQuery = &cobra.Command{
		Use:   "query [src|bin|repo:path] [packages]",
		Short: "Query the build order of the given packages",
		Long: `Query the build order of the given packages. For example: autobuild query src:../packages rocm-clr pytorch

When no arguments are passed, it tries to compute a build order of all the packages it can find.`,
		Run: runQuery,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("expects one arg for path to binary index or source repo")
			}
			return nil
		},
	}
)

func init() {
	cmdQuery.Flags().StringVar(&dotPath, "dot", "", "store the final build graph at the specified location in the DOT format")
	cmdQuery.Flags().BoolVarP(&tiers, "tiers", "t", false, "output tier-ed build order")
	cmdQuery.Flags().IntVarP(&forward, "forward", "F", 0, "extra level(s) of packages that depends on the list provided")
	cmdQuery.Flags().IntVarP(&reverse, "reverse", "R", 0, "extra level(s) of packages that the list provided depends on")
	// TODO(GZGavinZhao): do we always want detailed output?
	// Maybe we should output packages that are not parts of the query but are
	// on the dependency chain with a different color?
	cmdQuery.Flags().BoolVar(&detailed, "detailed", true, "report more detailed dependency chains during cycles output")
}

func runQuery(cmd *cobra.Command, args []string) {
	tpath := args[0]

	state, err := st.LoadState(tpath)
	if err != nil {
		waterlog.Fatalf("Failed to parse state: %s\n", err)
	}
	waterlog.Goodln("Successfully parsed state!")

	depGraph := state.DepGraph()
	if depGraph == nil {
		waterlog.Fatalf("Failed to obtain adjacency map for dependency graph: %s\n", err)
	}
	qset := map[int]bool{}

	revGraph := graph.Transpose(depGraph)

	var queries []string
	if len(args) < 2 {
		waterlog.Infoln("No packages are provided, will try to query all packages")
		queries = make([]string, len(state.Packages()))
		i := 0
		for _, pkg := range state.Packages() {
			queries[i] = pkg.Name
			i++
		}
	} else {
		queries = args[1:]
	}

	for _, query := range queries {
		pkg, idx := st.GetPackage(state, query)
		if idx < 0 {
			waterlog.Fatalf("Unable to find package %s\n", query)
		}

		if unresolved := pkg.Resolve(state.NameToSrcIdx(), state.Packages()); len(unresolved) > 0 {
			waterlog.Warnf("Package %s has unresolved build dependencies, build graph may be incomplete:", pkg.Name)
			for _, pkg := range unresolved {
				waterlog.Printf(" %s", pkg)
			}
			waterlog.Println()
		}

		qset[idx] = true
		utils.BFSWithDepth(depGraph, idx, func(node int, depth int) bool {
			if depth > forward {
				return true
			}
			qset[node] = true
			return false
		})
		if reverse > 0 {
			utils.BFSWithDepth(revGraph, idx, func(node int, depth int) bool {
				if depth > reverse {
					return true
				}
				qset[node] = true
				return false
			})
		}
	}
	// waterlog.Debugf("query: %q\n", queries)
	// waterlog.Debugf("qset: %v\n", qset)
	waterlog.Goodln("Found all requested packages in state!")

	lifted := graph.Sort(utils.LiftGraph(depGraph, func(i int) bool { return qset[i] }))
	if err != nil {
		waterlog.Fatalf("Failed to lift final graph from requested nodes: %s\n", err)
	}
	waterlog.Goodln("Successfully built dependency graph!")

	waterlog.Debugf("depgraph hash: %s\n", utils.GraphHash(depGraph))
	waterlog.Debugf("depgraph stats: %+v\n", graph.Check(depGraph))
	waterlog.Debugf("liftgraph hash: %s\n", utils.GraphHash(lifted))
	waterlog.Debugf("liftgraph stats: %+v\n", graph.Check(lifted))

	// {
	// 	bruh, _ := lifted.AdjacencyMap()
	// 	for node, edges := range bruh {
	// 		if len(edges) == 0 {
	// 			continue
	// 		}

	// 		fmt.Printf("%d: ", node)
	// 		for adj := range edges {
	// 			fmt.Printf("%d ", adj)
	// 		}
	// 		fmt.Println()
	// 	}
	// 	// fmt.Printf("%s : %q\n", err, bruh)
	// }
	// if len(dotPath) > 0 {
	// 	liftedDot, _ := os.Create("lifted.gv")
	// 	if err = draw.DOT(lifted, liftedDot); err != nil {
	// 		waterlog.Fatalf("Failed to output lifted.gv: %s\n", err)
	// 	}

	// 	depDot, _ := os.Create("dep.gv")
	// 	if err = draw.DOT(*depGraph, depDot); err != nil {
	// 		waterlog.Fatalf("Failed to output dep.gv: %s\n", err)
	// 	}
	// }

	order, ok := utils.TieredTopSort(lifted)
	if !ok {
		// Try to dump cycles if topological sort failed.
		// if cycles, err := graph.StrongComponents(lifted); err == nil {
		cycles := graph.StrongComponents(lifted)
		cycles = utils.Filter(cycles, func(cycle []int) bool { return len(cycle) > 1 })
		if len(cycles) == 0 {
			waterlog.Fatalln("No cycles detected ?!?")
		}

		for cycleIdx, cycle := range cycles {
			if len(cycle) <= 1 {
				continue
			}

			waterlog.Warnf("Cycle %d:", cycleIdx+1)
			cycleIdx++

			for _, nodeIdx := range cycle {
				waterlog.Printf(" %s", state.Packages()[nodeIdx].Name)
			}
			waterlog.Println()

			// the order in `cycle` may not be deterministic, so we have to
			// deterministically choose a starting node by ourselves
			startIdx := 0
			for idx, nodeIdx := range cycle {
				if nodeIdx < cycle[startIdx] {
					startIdx = idx
				}
			}
			nextIdx := (startIdx + 1) % len(cycle)

			// We always want the longer shortest path
			var depPath []int
			if detailed {
				depPath = utils.LongerShortestPath(depGraph, cycle[startIdx], cycle[nextIdx])
			} else {
				depPath = utils.LongerShortestPath(lifted, cycle[startIdx], cycle[nextIdx])
			}
			if len(depPath) < 2 {
				waterlog.Fatalf("Failed to calculate dependency path that led to this cycle, got: %q\n", depPath)
			}

			waterlog.Warnln("One of the dependency chains that led to this cycle:")
			for _, pidx := range depPath {
				waterlog.Printf("%s -> ", state.Packages()[pidx].Name)
			}
			waterlog.Println(state.Packages()[depPath[0]].Name)
		}
		// } else {
		// 	waterlog.Errorf("Failed to get SCC: %s\n", err)
		// }

		waterlog.Fatalln("Failed to get topological sort order: lifted graph has cycles!")
	}

	// Note that we still need an extra filter on the tier output,
	// because due to the limitation of the graph API, the lifted graph
	// includes nodes [0, n), not just the nodes in `query`/`qset`, so they will
	// appear in the topological sort output.
	if tiers {
		waterlog.Goodln("Build order:")
		for tIdx, tier := range order {
			tier = utils.Filter(tier, func(i int) bool { return qset[i] })
			waterlog.Goodf("Tier %d: ", tIdx+1)
			for _, pkgIdx := range tier {
				fmt.Printf("%s ", state.Packages()[pkgIdx].Name)
			}
			fmt.Println()
		}
	} else {
		waterlog.Good("Build order: ")
		tier := utils.Filter(utils.Flatten(order), func(i int) bool { return qset[i] })
		for _, orderIdx := range tier {
			fmt.Printf("%s ", state.Packages()[orderIdx].Name)
		}
		fmt.Println()
	}
}
