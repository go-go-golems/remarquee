package cmds

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show basic remarquee status",
		RunE: func(cmd *cobra.Command, args []string) error {
			// First command is intentionally simple: it validates the CLI wiring.
			fmt.Fprintln(cmd.OutOrStdout(), "remarquee: ok")
			return nil
		},
	}

	return cmd
}
