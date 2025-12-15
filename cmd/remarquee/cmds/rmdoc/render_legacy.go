package rmdoc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	rmapi_annotations "github.com/juruen/rmapi/annotations"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	pkg_rmdoc "github.com/go-go-golems/remarquee/pkg/rmdoc"
)

type renderLegacySettings struct {
	Out             string
	Force           bool
	AddPageNumbers  bool
	AllPages        bool
	AnnotationsOnly bool
}

func NewRenderLegacyCommand() *cobra.Command {
	s := &renderLegacySettings{}

	cmd := &cobra.Command{
		Use:   "render-legacy <file.rmdoc|file.zip>",
		Short: "Render a legacy (V3/V5) .rmdoc/.zip to an annotated PDF (rmapi-backed)",
		Long: "This command only supports legacy .rmdoc/.zip archives (legacy .content + V3/V5 .rm).\n" +
			"It delegates PDF generation to rmapi's annotations PdfGenerator.\n" +
			"For cPages/V6 documents, this will return an error (V6 rendering is not implemented yet).",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			in := args[0]

			doc, err := pkg_rmdoc.OpenFile(context.Background(), in)
			if err != nil {
				return err
			}
			if doc.Schema != pkg_rmdoc.SchemaLegacy {
				return errors.Errorf("render-legacy only supports legacy archives; detected schema=%s", schemaString(doc.Schema))
			}
			if doc.Type == pkg_rmdoc.DocTypeEPUB {
				return errors.New("render-legacy: epub not supported")
			}

			out := s.Out
			if out == "" {
				base := filepath.Base(in)
				ext := filepath.Ext(base)
				base = base[:len(base)-len(ext)]
				out = base + "-annotations.pdf"
			}

			if !s.Force {
				if _, err := os.Stat(out); err == nil {
					return errors.Errorf("output file exists: %s (use --force to overwrite)", out)
				}
			}

			opts := rmapi_annotations.PdfGeneratorOptions{
				AddPageNumbers:  s.AddPageNumbers,
				AllPages:        s.AllPages,
				AnnotationsOnly: s.AnnotationsOnly,
			}

			g := rmapi_annotations.CreatePdfGenerator(in, out, opts)
			if err := g.Generate(); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "ok: wrote %s\n", out)
			return nil
		},
	}

	cmd.Flags().StringVar(&s.Out, "out", "", "Output PDF path (default: <input>-annotations.pdf in current dir)")
	cmd.Flags().BoolVar(&s.Force, "force", false, "Overwrite output file if it exists")
	cmd.Flags().BoolVar(&s.AddPageNumbers, "add-page-numbers", false, "Add page numbers")
	cmd.Flags().BoolVar(&s.AllPages, "all-pages", false, "Include pages without annotations")
	cmd.Flags().BoolVar(&s.AnnotationsOnly, "annotations-only", false, "Export annotations only (no background PDF)")

	return cmd
}
