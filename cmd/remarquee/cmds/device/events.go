package device

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	perrors "github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type eventsSettings struct {
	BaseURL  string
	Username string
	Password string
	Insecure bool
	OutPath  string
	Duration time.Duration
}

func NewEventsCommand() *cobra.Command {
	s := &eventsSettings{}
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Stream pen events from the device server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEvents(cmd, s)
		},
	}
	cmd.Flags().StringVar(&s.BaseURL, "url", "http://remarkable.local:2718", "Base URL for device capture server")
	cmd.Flags().StringVar(&s.Username, "username", "admin", "Basic auth username")
	cmd.Flags().StringVar(&s.Password, "password", "password", "Basic auth password")
	cmd.Flags().BoolVar(&s.Insecure, "insecure", false, "Skip TLS verification")
	cmd.Flags().StringVar(&s.OutPath, "out", "events.sse", "Output file (use - for stdout)")
	cmd.Flags().DurationVar(&s.Duration, "duration", 5*time.Second, "Stream duration (0 = until interrupted)")
	return cmd
}

func runEvents(cmd *cobra.Command, s *eventsSettings) error {
	ctx := context.Background()
	if s.Duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.Duration)
		defer cancel()
	}

	timeout := s.Duration + 5*time.Second
	if s.Duration == 0 {
		timeout = 0
	}
	client := newHTTPClient(&clientSettings{Insecure: s.Insecure, Timeout: timeout})
	url, err := buildURL(s.BaseURL, "/api/v1/events")
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	applyBasicAuth(req, &clientSettings{Username: s.Username, Password: s.Password})
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return perrors.Errorf("request failed: %s (%s)", resp.Status, string(body))
	}

	writer, closeFn, err := openOutputWriter(cmd, s.OutPath)
	if err != nil {
		return err
	}
	defer closeFn()

	_, err = io.Copy(writer, resp.Body)
	if err != nil && ctx.Err() != nil {
		return nil
	}
	return err
}

func openOutputWriter(cmd *cobra.Command, path string) (io.Writer, func(), error) {
	if path == "-" {
		return cmd.OutOrStdout(), func() {}, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, nil, perrors.Wrap(err, "failed to create output dir")
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}
	return f, func() { _ = f.Close() }, nil
}
