package compile

import (
	"encoding/json"
	"fmt"
	"time"
)

type metadataEnvelope struct {
	CreatedTime    string `json:"createdTime"`
	LastModified   string `json:"lastModified"`
	LastOpened     string `json:"lastOpened"`
	LastOpenedPage int    `json:"lastOpenedPage"`
	New            bool   `json:"new"`
	Parent         string `json:"parent"`
	Pinned         bool   `json:"pinned"`
	Source         string `json:"source"`
	Type           string `json:"type"`
	VisibleName    string `json:"visibleName"`
}

func buildMetadataJSON(doc *CompiledDoc, now time.Time) ([]byte, error) {
	if now.IsZero() {
		now = time.Now()
	}
	nowMS := fmt.Sprintf("%d", now.UnixMilli())

	lastPage := 0
	if len(doc.Pages) > 0 {
		lastPage = len(doc.Pages) - 1
	}

	env := metadataEnvelope{
		CreatedTime:    nowMS,
		LastModified:   nowMS,
		LastOpened:     nowMS,
		LastOpenedPage: lastPage,
		New:            false,
		Parent:         "",
		Pinned:         false,
		Source:         "",
		Type:           "DocumentType",
		VisibleName:    doc.Name,
	}

	return json.MarshalIndent(env, "", "    ")
}
