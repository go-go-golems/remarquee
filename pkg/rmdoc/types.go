package rmdoc

type ArchiveSchema int

const (
	SchemaUnknown ArchiveSchema = iota
	SchemaLegacy
	SchemaCPages
)

type DocumentType int

const (
	DocTypeUnknown DocumentType = iota
	DocTypeNotebook
	DocTypePDF
	DocTypeEPUB
)

// InsertedPage indicates that a UI page has no corresponding source PDF page.
// This usually means a blank page inserted into a PDF-based document.
const InsertedPage = -1

// Document is an opened `.rmdoc` archive with minimal parsed metadata and an ordered page plan.
// This struct intentionally keeps raw JSON bytes for forward-compat/debugging.
type Document struct {
	UUID   string
	Schema ArchiveSchema
	Type   DocumentType

	// Raw JSON blobs from the archive.
	ContentJSON  []byte
	MetadataJSON []byte

	// Pagedata is one template name per line, as stored by the device.
	Pagedata string

	// PayloadPDF is the original PDF payload when present (PDF docs).
	PayloadPDF []byte

	Pages []PageRef
}

// PageRef describes the i-th UI page in reading order, including how it maps back to a source PDF.
type PageRef struct {
	Index int

	// PageID is the identifier used by `.rm` filenames.
	// For cPages this is a UUID; for legacy it is often a UUID but can vary.
	PageID string

	// SourcePDFPage is the background PDF page index (0-based), or InsertedPage.
	SourcePDFPage int

	// Template is the template background name (from cPages.template or pagedata).
	Template string

	Deleted bool
}
