//go:build cgo

package main

import (
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/go-go-golems/remarquee/pkg/doc"
	"github.com/spf13/cobra"
)

func setupHelpSystem(rootCmd *cobra.Command) {
	helpSystem := help.NewHelpSystem()
	if err := doc.AddDocToHelpSystem(helpSystem); err == nil {
		help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)
		return
	}
	// If docs fail to load, still run the CLI (Cobra --help remains usable).
	// The error will show up when users try to run `remarquee help`.
	help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)
}
