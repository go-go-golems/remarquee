package cloud

import (
	"github.com/spf13/cobra"
)

func NewCloudCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cloud",
		Short: "Interact with the reMarkable cloud (rmapi-backed)",
	}

	refreshCmd, err := NewRefreshCobraCommand()
	if err != nil {
		// Keep wiring simple: if we can’t construct a subcommand, expose the error at runtime.
		cmd.AddCommand(&cobra.Command{
			Use:   "refresh",
			Short: "Refresh the remote document tree (unavailable due to init error)",
			RunE: func(cmd *cobra.Command, args []string) error {
				return err
			},
		})
	} else {
		cmd.AddCommand(refreshCmd)
	}

	return cmd
}
