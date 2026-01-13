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
	cmd.AddCommand(NewStreamCommand())
	cmd.AddCommand(NewEventsCommand())
	cmd.AddCommand(NewGesturesCommand())
	return cmd
}
