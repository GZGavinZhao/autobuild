package cmd

import (
	"fmt"
	"os"

	"github.com/DataDrake/waterlog"
	"github.com/GZGavinZhao/autobuild/common"
	"github.com/GZGavinZhao/autobuild/utils"
	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
	"github.com/spf13/cobra"
)

var (
	dotPath  string
	cmdQuery = &cobra.Command{
		Use:   "query [packages]",
		Short: "Query the build order of the given packages or the currently unsynced packages",
		Run:   runQuery,
	}
)

func init() {
	pathsInit(cmdQuery)
	cmdQuery.LocalFlags().StringVar(&dotPath, "dot", "", "stores the final build graph at the specified location in the DOT format")
}

func runQuery(cmd *cobra.Command, args []string) {
	srcPkgs, nameToSrcIdx, depGraph, err := common.PrepareSrcAndDepGraph(sourcesPath, indexPath)
	if err != nil {
		waterlog.Fatalf("Failed to parse and construct dependency graph: %s\n", err)
	}

	qset := map[int]bool{}
	if len(args) > 0 {
		for _, pkg := range args {
			idx, ok := nameToSrcIdx[pkg]
			if !ok {
				waterlog.Fatalf("Unable to find package", pkg)
			}

			if !srcPkgs[idx].Resolved {
				waterlog.Warnf("Package %s has unresolved build dependencies, build graph may be incomplete!\n", srcPkgs[idx].Name)
			}

			qset[idx] = true
		}
	} else {
		for idx, pkg := range srcPkgs {
			if !pkg.Synced && pkg.Resolved /* || !pkg.Resolved */ {
				qset[idx] = true
			}
		}
	}

	fing, err := utils.LiftGraph(&depGraph, func(i int) bool {
		return qset[i]
	})
	if err != nil {
		waterlog.Fatalf("Failed to lift final graph from requested nodes: %s\n", err)
	}

	if len(dotPath) > 0 {
		fingDot, _ := os.Create(dotPath)
		_ = draw.DOT(fing, fingDot)
	}

	order, err := graph.TopologicalSort(fing)
	if err != nil {
		waterlog.Fatalf("Failed to get topological sort order: %s\n", err)
	}

	waterlog.Good("Build order: ")
	for _, orderIdx := range order {
		fmt.Printf("%s ", srcPkgs[orderIdx].Name)
	}
	fmt.Println()
}
