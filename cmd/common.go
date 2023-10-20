// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package cmd

import "github.com/spf13/cobra"

var (
	quiet       bool
	verbose     bool
	sourcesPath string
	indexPath   string
)

func pathsInit(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&sourcesPath, "sources", "s", "", "path to the source directory (containing Taskfile.yml)")
	cmd.MarkFlagRequired("sources")
	cmd.Flags().StringVarP(&indexPath, "index", "i", "", "path to the eopkg binary index to compare against")
	cmd.MarkFlagRequired("index")
}
