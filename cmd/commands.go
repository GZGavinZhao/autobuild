package cmd

import (
	"github.com/DataDrake/waterlog"
	"github.com/DataDrake/waterlog/format"
	"github.com/spf13/cobra"
)

var (
	quiet       bool
	verbose     bool
	sourcesPath string
	indexPath   string

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

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Quiet output")
	rootCmd.MarkFlagsMutuallyExclusive("verbose", "quiet")
}

func pathsInit(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&sourcesPath, "sources", "s", "", "Path to the source directory (containing Taskfile.yml)")
	cmd.MarkFlagRequired("sources")
	cmd.Flags().StringVarP(&indexPath, "index", "i", "", "Path to the eopkg binary index to compare against")
	cmd.MarkFlagRequired("index")
}

func Execute() {
	rootCmd.Execute()
	// if err := rootCmd.Execute(); err != nil {
	// 	waterlog.Fatalf("autobuild failed: %s\n", err)
	// }
}
