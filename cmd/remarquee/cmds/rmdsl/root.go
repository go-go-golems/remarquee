package rmdsl

import "github.com/spf13/cobra"

func NewRmdslCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rmdsl",
		Short: "Compile and inspect RMDoc-DSL cases",
	}

	compileCmd, err := NewCompileCobraCommand()
	if err != nil {
		cmd.AddCommand(&cobra.Command{
			Use:   "compile",
			Short: "Compile RMDoc-DSL to .rmdoc (unavailable due to init error)",
			RunE: func(cmd *cobra.Command, args []string) error {
				return err
			},
		})
	} else {
		cmd.AddCommand(compileCmd)
	}

	return cmd
}
