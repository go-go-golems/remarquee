package rmdoc

import "testing"

func TestParseContentV6_CPages(t *testing.T) {
	j := []byte(`{
  "fileType": "pdf",
  "cPages": {
    "pages": [
      {"id":"a","redir":{"timestamp":"1:1","value":0},"template":{"timestamp":"1:1","value":"Blank"}},
      {"id":"b","deleted":{"timestamp":"1:1","value":1}},
      {"id":"c"}
    ]
  }
}`)

	schema, dt, pages, err := ParseContent(j)
	if err != nil {
		t.Fatalf("ParseContent: %v", err)
	}
	if schema != SchemaCPages {
		t.Fatalf("schema = %v, want %v", schema, SchemaCPages)
	}
	if dt != DocTypePDF {
		t.Fatalf("docType = %v, want %v", dt, DocTypePDF)
	}
	if len(pages) != 2 {
		t.Fatalf("pages = %d, want 2", len(pages))
	}

	if pages[0].Index != 0 || pages[0].PageID != "a" || pages[0].SourcePDFPage != 0 || pages[0].Template != "Blank" {
		t.Fatalf("page0 = %+v (unexpected)", pages[0])
	}
	if pages[1].Index != 1 || pages[1].PageID != "c" || pages[1].SourcePDFPage != InsertedPage {
		t.Fatalf("page1 = %+v (unexpected)", pages[1])
	}
}

func TestParseContentLegacy(t *testing.T) {
	j := []byte(`{
  "fileType": "pdf",
  "pageCount": 2,
  "pages": ["u1","u2"],
  "redirectionPageMap": [0, 1]
}`)

	schema, dt, pages, err := ParseContent(j)
	if err != nil {
		t.Fatalf("ParseContent: %v", err)
	}
	if schema != SchemaLegacy {
		t.Fatalf("schema = %v, want %v", schema, SchemaLegacy)
	}
	if dt != DocTypePDF {
		t.Fatalf("docType = %v, want %v", dt, DocTypePDF)
	}
	if len(pages) != 2 {
		t.Fatalf("pages = %d, want 2", len(pages))
	}
	if pages[0].PageID != "u1" || pages[0].SourcePDFPage != 0 {
		t.Fatalf("page0 = %+v (unexpected)", pages[0])
	}
	if pages[1].PageID != "u2" || pages[1].SourcePDFPage != 1 {
		t.Fatalf("page1 = %+v (unexpected)", pages[1])
	}
}

func TestApplyPagedataTemplates(t *testing.T) {
	pages := []PageRef{
		{Index: 0, PageID: "a"},
		{Index: 1, PageID: "b", Template: "AlreadySet"},
		{Index: 2, PageID: "c"},
	}
	out := ApplyPagedataTemplates(pages, "T1\nT2\nT3\n")
	if out[0].Template != "T1" {
		t.Fatalf("out[0].Template=%q, want %q", out[0].Template, "T1")
	}
	if out[1].Template != "AlreadySet" {
		t.Fatalf("out[1].Template=%q, want %q", out[1].Template, "AlreadySet")
	}
	if out[2].Template != "T3" {
		t.Fatalf("out[2].Template=%q, want %q", out[2].Template, "T3")
	}
}
