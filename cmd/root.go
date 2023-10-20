// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"github.com/DataDrake/waterlog"
	"github.com/DataDrake/waterlog/format"
	"github.com/spf13/cobra"
)

var (
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
	}
)

func init() {
	rootCmd.AddCommand(cmdQuery)

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
