// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"strings"

	"github.com/DataDrake/waterlog"
	"github.com/GZGavinZhao/autobuild/state"
	"github.com/spf13/cobra"
)

var (
	cmdDiff = &cobra.Command{
		Use:   "diff <[src|bin]:path-to-old> <[src|bin]:path-to-new>",
		Short: "Diff the packages between binary indices or sources or a mix of them",
		Run:   runDiff,
		Args:  cobra.ExactArgs(2),
	}
)

func init() {
}

func runDiff(cmd *cobra.Command, args []string) {
	olds := strings.Split(args[0], ":")
	news := strings.Split(args[1], ":")

	var oldState, newState state.State
	var err error

	if olds[0] == "src" {
		oldState, err = state.LoadSource(olds[1])

		if err != nil {
			waterlog.Fatalf("Failed to load old source: %s\n", err)
		}
	} else if olds[0] == "bin" {
		oldState, err = state.LoadBinary(olds[1])

		if err != nil {
			waterlog.Fatalf("Failed to load old binary index: %s\n", err)
		}
	}
	waterlog.Goodln("Successfully parsed old state!")

	if news[0] == "src" {
		newState, err = state.LoadSource(news[1])

		if err != nil {
			waterlog.Fatalf("Failed to load old source: %s\n", err)
		}
	} else if news[0] == "bin" {
		newState, err = state.LoadBinary(news[1])

		if err != nil {
			waterlog.Fatalf("Failed to load old binary index: %s\n", err)
		}
	}
	waterlog.Goodln("Successfully parsed new state!")

	waterlog.Infoln("Diffing...")
	for _, diff := range state.Changed(&oldState, &newState) {
		name := newState.Packages()[diff.Idx].Name

		if diff.OldRelNum == 0 {
			waterlog.Infof("New: %s: %s-%d\n", name, diff.Ver, diff.RelNum)
		} else if diff.Ver != diff.OldVer {
			waterlog.Infof("Update: %s: %s-%d -> %s-%d\n", name, diff.OldVer, diff.OldRelNum, diff.Ver, diff.RelNum)
		} else if diff.RelNum > diff.OldRelNum {
			waterlog.Infof("Rebuild/Change: %s: %s-%d -> %s-%d\n", name, diff.OldVer, diff.OldRelNum, diff.Ver, diff.RelNum)
		}
	}
}
