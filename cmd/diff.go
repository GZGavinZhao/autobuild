// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"github.com/DataDrake/waterlog"
	"github.com/GZGavinZhao/autobuild/state"
	"github.com/spf13/cobra"
)

var (
	strictDiff bool

	cmdDiff = &cobra.Command{
		Use:   "diff <[src|bin|repo]:path-to-old> <[src|bin|repo]:path-to-new>",
		Short: "Diff the packages between binary indices or sources or a mix of them",
		Run:   runDiff,
		Args:  cobra.ExactArgs(2),
	}
)

func init() {
	cmdDiff.Flags().BoolVarP(&strictDiff, "strict", "s", false, "show and warn suspicious changes such as outdated packages or unbumped relnos")
}

func runDiff(cmd *cobra.Command, args []string) {
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
	for _, diff := range state.Changed(&oldState, &newState) {
		name := newState.Packages()[diff.Idx].Source

		if diff.OldRelNum == 0 {
			waterlog.Infof("New: %s: %s-%d\n", name, diff.Ver, diff.RelNum)
		} else if diff.RelNum > diff.OldRelNum {
			waterlog.Infof("Rebuild/Change: %s: %s-%d -> %s-%d\n", name, diff.OldVer, diff.OldRelNum, diff.Ver, diff.RelNum)
		} else if diff.RelNum < diff.OldRelNum {
			if strictDiff {
				waterlog.Warnf("Outdated: %s: %s-%d <- %s-%d\n", name, diff.OldVer, diff.OldRelNum, diff.Ver, diff.RelNum)
			}
		} else if diff.Ver != diff.OldVer {
			if strictDiff {
				waterlog.Warnf("Different version but same relno: %s: %s-%d -> %s-%d\n", name, diff.OldVer, diff.OldRelNum, diff.Ver, diff.RelNum)
			}
		}
	}
}
