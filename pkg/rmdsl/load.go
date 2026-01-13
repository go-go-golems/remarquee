package rmdsl

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type LoadOptions struct {
	// CaseRoot restricts rm.include() for JS to this directory.
	// If empty, defaults to the directory containing the entry JS file.
	CaseRoot string

	// Params are passed to main(params) for JS generators.
	Params map[string]any
}

func LoadFromFile(ctx context.Context, path string, opts LoadOptions) (*Doc, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		return loadYAML(path)
	case ".js":
		return loadJS(ctx, path, opts)
	default:
		return nil, errors.Errorf("unsupported case file extension %q (supported: .yaml/.yml/.js)", ext)
	}
}

func loadYAML(path string) (*Doc, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "read yaml")
	}
	var d Doc
	if err := yaml.Unmarshal(b, &d); err != nil {
		return nil, errors.Wrap(err, "unmarshal yaml")
	}
	if err := Normalize(&d); err != nil {
		return nil, errors.Wrap(err, "normalize")
	}
	return &d, nil
}
