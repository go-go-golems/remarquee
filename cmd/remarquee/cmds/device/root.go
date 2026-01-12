package device

import "github.com/spf13/cobra"

func NewDeviceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "device",
		Short: "Device-side capture and streaming tools",
	}

	cmd.AddCommand(NewServeCommand())
	cmd.AddCommand(NewInfoCommand())
	cmd.AddCommand(NewScreenshotCommand())
	cmd.AddCommand(NewRawCommand())
	return cmd
}
