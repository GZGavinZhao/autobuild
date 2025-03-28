// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"fmt"
	"log/slog"
	"os"

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
		slog.Error("Failed to load old state", "tpath", oldTPath, "err", err)
		os.Exit(1)
	}
	slog.Info("Successfully parsed old state!")

	newState, err = state.LoadState(newTPath)
	if err != nil {
		slog.Error("Failed to load new state", "tpath", newTPath, "err", err)
		os.Exit(1)
	}
	slog.Info("Successfully parsed new state!")

	slog.Info("Diffing...")
	changes, err := state.Changed(&oldState, &newState)
	if err != nil {
		slog.Error("Failed to diff between states", err, "error")
		os.Exit(1)
	}

	for _, diff := range changes {
		name := newState.Packages()[diff.Idx].Source

		// TODO: probably shouldn't use logging for this!
		if diff.OldRelNum == 0 {
			slog.Info(fmt.Sprintf("New: %s: %s-%d", name, diff.Ver, diff.RelNum))
		} else if diff.RelNum > diff.OldRelNum {
			slog.Info(fmt.Sprintf("Rebuild/Change: %s: %s-%d -> %s-%d", name, diff.OldVer, diff.OldRelNum, diff.Ver, diff.RelNum))
		} else if diff.RelNum < diff.OldRelNum {
			if strictDiff {
				slog.Warn(fmt.Sprintf("Outdated: %s: %s-%d <- %s-%d", name, diff.OldVer, diff.OldRelNum, diff.Ver, diff.RelNum))
			}
		} else if diff.Ver != diff.OldVer {
			if strictDiff {
				slog.Warn(fmt.Sprintf("Different version but same relno: %s: %s-%d -> %s-%d", name, diff.OldVer, diff.OldRelNum, diff.Ver, diff.RelNum))
			}
		}
	}
}
