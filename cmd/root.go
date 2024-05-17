// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"runtime/debug"

	"github.com/DataDrake/waterlog"
	"github.com/DataDrake/waterlog/format"
	"github.com/spf13/cobra"
)

var (
	GitCommit = func() string {
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					return setting.Value
				}
			}
		}
		return ""
	}()

	rootCmd = &cobra.Command{
		Use:   "autobuild",
		Short: "Automatically query, build, and push packages elegantly.",
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			waterlog.SetFormat(format.Min)
			if quiet {
				waterlog.SetLevel(0)
			} else if verbose {
				waterlog.SetLevel(7)
			} else {
				waterlog.SetLevel(6)
			}
		},
		Version: "0.0.0+" + GitCommit,
	}
)

func init() {
	rootCmd.AddCommand(cmdQuery)
	rootCmd.AddCommand(cmdDiff)
	rootCmd.AddCommand(cmdPush)

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "quiet output")
	rootCmd.MarkFlagsMutuallyExclusive("verbose", "quiet")
}

func Execute() {
	rootCmd.Execute()
	// if err := rootCmd.Execute(); err != nil {
	// 	waterlog.Fatalf("autobuild failed: %s\n", err)
	// }
}
