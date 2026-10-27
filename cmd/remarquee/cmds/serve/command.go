package serve

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-go-golems/glazed/pkg/cli"
	glazecmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/remarquee/cmd/remarquee/internal/appconfig"
	"github.com/go-go-golems/remarquee/pkg/rmcloud"
	"github.com/go-go-golems/remarquee/pkg/rmfiles"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// ServeCommand wraps the glazed command description for `remarquee serve`.
type ServeCommand struct {
	*glazecmds.CommandDescription
}

// ServeSettings holds the `remarquee serve` flags.
type ServeSettings struct {
	Addr             string `glazed:"addr"`
	Dev              bool   `glazed:"dev"`
	DefaultRemoteDir string `glazed:"remote-dir"`
	IncludeTemplates bool   `glazed:"include-templates"`
	Workers          int    `glazed:"workers"`
}

var _ glazecmds.BareCommand = &ServeCommand{}

// NewServeCommand builds the glazed `serve` command description.
func NewServeCommand() (*ServeCommand, error) {
	commandSettingsLayer, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	cmdDesc := glazecmds.NewCommandDescription(
		"serve",
		glazecmds.WithShort("Serve a local web UI to upload, manage, and search reMarkable files"),
		glazecmds.WithLong(strings.TrimSpace(`
Serve a local, loopback-only web UI for your reMarkable cloud files.

The UI lets you upload Markdown/PDF/EPUB files, browse and search the cloud
tree, create folders, rename/move entries, and delete entries. It runs in the
existing remarquee binary and never exposes your credentials to the browser.

Examples:
  remarquee serve
  remarquee serve --addr 127.0.0.1:9090
  remarquee serve --remote-dir /ai
  remarquee serve --include-templates --workers 4
`)),
		glazecmds.WithFlags(
			fields.New(
				"addr",
				fields.TypeString,
				fields.WithDefault("127.0.0.1:8080"),
				fields.WithHelp("HTTP listen address (loopback by default)"),
			),
			fields.New(
				"dev",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Disable static asset caching while editing the frontend"),
			),
			fields.New(
				"remote-dir",
				fields.TypeString,
				fields.WithDefault("/"),
				fields.WithHelp("Default upload destination directory"),
			),
			fields.New(
				"include-templates",
				fields.TypeBool,
				fields.WithDefault(false),
				fields.WithHelp("Show template documents (hidden by default)"),
			),
			fields.New(
				"workers",
				fields.TypeInteger,
				fields.WithDefault(2),
				fields.WithHelp("Concurrent conversions per upload job"),
			),
		),
		glazecmds.WithSections(commandSettingsLayer),
	)

	return &ServeCommand{CommandDescription: cmdDesc}, nil
}

// Run executes `remarquee serve` with the parsed flag values.
func (c *ServeCommand) Run(ctx context.Context, parsedValues *values.Values) error {
	s := &ServeSettings{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
		return err
	}
	return runServe(ctx, s)
}

// NewServeCobraCommand builds the cobra command for `remarquee serve`.
func NewServeCobraCommand() (*cobra.Command, error) {
	cmd, err := NewServeCommand()
	if err != nil {
		return nil, err
	}

	return cli.BuildCobraCommand(cmd,
		cli.WithParserConfig(appconfig.DefaultParserConfig()),
	)
}

func runServe(ctx context.Context, s *ServeSettings) error {
	log.Info().Str("remote_dir", s.DefaultRemoteDir).Msg("remarquee serve: initializing cloud context (first run may take a while)")

	svc, err := rmfiles.NewServiceFromCloud(ctx, rmfiles.Config{
		Auth:             rmcloud.AuthSettings{NonInteractive: true, Progress: os.Stderr},
		DefaultRemoteDir: s.DefaultRemoteDir,
		IncludeTemplates: s.IncludeTemplates,
		Workers:          s.Workers,
	})
	if err != nil {
		return err
	}

	srv := NewServer(svc, Options{Dev: s.Dev})
	httpSrv := &http.Server{
		Addr:              s.Addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       120 * time.Second,
		// Downloads and (synchronous) request bodies can be large, so no write
		// timeout is enforced; uploads run as background jobs.
		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
	}()

	log.Info().Str("url", "http://"+s.Addr).Msg("remarquee serve listening")

	if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
