package compile

import (
	"math"
	"strings"
	"unicode/utf8"

	"github.com/go-go-golems/remarquee/pkg/rmdsl"
)

var textStyleMap = map[string]uint8{
	"basic":            0,
	"plain":            1,
	"heading":          2,
	"bold":             3,
	"bullet":           4,
	"bullet2":          5,
	"checkbox":         6,
	"checkbox_checked": 7,
}

func compileTextBlock(tb *rmdsl.Text, canvasW int) *CompiledText {
	if tb == nil {
		return nil
	}
	text := strings.TrimRight(tb.Content, "\n")
	if text == "" {
		return nil
	}

	width := tb.Width
	if width <= 0 {
		width = float64(canvasW) * 2.0 / 3.0
	}

	posX := tb.PosX
	posY := tb.PosY
	if posX == 0 && posY == 0 {
		posX = -width / 2.0
		posY = 234.0
	}

	style, ok := textStyleMap[strings.ToLower(strings.TrimSpace(tb.Style))]
	if !ok {
		style = 1
	}

	return &CompiledText{
		Text:  text,
		Style: style,
		PosX:  posX,
		PosY:  posY,
		Width: width,
	}
}

func textCounts(text string) (uint32, uint32) {
	if text == "" {
		return 0, 0
	}
	chars := clampIntToUint32(utf8.RuneCountInString(text) + 1)
	lines := clampIntToUint32(strings.Count(text, "\n") + 1)
	return chars, lines
}

func clampIntToUint32(v int) uint32 {
	if v <= 0 {
		return 0
	}
	if v > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(v)
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 0x7F {
			return false
		}
	}
	return true
}
