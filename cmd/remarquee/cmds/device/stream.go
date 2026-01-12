package device

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	perrors "github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type streamSettings struct {
	BaseURL  string
	Username string
	Password string
	Insecure bool
	OutPath  string
	RateMS   int
	Duration time.Duration
}

func NewStreamCommand() *cobra.Command {
	s := &streamSettings{}
	cmd := &cobra.Command{
		Use:   "stream",
		Short: "Stream raw framebuffer bytes to a file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStream(cmd, s)
		},
	}
	cmd.Flags().StringVar(&s.BaseURL, "url", "http://remarkable.local:2718", "Base URL for device capture server")
	cmd.Flags().StringVar(&s.Username, "username", "admin", "Basic auth username")
	cmd.Flags().StringVar(&s.Password, "password", "password", "Basic auth password")
	cmd.Flags().BoolVar(&s.Insecure, "insecure", false, "Skip TLS verification")
	cmd.Flags().StringVar(&s.OutPath, "out", "stream.raw", "Output raw stream path")
	cmd.Flags().IntVar(&s.RateMS, "rate", 200, "Frame rate in ms (server-side)")
	cmd.Flags().DurationVar(&s.Duration, "duration", 5*time.Second, "Stream duration (0 = until interrupted)")
	return cmd
}

func runStream(cmd *cobra.Command, s *streamSettings) error {
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
	url, err := buildURL(s.BaseURL, "/api/v1/stream")
	if err != nil {
		return err
	}
	if s.RateMS > 0 {
		url = fmt.Sprintf("%s?rate=%d", url, s.RateMS)
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

	if err := os.MkdirAll(filepath.Dir(s.OutPath), 0o755); err != nil {
		return perrors.Wrap(err, "failed to create output dir")
	}
	out, err := os.Create(s.OutPath)
	if err != nil {
		return err
	}
	defer out.Close()

	written, err := io.Copy(out, resp.Body)
	if err != nil && ctx.Err() != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(ctx.Err(), context.Canceled) {
			_, _ = cmd.OutOrStdout().Write([]byte(fmt.Sprintf("OK: wrote %s (%d bytes, timeout)\n", s.OutPath, written)))
			return nil
		}
		return err
	}
	_, _ = cmd.OutOrStdout().Write([]byte(fmt.Sprintf("OK: wrote %s (%d bytes)\n", s.OutPath, written)))
	return nil
}
