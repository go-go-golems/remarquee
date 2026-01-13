package compile

import (
	"encoding/json"
	"fmt"
	"time"
)

type lwwValue[T any] struct {
	Timestamp string `json:"timestamp"`
	Value     T      `json:"value"`
}

type cPage struct {
	ID       string            `json:"id"`
	Idx      *lwwValue[string] `json:"idx,omitempty"`
	Modifed  string            `json:"modifed,omitempty"`
	Redir    *lwwValue[int]    `json:"redir,omitempty"`
	Deleted  *lwwValue[int]    `json:"deleted,omitempty"`
	Template *lwwValue[string] `json:"template,omitempty"`
}

type cPages struct {
	LastOpened *lwwValue[string] `json:"lastOpened,omitempty"`
	Original   *lwwValue[int]    `json:"original,omitempty"`
	Pages      []cPage           `json:"pages"`
	UUIDs      []uuidPair        `json:"uuids,omitempty"`
}

type uuidPair struct {
	First  string `json:"first"`
	Second int    `json:"second"`
}

type contentEnvelope struct {
	CPages                cPages            `json:"cPages"`
	CoverPageNumber       int               `json:"coverPageNumber"`
	CustomZoomCenterX     int               `json:"customZoomCenterX"`
	CustomZoomCenterY     int               `json:"customZoomCenterY"`
	CustomZoomOrientation string            `json:"customZoomOrientation"`
	CustomZoomPageHeight  int               `json:"customZoomPageHeight"`
	CustomZoomPageWidth   int               `json:"customZoomPageWidth"`
	CustomZoomScale       int               `json:"customZoomScale"`
	DocumentMetadata      map[string]any    `json:"documentMetadata"`
	ExtraMetadata         map[string]string `json:"extraMetadata"`
	FileType              string            `json:"fileType"`
	FontName              string            `json:"fontName"`
	FormatVersion         int               `json:"formatVersion"`
	LineHeight            int               `json:"lineHeight"`
	Orientation           string            `json:"orientation"`
	PageCount             int               `json:"pageCount"`
	PageTags              []string          `json:"pageTags"`
	Tags                  []string          `json:"tags"`
	TextAlignment         string            `json:"textAlignment"`
	TextScale             int               `json:"textScale"`
	ZoomMode              string            `json:"zoomMode"`
}

func buildContentJSON(doc *CompiledDoc, now time.Time, canvasW int, canvasH int, authorID uint8, authorUUID string) ([]byte, error) {
	if now.IsZero() {
		now = time.Now()
	}
	nowMS := fmt.Sprintf("%d", now.UnixMilli())

	pages := make([]cPage, 0, len(doc.Pages))
	for i, p := range doc.Pages {
		ts := "1:1"
		pages = append(pages, cPage{
			ID:       p.ID,
			Idx:      &lwwValue[string]{Timestamp: ts, Value: pageIdx(i)},
			Modifed:  nowMS,
			Template: &lwwValue[string]{Timestamp: ts, Value: defaultTemplate(p.Template)},
		})
	}

	lastPageID := ""
	if len(pages) > 0 {
		lastPageID = pages[len(pages)-1].ID
	}

	env := contentEnvelope{
		CPages: cPages{
			LastOpened: &lwwValue[string]{Timestamp: "1:1", Value: lastPageID},
			Original:   &lwwValue[int]{Timestamp: "1:1", Value: -1},
			Pages:      pages,
			UUIDs:      []uuidPair{{First: authorUUID, Second: int(authorID)}},
		},
		CoverPageNumber:       -1,
		CustomZoomCenterX:     0,
		CustomZoomCenterY:     canvasH / 2,
		CustomZoomOrientation: "portrait",
		CustomZoomPageHeight:  canvasH,
		CustomZoomPageWidth:   canvasW,
		CustomZoomScale:       1,
		DocumentMetadata:      map[string]any{},
		ExtraMetadata:         map[string]string{},
		FileType:              "notebook",
		FontName:              "",
		FormatVersion:         2,
		LineHeight:            -1,
		Orientation:           "portrait",
		PageCount:             len(pages),
		PageTags:              []string{},
		Tags:                  []string{},
		TextAlignment:         "justify",
		TextScale:             1,
		ZoomMode:              "bestFit",
	}

	return json.MarshalIndent(env, "", "    ")
}

func pageIdx(i int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	if i < 0 {
		return "ba"
	}
	i = i % (26 * 26)
	first := 1 + (i / 26)
	if first >= len(letters) {
		first = len(letters) - 1
	}
	second := i % 26
	return string(letters[first]) + string(letters[second])
}

func defaultTemplate(t string) string {
	if t == "" {
		return "Blank"
	}
	return t
}
