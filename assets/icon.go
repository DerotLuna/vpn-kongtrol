package assets

import (
	"bytes"
	"image"
	"image/png"

	"golang.org/x/image/draw"
)

// TrayIcon returns the logo resized to size×size pixels as PNG bytes,
// suitable for use with fyne.io/systray.SetIcon().
// Uses Lanczos (CatmullRom) interpolation for crisp downscaling.
func TrayIcon(size int) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(LogoKongPNG))
	if err != nil {
		return nil, err
	}

	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
