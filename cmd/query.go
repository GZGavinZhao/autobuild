// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/DataDrake/waterlog"
	"github.com/GZGavinZhao/autobuild/state"
	"github.com/GZGavinZhao/autobuild/utils"
	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
	"github.com/spf13/cobra"
)

var (
	dotPath  string
	tiers    bool
	forward  int
	reverse  int
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
	cmdQuery.Flags().StringVar(&dotPath, "dot", "", "stores the final build graph at the specified location in the DOT format")
	cmdQuery.Flags().BoolVarP(&tiers, "tiers", "t", false, "output tier-ed build order")
	cmdQuery.Flags().IntVarP(&forward, "forward", "F", 0, "extra level(s) of packages that depends on the list provided")
	cmdQuery.Flags().IntVarP(&reverse, "reverse", "R", 0, "extra level(s) of packages that the list provided depends on")
}

func runQuery(cmd *cobra.Command, args []string) {
	tpath := args[0]

	state, err := state.LoadState(tpath)
	if err != nil {
		waterlog.Fatalf("Failed to parse state: %s\n", err)
	}
	waterlog.Goodln("Successfully parsed state!")

	depGraph := state.DepGraph()
	if depGraph == nil {
		waterlog.Fatalf("Failed to obtain adjacency map for dependency graph: %s\n", err)
	}
	qset := map[int]bool{}

	var revGraph *graph.Graph[int, int]
	if reverse > 0 {
		if revGraph, err = utils.ReverseGraph(depGraph); err != nil {
			waterlog.Fatalf("Failed to reverse dependency graph: %s\n", err)
		}
	}

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
		idx, ok := state.NameToSrcIdx()[query]
		if !ok {
			waterlog.Fatalf("Unable to find package %s\n", query)
		}

		pkg := state.Packages()[idx]
		if unresolved := pkg.Resolve(state.NameToSrcIdx()); len(unresolved) > 0 {
			waterlog.Warnf("Package %s has unresolved build dependencies, build graph may be incomplete:", pkg.Name)
			for _, pkg := range unresolved {
				waterlog.Printf(" %s", pkg)
			}
			waterlog.Println()
		}

		qset[idx] = true
		utils.BFSWithDepth(*depGraph, idx, func(node int, depth int) bool {
			if depth > forward {
				return true
			}
			qset[node] = true
			return false
		})
		if reverse > 0 {
			utils.BFSWithDepth(*revGraph, idx, func(node int, depth int) bool {
				if depth > reverse {
					return true
				}
				qset[node] = true
				return false
			})
		}
	}
	waterlog.Goodln("Found all requested packages in state!")

	lifted, err := utils.LiftGraph(depGraph, func(i int) bool { return qset[i] })
	if err != nil {
		waterlog.Fatalf("Failed to lift final graph from requested nodes: %s\n", err)
	}
	waterlog.Goodln("Successfully built dependency graph!")

	if len(dotPath) > 0 {
		liftedDot, _ := os.Create(dotPath)
		_ = draw.DOT(lifted, liftedDot)
	}

	order, err := utils.TopologicalSort(lifted)
	if err != nil {
		// Try to dump cycles if topological sort failed.
		if cycles, err := graph.StronglyConnectedComponents(lifted); err == nil {
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
				path1, err := graph.ShortestPath(*depGraph, cycle[startIdx], cycle[nextIdx])
				if err != nil {
					waterlog.Warnf("Failed to calculate dependency chain that formed this cycle!")
				}
				path2, err := graph.ShortestPath(*depGraph, cycle[nextIdx], cycle[startIdx])
				if err != nil {
					waterlog.Warnf("Failed to calculate dependency chain that formed this cycle!")
				}

				if len(path1) < len(path2) {
					path1 = path2
				}

				waterlog.Warnln("Dependency chain that led to this cycle:")
				for _, pidx := range path1 {
					waterlog.Printf("%s -> ", state.Packages()[pidx].Name)
				}
				waterlog.Println(state.Packages()[path1[0]].Name)
			}
		} else {
			waterlog.Errorf("Failed to get SCC: %s\n", err)
		}

		waterlog.Fatalf("Failed to get topological sort order: %s\n", err)
	}

	if tiers {
		waterlog.Goodln("Build order:")
		for tIdx, tier := range order {
			waterlog.Goodf("Tier %d: ", tIdx+1)
			for _, pkgIdx := range tier {
				fmt.Printf("%s ", state.Packages()[pkgIdx].Name)
			}
			fmt.Println()
		}
	} else {
		waterlog.Good("Build order: ")
		for _, orderIdx := range utils.Flatten(order) {
			fmt.Printf("%s ", state.Packages()[orderIdx].Name)
		}
		fmt.Println()
	}
}
