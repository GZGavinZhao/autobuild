package cmd

import (
	"fmt"

	"github.com/DataDrake/waterlog"
	"github.com/GZGavinZhao/autobuild/common"
	"github.com/GZGavinZhao/autobuild/utils"
	"github.com/dominikbraun/graph"
	"github.com/spf13/cobra"
)

var (
	cmdQuery = &cobra.Command{
		Use:   "query <path-to-sources> <path-to-index> [packages]",
		Short: "Query the build order of the given packages or the currently unsynced packages",
		Run: runQuery,
	}
)

func runQuery(cmd *cobra.Command, args []string) {
	sourcesPath := args[0]
	indexPath := args[1]

	nameToSrcIdx := make(map[string]int)

	srcPkgs, err := common.ReadSrcPkgs(sourcesPath)
	if err != nil {
		waterlog.Fatalf("Failed to walk through sources: %s\n", err)
	}

	common.MapProvidesToIdx(srcPkgs[:], nameToSrcIdx)

	// Iterate through every source package, and check if all of their
	// dependencies are present in the source repository.
	//
	// If not, then it's possible that missing dependency is a new package that
	// needs to be built to generate the `pspec_x86_64.xml` file that shows all
	// the binary packages that a source package provides. This usually happens
	// when `a` is a new package that has yet to be built locally and some
	// package `b` depends on `a-devel`.
	for idx := range srcPkgs {
		srcPkgs[idx].Resolve(nameToSrcIdx)
	}

	waterlog.Goodln("Dependency resolving complete. Now scanning binary index...")

	err = common.CheckSrcPkgsSynced(indexPath, srcPkgs[:], nameToSrcIdx)
	if err != nil {
		waterlog.Fatalf("Failed to compare source packages with binary index %s: %s\n", indexPath, err)
	}
	waterlog.Goodln("Scanning binary index complete. Constructing dependency graph...")

	depGraph, err := common.BuildGraph(srcPkgs[:], nameToSrcIdx)

	qset := map[int]bool{}
	if len(args) > 3 {
		for _, pkg := range args[2:] {
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
			if !pkg.Synced /* || !pkg.Resolved */ {
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

	// fingDot, _ := os.Create("./fing.gv")
	// _ = draw.DOT(fing, fingDot)

	order, err := graph.TopologicalSort(fing)
	if err != nil {
		waterlog.Fatalf("Failed to get topological sort order: %s\n", err)
	}

	waterlog.Good("Build order:")
	for _, orderIdx := range order {
		fmt.Printf(" %s", srcPkgs[orderIdx].Name)
	}
	fmt.Println()
}
