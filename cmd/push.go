// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"os"

	"github.com/DataDrake/waterlog"
	"github.com/GZGavinZhao/autobuild/common"
	"github.com/GZGavinZhao/autobuild/state"
	"github.com/GZGavinZhao/autobuild/utils"
	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
	"github.com/spf13/cobra"
)

var (
	cmdPush = &cobra.Command{
		Use:   "push <[src|bin]:path-to-old> <[src|bin]:path-to-new>",
		Short: "Push package changes to the build server",
		Run:   runPush,
		Args:  cobra.ExactArgs(2),
	}
)

func init() {
	cmdPush.Flags().BoolP("force", "f", false, "whether to ignore safety checks")
}

func runPush(cmd *cobra.Command, args []string) {

	oldTPath := args[0]
	newTPath := args[1]

	var oldState, newState state.State

	oldState, err := state.LoadState(oldTPath)
	if err != nil {
		waterlog.Fatalf("Failed to load old state %s: %s\n", oldTPath, err)
	}
	waterlog.Goodln("Successfully parsed old state!")

	newState, err = state.LoadState(newTPath)
	if err != nil {
		waterlog.Fatalf("Failed to load new state %s: %s\n", newTPath, err)
	}
	waterlog.Goodln("Successfully parsed new state!")

	waterlog.Infoln("Diffing...")
	changes := state.Changed(&oldState, &newState)

	bumped := []common.Package{}
	bset := make(map[int]bool)
	outdated := []common.Package{}
	bad := []common.Package{}

	for _, diff := range changes {
		pkg := newState.Packages()[diff.Idx]
		if diff.IsNewRel() {
			bumped = append(bumped, pkg)
			bset[diff.Idx] = true
		} else if diff.IsSameRel() && !diff.IsSame() {
			bad = append(bad, pkg)
		} else if diff.IsDowngrade() {
			outdated = append(outdated, pkg)
		}
	}

	force, _ := cmd.Flags().GetBool("force")

	if len(bad) != 0 && !force {
		waterlog.Warnf("The following packages have the same release number but different version:")
		for _, pkg := range bad {
			waterlog.Printf(" %s", pkg.Name)
		}
		waterlog.Fatalln()
	}

	if len(outdated) != 0 {
		waterlog.Warnf("The following packages have older release numbers:")
		for _, pkg := range outdated {
			waterlog.Printf(" %s", pkg.Name)
		}
		waterlog.Println()
	}

	if len(bumped) == 0 {
		waterlog.Infoln("No packages to update. Exiting...")
		return
	}

	waterlog.Goodf("The following packages will be updated:")
	for _, pkg := range bumped {
		waterlog.Printf(" %s", pkg.Name)
	}
	waterlog.Println()

	depGraph := newState.DepGraph()
	waterlog.Goodln("Successfully generated dependency graph!")

	lifted, err := utils.LiftGraph(depGraph, func(i int) bool { return bset[i] })
	if err != nil {
		waterlog.Fatalf("Failed to lift updated packages from dependency graph: %s\n", err)
	}
	waterlog.Goodln("Successfully isolated packages to update!")

	order, err := graph.TopologicalSort(lifted)
	if err != nil {
		fingDot, _ := os.Create("lifted.gv")
		_ = draw.DOT(lifted, fingDot)

		if cycles, err := graph.StronglyConnectedComponents(lifted); err == nil {
			cycleIdx := 0

			for _, cycle := range cycles {
				if len(cycle) <= 1 {
					continue
				}

				waterlog.Debugf("Cycle %d:", cycleIdx+1)
				cycleIdx++

				for _, nodeIdx := range cycle {
					waterlog.Printf(" %s", newState.Packages()[nodeIdx].Name)
				}
				waterlog.Println()
			}
		}

		waterlog.Fatalf("Failed to compute build order: %s\n", err)
	}

	waterlog.Goodln("Here's the build order:")
	for _, idx := range order {
		waterlog.Println(newState.Packages()[idx].Name)
	}
}
