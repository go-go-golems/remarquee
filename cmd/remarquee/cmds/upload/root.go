package upload

import "github.com/spf13/cobra"

func NewUploadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload content to a reMarkable device",
	}

	cmd.AddCommand(NewUploadMarkdownCommand())
	cmd.AddCommand(NewUploadBundleCommand())
	return cmd
}
