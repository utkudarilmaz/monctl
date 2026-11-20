package tray

import (
	"bytes"
	"image/color"
	"image/png"
	"testing"
)

func TestIconIsAPNGWithATransparentBackground(t *testing.T) {
	img, err := png.Decode(bytes.NewReader(Icon(color.White)))
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != iconSize || b.Dy() != iconSize {
		t.Fatalf("icon is %dx%d, want %dx%d", b.Dx(), b.Dy(), iconSize, iconSize)
	}
	if _, _, _, a := img.At(0, 0).RGBA(); a != 0 {
		t.Fatal("the corner must be transparent")
	}
	if r, _, _, a := img.At(5, 9).RGBA(); a == 0 || r == 0 {
		t.Fatal("the frame must be drawn in the given colour")
	}
}
