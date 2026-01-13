package devicecapture

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
)

// ScreenInfo describes the framebuffer layout for the current device.
type ScreenInfo struct {
	Model           string `json:"model"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	BytesPerPixel   int    `json:"bytesPerPixel"`
	ScreenSizeBytes int    `json:"screenSizeBytes"`
}

// FramebufferReader reads raw framebuffer bytes from the device.
type FramebufferReader interface {
	ReadFrame(dst []byte) error
	ScreenInfo() ScreenInfo
	Close() error
}

type reader struct {
	file        readAtCloser
	pointerAddr int64
	info        ScreenInfo
}

// NewFramebufferReader returns a reader configured for the current device.
func NewFramebufferReader() (FramebufferReader, error) {
	file, pointerAddr, info, err := getFileAndPointer()
	if err != nil {
		return nil, err
	}
	return &reader{
		file:        file,
		pointerAddr: pointerAddr,
		info:        info,
	}, nil
}

func (r *reader) ReadFrame(dst []byte) error {
	if len(dst) < r.info.ScreenSizeBytes {
		return fmt.Errorf("frame buffer too small: need %d bytes", r.info.ScreenSizeBytes)
	}
	_, err := r.file.ReadAt(dst[:r.info.ScreenSizeBytes], r.pointerAddr)
	return err
}

func (r *reader) ScreenInfo() ScreenInfo {
	return r.info
}

func (r *reader) Close() error {
	return r.file.Close()
}

// CaptureRaw reads the framebuffer into a new byte slice.
func CaptureRaw() ([]byte, ScreenInfo, error) {
	r, err := NewFramebufferReader()
	if err != nil {
		return nil, ScreenInfo{}, err
	}
	defer func() {
		if err := r.Close(); err != nil {
			// Best-effort cleanup; capture already happened.
			_ = err
		}
	}()

	info := r.ScreenInfo()
	buf := make([]byte, info.ScreenSizeBytes)
	if err := r.ReadFrame(buf); err != nil {
		return nil, ScreenInfo{}, err
	}
	return buf, info, nil
}

// CapturePNG captures the framebuffer and returns encoded PNG bytes.
func CapturePNG() ([]byte, ScreenInfo, error) {
	raw, info, err := CaptureRaw()
	if err != nil {
		return nil, ScreenInfo{}, err
	}
	img, err := RawToRGBA(raw, info)
	if err != nil {
		return nil, ScreenInfo{}, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, ScreenInfo{}, err
	}
	return buf.Bytes(), info, nil
}

// EncodePNG writes the image as PNG into w.
func EncodePNG(w io.Writer, img image.Image) error {
	return png.Encode(w, img)
}

// RawToRGBA converts raw framebuffer bytes into an RGBA image.
func RawToRGBA(raw []byte, info ScreenInfo) (*image.RGBA, error) {
	if info.Width == 0 || info.Height == 0 {
		return nil, fmt.Errorf("invalid screen info: width=%d height=%d", info.Width, info.Height)
	}
	if len(raw) < info.ScreenSizeBytes {
		return nil, fmt.Errorf("raw buffer too small: need %d bytes", info.ScreenSizeBytes)
	}
	img := image.NewRGBA(image.Rect(0, 0, info.Width, info.Height))
	switch info.BytesPerPixel {
	case 4:
		copy(img.Pix, raw[:info.ScreenSizeBytes])
	case 2:
		// Map 16-bit grayscale to RGBA using the low byte (matches goMarkableStream RLE mapping).
		for y := 0; y < info.Height; y++ {
			for x := 0; x < info.Width; x++ {
				idx := (y*info.Width + x) * 2
				val := int(raw[idx]) * 10
				if val > 255 {
					val = 255
				}
				img.SetRGBA(x, y, color.RGBA{R: uint8(val), G: uint8(val), B: uint8(val), A: 255})
			}
		}
	default:
		return nil, fmt.Errorf("unsupported bytes per pixel: %d", info.BytesPerPixel)
	}
	return img, nil
}
