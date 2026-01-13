package rmdoc

// RMV6TextTopY mirrors rmc's TEXT_TOP_Y (screen units).
const RMV6TextTopY = -88.0

var rmv6LineHeights = map[uint8]float64{
	1: 70,  // PLAIN
	3: 70,  // BOLD
	2: 150, // HEADING
	4: 35,  // BULLET
	5: 35,  // BULLET2
	6: 35,  // CHECKBOX
	7: 35,  // CHECKBOX_CHECKED
	0: 70,  // BASIC
}

func RMV6ParagraphLineHeight(style uint8) float64 {
	if lh, ok := rmv6LineHeights[style]; ok {
		return lh
	}
	return 70
}
