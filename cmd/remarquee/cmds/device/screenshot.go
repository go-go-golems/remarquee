package device

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type screenshotSettings struct {
	BaseURL  string
	Username string
	Password string
	Insecure bool
	OutPath  string
}

func NewScreenshotCommand() *cobra.Command {
	s := &screenshotSettings{}
	cmd := &cobra.Command{
		Use:   "screenshot",
		Short: "Fetch a PNG screenshot from the device server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScreenshot(cmd, s)
		},
	}
	cmd.Flags().StringVar(&s.BaseURL, "url", "http://remarkable.local:2718", "Base URL for device capture server")
	cmd.Flags().StringVar(&s.Username, "username", "admin", "Basic auth username")
	cmd.Flags().StringVar(&s.Password, "password", "password", "Basic auth password")
	cmd.Flags().BoolVar(&s.Insecure, "insecure", false, "Skip TLS verification")
	cmd.Flags().StringVar(&s.OutPath, "out", "screenshot.png", "Output PNG path")
	return cmd
}

func runScreenshot(cmd *cobra.Command, s *screenshotSettings) error {
	client := newHTTPClient(&clientSettings{Insecure: s.Insecure, Timeout: 30 * time.Second})
	url, err := buildURL(s.BaseURL, "/api/v1/screenshot.png")
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
	defer resp.Body.Close()
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
	defer out.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}
	_, _ = cmd.OutOrStdout().Write([]byte("OK: wrote " + s.OutPath + "\n"))
	return nil
}
