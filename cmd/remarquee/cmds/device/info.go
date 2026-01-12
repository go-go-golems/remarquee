package device

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type infoSettings struct {
	BaseURL  string
	Username string
	Password string
	Insecure bool
}

func NewInfoCommand() *cobra.Command {
	s := &infoSettings{}
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Fetch device capture info",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInfo(cmd, s)
		},
	}
	cmd.Flags().StringVar(&s.BaseURL, "url", "http://remarkable.local:2718", "Base URL for device capture server")
	cmd.Flags().StringVar(&s.Username, "username", "admin", "Basic auth username")
	cmd.Flags().StringVar(&s.Password, "password", "password", "Basic auth password")
	cmd.Flags().BoolVar(&s.Insecure, "insecure", false, "Skip TLS verification")
	return cmd
}

func runInfo(cmd *cobra.Command, s *infoSettings) error {
	client := newHTTPClient(&clientSettings{Insecure: s.Insecure, Timeout: 15 * time.Second})
	url, err := buildURL(s.BaseURL, "/api/v1/info")
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
	var out any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
