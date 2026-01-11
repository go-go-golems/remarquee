package rmdsl

import (
	"context"
	"path/filepath"

	"github.com/go-go-golems/remarquee/pkg/rmdsl"
	"github.com/go-go-golems/remarquee/pkg/rmdsl/compile"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type compileConfig struct {
	OutPath  string
	DocUUID  string
	AuthorID uint8
	CaseRoot string
}

func NewCompileCobraCommand() (*cobra.Command, error) {
	cfg := compileConfig{}

	cmd := &cobra.Command{
		Use:   "compile <case.(yaml|yml|js)> --out <file.rmdoc>",
		Short: "Compile RMDoc-DSL to a V6 .rmdoc (strokes-only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg.OutPath == "" {
				return errors.New("--out is required")
			}
			inPath := args[0]
			if cfg.CaseRoot == "" {
				cfg.CaseRoot = filepath.Dir(inPath)
			}

			ctx := context.Background()
			doc, err := rmdsl.LoadFromFile(ctx, inPath, rmdsl.LoadOptions{CaseRoot: cfg.CaseRoot})
			if err != nil {
				return errors.Wrap(err, "load rmdsl case")
			}

			opts := compile.CompileOptions{
				DocUUID: cfg.DocUUID,
				Author:  cfg.AuthorID,
			}
			if err := compile.CompileToRMDoc(ctx, doc, cfg.OutPath, opts); err != nil {
				return errors.Wrap(err, "compile rmdsl")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&cfg.OutPath, "out", "", "Output .rmdoc path")
	cmd.Flags().StringVar(&cfg.DocUUID, "doc-uuid", "", "Override document UUID (optional)")
	cmd.Flags().Uint8Var(&cfg.AuthorID, "author-id", 1, "CRDT author id (part1)")
	cmd.Flags().StringVar(&cfg.CaseRoot, "case-root", "", "Root for rm.include() paths (optional)")

	return cmd, nil
}
