package assets

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"

	"golang.org/x/image/draw"
)

// TrayIcon returns the logo resized to size×size pixels, wrapped in a minimal
// single-frame .ico container, suitable for use with fyne.io/systray.SetIcon().
// Uses Lanczos (CatmullRom) interpolation for crisp downscaling.
//
// The .ico wrapping matters specifically on Windows: systray's Windows
// backend loads icons via the Win32 LoadImage(IMAGE_ICON, ...) API, which
// requires a real ICO container — a bare PNG fails to load there (LoadImage
// returns 0, surfacing as a misleading "unable to set icon: The operation
// completed successfully" error, and leaving the tray icon blank). ICO files
// with a single PNG-compressed frame have been valid since Windows Vista and
// are also accepted by systray on macOS/Linux, so this format works everywhere.
func TrayIcon(size int) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(LogoKongPNG))
	if err != nil {
		return nil, err
	}

	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, dst); err != nil {
		return nil, err
	}

	return wrapICO(pngBuf.Bytes(), size), nil
}

// wrapICO wraps a single square PNG image in a minimal ICONDIR/ICONDIRENTRY
// .ico header. See https://en.wikipedia.org/wiki/ICO_(file_format).
func wrapICO(pngBytes []byte, size int) []byte {
	dim := byte(size)
	if size >= 256 {
		dim = 0 // 0 means 256 in the ICO format's 1-byte width/height fields
	}

	const headerLen = 6 + 16 // ICONDIR + one ICONDIRENTRY
	buf := make([]byte, 0, headerLen+len(pngBytes))

	// ICONDIR: reserved(2)=0, type(2)=1 (icon), count(2)=1
	buf = append(buf, 0, 0, 1, 0, 1, 0)

	// ICONDIRENTRY: width, height, colorCount, reserved, planes(2), bitCount(2)
	buf = append(buf, dim, dim, 0, 0, 1, 0, 32, 0)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(pngBytes))) // bytesInRes
	buf = binary.LittleEndian.AppendUint32(buf, uint32(headerLen))    // imageOffset

	return append(buf, pngBytes...)
}
