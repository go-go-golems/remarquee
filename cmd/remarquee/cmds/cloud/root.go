package cloud

import (
	"github.com/spf13/cobra"
)

func NewCloudCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cloud",
		Short: "Interact with the reMarkable cloud (rmapi-backed)",
	}

	accountCmd, err := NewAccountCobraCommand()
	if err != nil {
		cmd.AddCommand(&cobra.Command{
			Use:   "account",
			Short: "Show cloud account info (unavailable due to init error)",
			RunE: func(cmd *cobra.Command, args []string) error {
				return err
			},
		})
	} else {
		cmd.AddCommand(accountCmd)
	}

	getCmd, err := NewGetCobraCommand()
	if err != nil {
		cmd.AddCommand(&cobra.Command{
			Use:   "get",
			Short: "Download a remote document as .rmdoc (unavailable due to init error)",
			RunE: func(cmd *cobra.Command, args []string) error {
				return err
			},
		})
	} else {
		cmd.AddCommand(getCmd)
	}

	findCmd, err := NewFindCobraCommand()
	if err != nil {
		cmd.AddCommand(&cobra.Command{
			Use:   "find",
			Short: "Find entries recursively (unavailable due to init error)",
			RunE: func(cmd *cobra.Command, args []string) error {
				return err
			},
		})
	} else {
		cmd.AddCommand(findCmd)
	}

	lsCmd, err := NewLsCobraCommand()
	if err != nil {
		cmd.AddCommand(&cobra.Command{
			Use:   "ls",
			Short: "List remote directory entries (unavailable due to init error)",
			RunE: func(cmd *cobra.Command, args []string) error {
				return err
			},
		})
	} else {
		cmd.AddCommand(lsCmd)
	}

	mvCmd, err := NewMvCobraCommand()
	if err != nil {
		cmd.AddCommand(&cobra.Command{
			Use:   "mv",
			Short: "Move or rename a remote entry (unavailable due to init error)",
			RunE: func(cmd *cobra.Command, args []string) error {
				return err
			},
		})
	} else {
		cmd.AddCommand(mvCmd)
	}

	mkdirCmd, err := NewMkdirCobraCommand()
	if err != nil {
		cmd.AddCommand(&cobra.Command{
			Use:   "mkdir",
			Short: "Create a remote directory (unavailable due to init error)",
			RunE: func(cmd *cobra.Command, args []string) error {
				return err
			},
		})
	} else {
		cmd.AddCommand(mkdirCmd)
	}

	rmCmd, err := NewRmCobraCommand()
	if err != nil {
		cmd.AddCommand(&cobra.Command{
			Use:   "rm",
			Short: "Delete remote entries (unavailable due to init error)",
			RunE: func(cmd *cobra.Command, args []string) error {
				return err
			},
		})
	} else {
		cmd.AddCommand(rmCmd)
	}

	putCmd, err := NewPutCobraCommand()
	if err != nil {
		cmd.AddCommand(&cobra.Command{
			Use:   "put",
			Short: "Upload a local document to the cloud (unavailable due to init error)",
			RunE: func(cmd *cobra.Command, args []string) error {
				return err
			},
		})
	} else {
		cmd.AddCommand(putCmd)
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

	statCmd, err := NewStatCobraCommand()
	if err != nil {
		cmd.AddCommand(&cobra.Command{
			Use:   "stat",
			Short: "Show metadata for a remote entry (unavailable due to init error)",
			RunE: func(cmd *cobra.Command, args []string) error {
				return err
			},
		})
	} else {
		cmd.AddCommand(statCmd)
	}

	versionCmd, err := NewVersionCobraCommand()
	if err != nil {
		cmd.AddCommand(&cobra.Command{
			Use:   "version",
			Short: "Show rmapi version (unavailable due to init error)",
			RunE: func(cmd *cobra.Command, args []string) error {
				return err
			},
		})
	} else {
		cmd.AddCommand(versionCmd)
	}

	return cmd
}
