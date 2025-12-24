package rmdoc

import (
	"bytes"
	"io"
	"math"

	"github.com/pkg/errors"
)

// DecodeRMV6Line decodes a V6 line item payload (the bytes inside SceneLineItemBlock value subblock,
// after the initial item_type byte) into a normalized Stroke.
//
// It mirrors `rmscene.scene_stream.line_from_stream`.
func DecodeRMV6Line(lineVersion uint8, payload []byte) (*Stroke, error) {
	vr := newRMV6ValueReader(bytes.NewReader(payload), int64(len(payload)))

	tool, err := vr.readUint32(1)
	if err != nil {
		return nil, errors.Wrap(err, "read tool")
	}
	color, err := vr.readUint32(2)
	if err != nil {
		return nil, errors.Wrap(err, "read color")
	}
	thicknessScale, err := vr.readFloat64(3)
	if err != nil {
		return nil, errors.Wrap(err, "read thickness_scale")
	}
	startingLength, err := vr.readFloat32(4)
	if err != nil {
		return nil, errors.Wrap(err, "read starting_length")
	}

	sb, err := vr.readSubBlock(5)
	if err != nil {
		return nil, errors.Wrap(err, "read points subblock")
	}

	pointSize, err := rmv6PointSerializedSize(lineVersion)
	if err != nil {
		_ = vr.endSubBlock(sb)
		return nil, err
	}

	if sb.Size%uint32(pointSize) != 0 {
		_ = vr.endSubBlock(sb)
		return nil, errors.Errorf("point data size mismatch: %d is not multiple of point_size %d", sb.Size, pointSize)
	}
	numPoints := int(sb.Size) / pointSize

	points := make([]StrokePoint, 0, numPoints)
	for i := 0; i < numPoints; i++ {
		p, err := rmv6PointFromStream(vr, lineVersion)
		if err != nil {
			_ = vr.endSubBlock(sb)
			return nil, errors.Wrap(err, "read point")
		}
		points = append(points, p)
	}

	if err := vr.endSubBlock(sb); err != nil {
		return nil, errors.Wrap(err, "end points subblock")
	}

	// Timestamp (tag 6) appears present in python; keep optional in case of variants.
	if ok, _ := vr.checkTag(6, rmV6TagTypeID); ok {
		_, _ = vr.readID(6)
	}

	// Optional move_id at tag 7.
	if ok, _ := vr.checkTag(7, rmV6TagTypeID); ok {
		_, _ = vr.readID(7)
	}

	// Optional trailing highlight/shader RGBA marker: 2 bytes prefix + 4 bytes (b,g,r,a).
	// Only consume it if it matches the known prefix 0x84 0x01, otherwise leave bytes untouched.
	rem, _ := vr.bytesRemaining()
	if rem >= 6 {
		b6, err := vr.peekBytes(6)
		if err == nil && len(b6) == 6 && b6[0] == 0x84 && b6[1] == 0x01 {
			_, _ = vr.readBytes(6)
		}
	}

	// Ensure we don't fail on extra bytes: newer formats may add fields.
	// (We intentionally do not enforce "read to end".)
	return &Stroke{
		Tool:           tool,
		Color:          color,
		ThicknessScale: thicknessScale,
		StartingLength: float64(startingLength),
		Points:         points,
	}, nil
}

func rmv6PointSerializedSize(version uint8) (int, error) {
	switch version {
	case 1:
		return 0x18, nil
	case 2:
		return 0x0E, nil
	default:
		return 0, errors.Errorf("unknown point version %d", version)
	}
}

func rmv6PointFromStream(vr *rmV6ValueReader, version uint8) (StrokePoint, error) {
	x, err := vr.readFloat32LE()
	if err != nil {
		return StrokePoint{}, err
	}
	y, err := vr.readFloat32LE()
	if err != nil {
		return StrokePoint{}, err
	}

	switch version {
	case 1:
		// rmscene: speed = float32*4, direction = 255*float32/(2*pi), width = round(float32*4), pressure = float32*255
		speedRaw, err := vr.readFloat32LE()
		if err != nil {
			return StrokePoint{}, err
		}
		dirRaw, err := vr.readFloat32LE()
		if err != nil {
			return StrokePoint{}, err
		}
		widthRaw, err := vr.readFloat32LE()
		if err != nil {
			return StrokePoint{}, err
		}
		pressureRaw, err := vr.readFloat32LE()
		if err != nil {
			return StrokePoint{}, err
		}

		return StrokePoint{
			X:         float64(x),
			Y:         float64(y),
			Speed:     float64(speedRaw) * 4,
			Direction: 255 * float64(dirRaw) / (math.Pi * 2),
			Width:     math.Round(float64(widthRaw) * 4),
			Pressure:  float64(pressureRaw) * 255,
		}, nil

	case 2:
		speed, err := vr.readUint16LE()
		if err != nil {
			return StrokePoint{}, err
		}
		width, err := vr.readUint16LE()
		if err != nil {
			return StrokePoint{}, err
		}
		direction, err := vr.readUint8()
		if err != nil {
			return StrokePoint{}, err
		}
		pressure, err := vr.readUint8()
		if err != nil {
			return StrokePoint{}, err
		}
		return StrokePoint{
			X:         float64(x),
			Y:         float64(y),
			Speed:     float64(speed),
			Direction: float64(direction),
			Width:     float64(width),
			Pressure:  float64(pressure),
		}, nil

	default:
		return StrokePoint{}, io.ErrUnexpectedEOF
	}
}
