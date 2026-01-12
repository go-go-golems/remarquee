//go:build !cgo

package main

import "github.com/spf13/cobra"

func setupHelpSystem(_ *cobra.Command) {
	// No-op when cgo is disabled; fall back to Cobra's default help.
}
