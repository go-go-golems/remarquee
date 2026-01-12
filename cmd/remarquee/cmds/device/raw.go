package device

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type rawSettings struct {
	BaseURL  string
	Username string
	Password string
	Insecure bool
	OutPath  string
}

func NewRawCommand() *cobra.Command {
	s := &rawSettings{}
	cmd := &cobra.Command{
		Use:   "raw",
		Short: "Fetch a raw framebuffer capture from the device server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRaw(cmd, s)
		},
	}
	cmd.Flags().StringVar(&s.BaseURL, "url", "http://remarkable.local:2718", "Base URL for device capture server")
	cmd.Flags().StringVar(&s.Username, "username", "admin", "Basic auth username")
	cmd.Flags().StringVar(&s.Password, "password", "password", "Basic auth password")
	cmd.Flags().BoolVar(&s.Insecure, "insecure", false, "Skip TLS verification")
	cmd.Flags().StringVar(&s.OutPath, "out", "screenshot.raw", "Output raw path")
	return cmd
}

func runRaw(cmd *cobra.Command, s *rawSettings) error {
	client := newHTTPClient(&clientSettings{Insecure: s.Insecure, Timeout: 30 * time.Second})
	url, err := buildURL(s.BaseURL, "/api/v1/screenshot.raw")
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	applyBasicAuth(req, &clientSettings{Username: s.Username, Password: s.Password})
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: close response body: %v\n", err)
		}
	}()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return errors.Errorf("request failed: %s (%s)", resp.Status, string(body))
	}

	if err := os.MkdirAll(filepath.Dir(s.OutPath), 0o755); err != nil {
		return errors.Wrap(err, "failed to create output dir")
	}
	out, err := os.Create(s.OutPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := out.Close(); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: close output: %v\n", err)
		}
	}()
	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}
	_, _ = cmd.OutOrStdout().Write([]byte("OK: wrote " + s.OutPath + "\n"))
	return nil
}
