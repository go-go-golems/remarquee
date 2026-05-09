package device

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type clientSettings struct {
	BaseURL  string
	Username string
	Password string
	Insecure bool
	Timeout  time.Duration
}

func newHTTPClient(s *clientSettings) *http.Client {
	if s == nil || !s.Insecure {
		return &http.Client{Timeout: s.Timeout}
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402 -- explicitly requested insecure mode for self-signed device connections.
	}
	return &http.Client{
		Timeout:   s.Timeout,
		Transport: transport,
	}
}

func buildURL(base, suffix string) (string, error) {
	if strings.TrimSpace(base) == "" {
		return "", fmt.Errorf("missing base url")
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(strings.TrimRight(u.Path, "/"), suffix)
	return u.String(), nil
}

func applyBasicAuth(req *http.Request, s *clientSettings) {
	if s == nil {
		return
	}
	if s.Username == "" && s.Password == "" {
		return
	}
	req.SetBasicAuth(s.Username, s.Password)
}
