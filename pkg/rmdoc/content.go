package rmdoc

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// ParseContent parses `.content` JSON and returns:
// - detected schema (legacy vs cPages)
// - document type (pdf/notebook/epub)
// - deterministic page plan in UI order
func ParseContent(contentJSON []byte) (ArchiveSchema, DocumentType, []PageRef, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(contentJSON, &raw); err != nil {
		return SchemaUnknown, DocTypeUnknown, nil, errors.Wrap(err, "unmarshal .content (raw)")
	}

	docType := detectDocType(raw)

	if _, ok := raw["cPages"]; ok {
		pages, err := parseCPages(raw)
		if err != nil {
			return SchemaCPages, docType, nil, err
		}
		return SchemaCPages, docType, pages, nil
	}

	pages, err := parseLegacyPages(contentJSON)
	if err != nil {
		return SchemaLegacy, docType, nil, err
	}
	return SchemaLegacy, docType, pages, nil
}

func detectDocType(raw map[string]json.RawMessage) DocumentType {
	// legacy uses "fileType"; sometimes empty for notebooks
	var ft string
	if b, ok := raw["fileType"]; ok {
		_ = json.Unmarshal(b, &ft)
	}
	ft = strings.ToLower(strings.TrimSpace(ft))

	switch ft {
	case "pdf":
		return DocTypePDF
	case "epub":
		return DocTypeEPUB
	case "":
		return DocTypeNotebook
	default:
		return DocTypeUnknown
	}
}

type value[T any] struct {
	Timestamp string `json:"timestamp,omitempty"`
	Value     T      `json:"value,omitempty"`
}

type cPagesEnvelope struct {
	CPages cPages `json:"cPages"`
}

type cPages struct {
	Pages []cPage `json:"pages"`
}

type cPage struct {
	ID       string         `json:"id"`
	Redir    *value[int]    `json:"redir,omitempty"`
	Deleted  *value[int]    `json:"deleted,omitempty"`
	Template *value[string] `json:"template,omitempty"`
}

func parseCPages(raw map[string]json.RawMessage) ([]PageRef, error) {
	var env cPagesEnvelope
	if b, ok := raw["cPages"]; ok {
		// use a partial envelope to avoid double parse
		if err := json.Unmarshal(b, &env.CPages); err != nil {
			return nil, errors.Wrap(err, "unmarshal cPages")
		}
	} else {
		return nil, errors.New("expected cPages")
	}

	out := make([]PageRef, 0, len(env.CPages.Pages))
	for _, p := range env.CPages.Pages {
		deleted := false
		if p.Deleted != nil && p.Deleted.Value == 1 {
			deleted = true
		}
		if deleted {
			continue
		}

		src := InsertedPage
		if p.Redir != nil {
			src = p.Redir.Value
		}

		tmpl := ""
		if p.Template != nil {
			tmpl = p.Template.Value
		}

		out = append(out, PageRef{
			Index:         len(out),
			PageID:        p.ID,
			SourcePDFPage: src,
			Template:      tmpl,
			Deleted:       false,
		})
	}

	return out, nil
}

type legacyEnvelope struct {
	PageCount      int      `json:"pageCount"`
	Pages          []string `json:"pages"`
	RedirectionMap []int    `json:"redirectionPageMap"`
}

func parseLegacyPages(contentJSON []byte) ([]PageRef, error) {
	var env legacyEnvelope
	// Unmarshal only the fields we care about; unknown keys are ignored.
	if err := json.Unmarshal(contentJSON, &env); err != nil {
		return nil, errors.Wrap(err, "unmarshal legacy .content")
	}

	pageCount := env.PageCount
	if len(env.Pages) > pageCount {
		pageCount = len(env.Pages)
	}
	if len(env.RedirectionMap) > pageCount {
		pageCount = len(env.RedirectionMap)
	}

	out := make([]PageRef, 0, pageCount)
	for i := 0; i < pageCount; i++ {
		pageID := ""
		if i < len(env.Pages) {
			pageID = env.Pages[i]
		} else {
			// Some legacy archives only contain pageCount.
			pageID = strconv.Itoa(i)
		}

		src := i
		if i < len(env.RedirectionMap) {
			src = env.RedirectionMap[i]
		}

		out = append(out, PageRef{
			Index:         i,
			PageID:        pageID,
			SourcePDFPage: src,
			Template:      "",
			Deleted:       false,
		})
	}

	return out, nil
}
