package tray

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

const iconSize = 44

// Icon draws a small monitor shape and returns it as a PNG.
func Icon(c color.Color) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, iconSize, iconSize))
	fill := func(x0, y0, x1, y1 int) {
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				img.Set(x, y, c)
			}
		}
	}
	fill(4, 8, 40, 11)
	fill(4, 27, 40, 30)
	fill(4, 8, 7, 30)
	fill(37, 8, 40, 30)
	fill(20, 30, 24, 35)
	fill(13, 35, 31, 38)

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
