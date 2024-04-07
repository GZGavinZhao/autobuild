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
	cmdQuery = &cobra.Command{
		Use:   "query [src|bin|repo:path] [packages]",
		Short: "Query the build order of the given packages or the currently unsynced packages",
		Run:   runQuery,
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
}

func runQuery(cmd *cobra.Command, args []string) {
	tpath := args[0]
	queries := args[1:]

	state, err := state.LoadState(tpath)
	if err != nil {
		waterlog.Fatalf("Failed to parse state: %s\n", err)
	}
	waterlog.Goodln("Successfully parsed state!")

	qset := map[int]bool{}
	for _, query := range queries {
		idx, ok := state.NameToSrcIdx()[query]
		if !ok {
			waterlog.Fatalf("Unable to find package %s\n", query)
		}

		pkg := state.Packages()[idx]
		if !pkg.Resolved {
			waterlog.Warnf("Package %s has unresolved build dependencies, build graph may be incomplete!\n", pkg.Name)
		}

		qset[idx] = true
	}
	waterlog.Goodln("Found all requested packages in state!")

	depGraph := state.DepGraph()
	lifted, err := utils.LiftGraph(depGraph, func(i int) bool { return qset[i] })
	if err != nil {
		waterlog.Fatalf("Failed to lift final graph from requested nodes: %s\n", err)
	}
	waterlog.Goodln("Successfully built dependency graph!")

	if len(dotPath) > 0 {
		liftedDot, _ := os.Create(dotPath)
		_ = draw.DOT(lifted, liftedDot)
	}

	order, err := graph.TopologicalSort(lifted)
	if err != nil {
		// Try to dump cycles if topological sort failed.
		if cycles, err := graph.StronglyConnectedComponents(lifted); err == nil {
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

	waterlog.Good("Build order: ")
	for _, orderIdx := range order {
		fmt.Printf("%s ", state.Packages()[orderIdx].Name)
	}
	fmt.Println()
}
