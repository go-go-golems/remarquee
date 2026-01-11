package main

import (
	"os"

	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/go-go-golems/remarquee/cmd/remarquee/cmds"
	"github.com/go-go-golems/remarquee/cmd/remarquee/cmds/cloud"
	ocr_cmd "github.com/go-go-golems/remarquee/cmd/remarquee/cmds/ocr"
	rmdoc_cmd "github.com/go-go-golems/remarquee/cmd/remarquee/cmds/rmdoc"
	rmdsl_cmd "github.com/go-go-golems/remarquee/cmd/remarquee/cmds/rmdsl"
	"github.com/go-go-golems/remarquee/cmd/remarquee/cmds/upload"
	"github.com/go-go-golems/remarquee/pkg/doc"
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

	helpSystem := help.NewHelpSystem()
	if err := doc.AddDocToHelpSystem(helpSystem); err == nil {
		help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)
	} else {
		// If docs fail to load, still run the CLI (Cobra --help remains usable).
		// The error will show up when users try to run `remarquee help`.
		help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)
	}

	rootCmd.AddCommand(cmds.NewStatusCommand())
	rootCmd.AddCommand(cloud.NewCloudCommand())
	rootCmd.AddCommand(ocr_cmd.NewOCRCommand())
	rootCmd.AddCommand(rmdsl_cmd.NewRmdslCommand())
	rootCmd.AddCommand(rmdoc_cmd.NewRmdocCommand())
	rootCmd.AddCommand(upload.NewUploadCommand())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
