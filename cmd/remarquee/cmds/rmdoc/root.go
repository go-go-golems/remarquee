package rmdoc

import "github.com/spf13/cobra"

func NewRmdocCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rmdoc",
		Short: "Inspect and render .rmdoc archives",
	}

	cmd.AddCommand(NewInspectCommand())
	return cmd
}
