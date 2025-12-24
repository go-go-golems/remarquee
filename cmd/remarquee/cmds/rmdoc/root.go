package rmdoc

import "github.com/spf13/cobra"

func NewRmdocCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rmdoc",
		Short: "Inspect and render .rmdoc archives",
	}

	inspectCmd, err := NewInspectCobraCommand()
	if err != nil {
		cmd.AddCommand(&cobra.Command{
			Use:   "inspect",
			Short: "Inspect a local .rmdoc (unavailable due to init error)",
			RunE: func(cmd *cobra.Command, args []string) error {
				return err
			},
		})
	} else {
		cmd.AddCommand(inspectCmd)
	}

	bgCmd, err := NewBuildBackgroundCobraCommand()
	if err != nil {
		cmd.AddCommand(&cobra.Command{
			Use:   "build-background",
			Short: "Build a UI-ordered background PDF (unavailable due to init error)",
			RunE: func(cmd *cobra.Command, args []string) error {
				return err
			},
		})
	} else {
		cmd.AddCommand(bgCmd)
	}

	renderLegacyCmd, err := NewRenderLegacyCobraCommand()
	if err != nil {
		cmd.AddCommand(&cobra.Command{
			Use:   "render-legacy",
			Short: "Render a legacy (V3/V5) .rmdoc to PDF (unavailable due to init error)",
			RunE: func(cmd *cobra.Command, args []string) error {
				return err
			},
		})
	} else {
		cmd.AddCommand(renderLegacyCmd)
	}

	renderV6Cmd, err := NewRenderV6CobraCommand()
	if err != nil {
		cmd.AddCommand(&cobra.Command{
			Use:   "render-v6",
			Short: "Render a V6 (cPages) .rmdoc to PDF (unavailable due to init error)",
			RunE: func(cmd *cobra.Command, args []string) error {
				return err
			},
		})
	} else {
		cmd.AddCommand(renderV6Cmd)
	}
	return cmd
}
