package main

import (
	"context"
	"errors"
	"os"

	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/remarquee/cmd/remarquee/cmds"
	"github.com/go-go-golems/remarquee/cmd/remarquee/cmds/cloud"
	device_cmd "github.com/go-go-golems/remarquee/cmd/remarquee/cmds/device"
	ocr_cmd "github.com/go-go-golems/remarquee/cmd/remarquee/cmds/ocr"
	rmdoc_cmd "github.com/go-go-golems/remarquee/cmd/remarquee/cmds/rmdoc"
	rmdsl_cmd "github.com/go-go-golems/remarquee/cmd/remarquee/cmds/rmdsl"
	serve_cmd "github.com/go-go-golems/remarquee/cmd/remarquee/cmds/serve"
	"github.com/go-go-golems/remarquee/cmd/remarquee/cmds/upload"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:          "remarquee",
	Short:        "remarquee is a unified toolkit for reMarkable workflows",
	SilenceUsage: true,
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
	if serveCobraCmd, err := serve_cmd.NewServeCobraCommand(); err != nil {
		rootCmd.AddCommand(&cobra.Command{
			Use:   "serve",
			Short: "Serve a local web UI for reMarkable files (unavailable due to init error)",
			RunE: func(cmd *cobra.Command, args []string) error {
				return err
			},
		})
	} else {
		rootCmd.AddCommand(serveCobraCmd)
	}
	rootCmd.AddCommand(upload.NewUploadCommand())

	// ExecuteContext propagates SIGINT cancellation to the verbs. The wrapper
	// also forces exit on a second interrupt or after two seconds: rmapi's
	// AuthHttpCtx does not accept context.Context, so authentication can remain
	// blocked even after the command context is canceled (see runInterruptible).
	if err := executeInterruptible(rootCmd.ExecuteContext); err != nil {
		if errors.Is(err, context.Canceled) {
			os.Exit(130)
		}
		os.Exit(1)
	}
}
