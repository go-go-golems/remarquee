package js

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	
	"github.com/go-go-golems/remarquee/pkg/rmdoc"
	"github.com/pkg/errors"
)

// BuildAndSave builds the rmdoc document and saves it to a file
func BuildAndSave(doc *Document, path string) error {
	data, err := BuildToBytes(doc)
	if err != nil {
		return err
	}
	
	return os.WriteFile(path, data, 0644)
}

// BuildToBytes builds the rmdoc document and returns it as bytes
func BuildToBytes(doc *Document) ([]byte, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	
	// Create .content file
	content, err := buildContent(doc)
	if err != nil {
		zipWriter.Close()
		return nil, errors.Wrap(err, "failed to build content")
	}
	
	contentFile, err := zipWriter.Create(doc.uuid + ".content")
	if err != nil {
		zipWriter.Close()
		return nil, errors.Wrap(err, "failed to create content file")
	}
	if _, err := contentFile.Write(content); err != nil {
		zipWriter.Close()
		return nil, errors.Wrap(err, "failed to write content")
	}
	
	// Create .metadata file
	metadata, err := buildMetadata(doc)
	if err != nil {
		zipWriter.Close()
		return nil, errors.Wrap(err, "failed to build metadata")
	}
	
	metadataFile, err := zipWriter.Create(doc.uuid + ".metadata")
	if err != nil {
		zipWriter.Close()
		return nil, errors.Wrap(err, "failed to create metadata file")
	}
	if _, err := metadataFile.Write(metadata); err != nil {
		zipWriter.Close()
		return nil, errors.Wrap(err, "failed to write metadata")
	}
	
	// Create .pagedata file
	pagedata := buildPagedata(doc)
	pagedataFile, err := zipWriter.Create(doc.uuid + ".pagedata")
	if err != nil {
		zipWriter.Close()
		return nil, errors.Wrap(err, "failed to create pagedata file")
	}
	if _, err := pagedataFile.Write([]byte(pagedata)); err != nil {
		zipWriter.Close()
		return nil, errors.Wrap(err, "failed to write pagedata")
	}
	
	// Create .rm files for each page
	for _, page := range doc.pages {
		rmData, err := buildRMFile(page)
		if err != nil {
			zipWriter.Close()
			return nil, errors.Wrapf(err, "failed to build .rm file for page %d", page.index)
		}
		
		rmFile, err := zipWriter.Create(fmt.Sprintf("%s/%s.rm", doc.uuid, page.pageID))
		if err != nil {
			zipWriter.Close()
			return nil, errors.Wrap(err, "failed to create .rm file")
		}
		if _, err := rmFile.Write(rmData); err != nil {
			zipWriter.Close()
			return nil, errors.Wrap(err, "failed to write .rm file")
		}
	}
	
	if err := zipWriter.Close(); err != nil {
		return nil, errors.Wrap(err, "failed to close zip")
	}
	
	return buf.Bytes(), nil
}

func buildContent(doc *Document) ([]byte, error) {
	// Build cPages format
	pages := make([]map[string]interface{}, len(doc.pages))
	for i, page := range doc.pages {
		pages[i] = map[string]interface{}{
			"id": page.pageID,
			"template": map[string]interface{}{
				"value": page.template,
			},
		}
	}
	
	content := map[string]interface{}{
		"formatVersion":     1,
		"fileType":          doc.docType,
		"lastOpenedPage":    0,
		"lineHeight":        -1,
		"margins":           100,
		"pageCount":         len(doc.pages),
		"textScale":         1,
		"transform":         map[string]interface{}{},
		"cPages": map[string]interface{}{
			"pages": pages,
		},
	}
	
	return json.Marshal(content)
}

func buildMetadata(doc *Document) ([]byte, error) {
	metadata := map[string]interface{}{
		"visibleName":      doc.title,
		"lastModified":     "0",
		"version":          1,
		"pinned":           false,
		"synced":           false,
		"modified":         false,
		"deleted":          false,
		"metadatamodified": false,
		"parent":           "",
		"type":             "DocumentType",
	}
	
	return json.Marshal(metadata)
}

func buildPagedata(doc *Document) string {
	var buf bytes.Buffer
	for _, page := range doc.pages {
		buf.WriteString(page.template)
		buf.WriteString("\n")
	}
	return buf.String()
}

func buildRMFile(page *Page) ([]byte, error) {
	// Convert JS strokes to rmdoc strokes
	strokes := make([]rmdoc.Stroke, len(page.strokes))
	for i, jsStroke := range page.strokes {
		points := make([]rmdoc.StrokePoint, len(jsStroke.points))
		for j, jsPt := range jsStroke.points {
			points[j] = rmdoc.StrokePoint{
				X:         jsPt.X,
				Y:         jsPt.Y,
				Pressure:  jsPt.Pressure,
				Speed:     jsPt.Speed,
				Direction: jsPt.Direction,
				Width:     jsPt.Width,
			}
		}
		
		strokes[i] = rmdoc.Stroke{
			Tool:           jsStroke.tool,
			Color:          jsStroke.color,
			ThicknessScale: jsStroke.thickness,
			Points:         points,
		}
	}
	
	// Build scene tree
	tree := buildSceneTree(strokes)
	
	// Write to buffer using the new writer v2
	var buf bytes.Buffer
	if err := WriteSceneTreeV2(&buf, tree, strokes); err != nil {
		return nil, errors.Wrap(err, "failed to write scene tree")
	}
	
	return buf.Bytes(), nil
}

func buildSceneTree(strokes []rmdoc.Stroke) *rmdoc.RMV6SceneTree {
	tree := rmdoc.NewRMV6SceneTree()
	
	// Add each stroke as a line item to the root group
	for i := range strokes {
		itemID := rmdoc.RMV6CrdtID{Part1: uint8(i + 1), Part2: uint64(i + 1)}
		leftID := rmdoc.RMV6CrdtID{Part1: 0, Part2: 0}
		if i > 0 {
			leftID = rmdoc.RMV6CrdtID{Part1: uint8(i), Part2: uint64(i)}
		}
		rightID := rmdoc.RMV6CrdtID{Part1: 0, Part2: 0}
		
		// Line data will be encoded by the writer
		lineData := []byte{}
		
		item := rmdoc.RMV6CrdtSequenceItem[rmdoc.RMV6SceneItem]{
			ItemID:        itemID,
			LeftID:        leftID,
			RightID:       rightID,
			DeletedLength: 0,
			Value: rmdoc.RMV6SceneItem{
				Kind: rmdoc.RMV6SceneItemLine,
				Line: &rmdoc.RMV6Line{
					BlockVersion: 2,
					Raw:          lineData,
				},
			},
		}
		
		tree.AddItem(item, rmdoc.RMV6RootGroupID)
	}
	
	return tree
}


