package rmdoc

// StrokePoint is a normalized point primitive for rendering.
//
// For V6 it maps closely to the point encodings in rmscene's `scene_stream.point_from_stream`.
// For V3/V5 we can adapt rmapi's stroke representations later.
type StrokePoint struct {
	X float64
	Y float64

	// Optional per-point dynamics. Units depend on RM format version.
	Speed     float64
	Direction float64
	Width     float64
	Pressure  float64
}

// Stroke is a normalized stroke primitive for rendering.
type Stroke struct {
	Tool  uint32
	Color uint32

	ThicknessScale float64
	StartingLength float64

	Points []StrokePoint
}
