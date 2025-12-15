package rmdoc

import pkg_rmdoc "github.com/go-go-golems/remarquee/pkg/rmdoc"

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


