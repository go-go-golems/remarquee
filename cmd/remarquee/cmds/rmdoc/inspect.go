package rmdoc

import (
	"context"
	"encoding/json"
	"fmt"

	pkg_rmdoc "github.com/go-go-golems/remarquee/pkg/rmdoc"
	"github.com/spf13/cobra"
)

type inspectSettings struct {
	AsJSON bool
}

func NewInspectCommand() *cobra.Command {
	s := &inspectSettings{}

	cmd := &cobra.Command{
		Use:   "inspect <file.rmdoc>",
		Short: "Inspect a local .rmdoc and print detected schema + page plan",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			doc, err := pkg_rmdoc.OpenFile(ctx, args[0])
			if err != nil {
				return err
			}

			if s.AsJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(doc)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "uuid=%s schema=%s type=%s pages=%d\n", doc.UUID, schemaString(doc.Schema), docTypeString(doc.Type), len(doc.Pages))
			fmt.Fprintln(cmd.OutOrStdout(), "idx\tpage_id\tsrc_pdf\ttemplate")
			for _, p := range doc.Pages {
				fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\t%d\t%s\n", p.Index, p.PageID, p.SourcePDFPage, p.Template)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&s.AsJSON, "json", false, "Output as JSON")
	return cmd
}

func schemaString(s pkg_rmdoc.ArchiveSchema) string {
	switch s {
	case pkg_rmdoc.SchemaLegacy:
		return "legacy"
	case pkg_rmdoc.SchemaCPages:
		return "cPages"
	default:
		return "unknown"
	}
}

func docTypeString(t pkg_rmdoc.DocumentType) string {
	switch t {
	case pkg_rmdoc.DocTypeNotebook:
		return "notebook"
	case pkg_rmdoc.DocTypePDF:
		return "pdf"
	case pkg_rmdoc.DocTypeEPUB:
		return "epub"
	default:
		return "unknown"
	}
}
