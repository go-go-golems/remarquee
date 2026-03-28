package rmdoc

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/remarquee/pkg/rmcloud"
	"github.com/pkg/errors"
)

type CloudInputSettings struct {
	Cloud          bool `glazed.parameter:"cloud"`
	NonInteractive bool `glazed.parameter:"non-interactive"`
	Reauth         bool `glazed.parameter:"reauth"`
}

type ResolvedRMDocInput struct {
	RequestedPath string
	LocalPath     string
	Source        string
	Cleanup       func() error
}

var downloadDocumentByPath = rmcloud.DownloadDocumentByPath

func cloudInputParameterDefinitions() []*parameters.ParameterDefinition {
	return []*parameters.ParameterDefinition{
		parameters.NewParameterDefinition(
			"cloud",
			parameters.ParameterTypeBool,
			parameters.WithDefault(false),
			parameters.WithHelp("Treat the input path as a remote reMarkable cloud path and download it before rendering"),
		),
		parameters.NewParameterDefinition(
			"non-interactive",
			parameters.ParameterTypeBool,
			parameters.WithDefault(false),
			parameters.WithHelp("Do not prompt for one-time code; fail if tokens are missing"),
		),
		parameters.NewParameterDefinition(
			"reauth",
			parameters.ParameterTypeBool,
			parameters.WithDefault(false),
			parameters.WithHelp("Force re-authentication (re-fetch user token)"),
		),
	}
}

func initializeCloudInputSettings(parsedLayers *layers.ParsedLayers, target *CloudInputSettings) error {
	if target == nil {
		return errors.New("cloud input settings target is nil")
	}
	return parsedLayers.InitializeStruct(layers.DefaultSlug, target)
}

func ResolveRMDocInput(ctx context.Context, file string, s CloudInputSettings) (*ResolvedRMDocInput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	file = strings.TrimSpace(file)
	if file == "" {
		return nil, errors.New("file is empty")
	}

	if !s.Cloud {
		return &ResolvedRMDocInput{
			RequestedPath: file,
			LocalPath:     file,
			Source:        "local",
			Cleanup:       func() error { return nil },
		}, nil
	}

	tmpDir, err := os.MkdirTemp("", "remarquee-rmdoc-cloud-*")
	if err != nil {
		return nil, errors.Wrap(err, "create temp dir")
	}

	downloaded, err := downloadDocumentByPath(ctx, rmcloud.AuthSettings{
		NonInteractive: s.NonInteractive,
		Reauth:         s.Reauth,
	}, file, tmpDir)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, err
	}

	return &ResolvedRMDocInput{
		RequestedPath: file,
		LocalPath:     downloaded.LocalPath,
		Source:        "cloud",
		Cleanup: func() error {
			return os.RemoveAll(tmpDir)
		},
	}, nil
}

func defaultOutputPath(inputPath, suffix string) string {
	base := filepath.Base(inputPath)
	ext := filepath.Ext(base)
	if ext != "" {
		base = base[:len(base)-len(ext)]
	}
	return base + suffix
}

func ensureOutputWritable(path string, force bool) error {
	if force {
		return nil
	}
	if _, err := os.Stat(path); err == nil {
		return errors.Errorf("output file exists: %s (use --force to overwrite)", path)
	}
	return nil
}
