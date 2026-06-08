package cmd

import (
	"fmt"
	"os"

	"plates/internal/shell"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "plates",
	Short: "PLATES renders command templates for manual execution",
	Long:  "PLATES is a local command-template rendering framework. Phase 1 provides the core shell, variable storage, and plate discovery.",
	RunE: func(cmd *cobra.Command, args []string) error {
		sh, err := shell.New(shell.Options{})
		if err != nil {
			return err
		}
		return sh.Run()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
