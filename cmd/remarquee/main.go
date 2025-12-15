package main

import (
	"os"

	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/remarquee/cmd/remarquee/cmds"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "remarquee",
	Short: "remarquee is a unified toolkit for reMarkable workflows",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize logger after Cobra has parsed flags.
		return logging.InitLoggerFromCobra(cmd)
	},
}

func main() {
	_ = logging.AddLoggingLayerToRootCommand(rootCmd, "remarquee")

	rootCmd.AddCommand(cmds.NewStatusCommand())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
