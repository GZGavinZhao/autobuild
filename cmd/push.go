// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"github.com/DataDrake/waterlog"
	"github.com/GZGavinZhao/autobuild/common"
	"github.com/GZGavinZhao/autobuild/state"
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
	outdated := []common.Package{}
	bad := []common.Package{}

	for _, diff := range changes {
		pkg := newState.Packages()[diff.Idx]
		if diff.IsNewRel() {
			bumped = append(bumped, pkg)
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

	waterlog.Infof("The following packages will be updated:")
	for _, pkg := range bumped {
		waterlog.Printf(" %s", pkg.Name)
	}
	waterlog.Println()
}
