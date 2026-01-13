package rmdoc

import "strings"

// ApplyPagedataTemplates fills in PageRef.Template from `.pagedata` lines when it is not already set.
//
// The `.pagedata` file stores one template name per line. The number of lines should match the UI page count.
func ApplyPagedataTemplates(pages []PageRef, pagedata string) []PageRef {
	if pagedata == "" || len(pages) == 0 {
		return pages
	}

	lines := strings.Split(pagedata, "\n")
	for i := range pages {
		if pages[i].Template != "" {
			continue
		}
		if i >= len(lines) {
			break
		}
		t := strings.TrimSpace(lines[i])
		if t == "" {
			continue
		}
		pages[i].Template = t
	}

	return pages
}
