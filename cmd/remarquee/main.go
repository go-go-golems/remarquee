package main

import (
	"os"

	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/remarquee/cmd/remarquee/cmds"
	"github.com/go-go-golems/remarquee/cmd/remarquee/cmds/cloud"
	device_cmd "github.com/go-go-golems/remarquee/cmd/remarquee/cmds/device"
	ocr_cmd "github.com/go-go-golems/remarquee/cmd/remarquee/cmds/ocr"
	rmdoc_cmd "github.com/go-go-golems/remarquee/cmd/remarquee/cmds/rmdoc"
	rmdsl_cmd "github.com/go-go-golems/remarquee/cmd/remarquee/cmds/rmdsl"
	"github.com/go-go-golems/remarquee/cmd/remarquee/cmds/upload"
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
	_ = logging.AddLoggingSectionToRootCommand(rootCmd, "remarquee")

	setupHelpSystem(rootCmd)

	rootCmd.AddCommand(cmds.NewStatusCommand())
	rootCmd.AddCommand(cloud.NewCloudCommand())
	rootCmd.AddCommand(device_cmd.NewDeviceCommand())
	rootCmd.AddCommand(ocr_cmd.NewOCRCommand())
	rootCmd.AddCommand(rmdsl_cmd.NewRmdslCommand())
	rootCmd.AddCommand(rmdoc_cmd.NewRmdocCommand())
	rootCmd.AddCommand(upload.NewUploadCommand())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
