package cmd

import (
	"github.com/DataDrake/waterlog"
	"github.com/DataDrake/waterlog/format"
	"github.com/spf13/cobra"
)

var (
	Verbose bool

	rootCmd = &cobra.Command{
		Use:   "autobuild",
		Short: "Automatically query, build, and push packages elegantly.",
	}
)

func init() {
	waterlog.SetFormat(format.Min)
	rootCmd.AddCommand(cmdQuery)
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Verbose output")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		waterlog.Fatalf("autobuild failed: %s\n", err)
	}
}
