package handler

import (
	"image"
	"image/color"
	"testing"
)

func TestPrepareImageRespectsMaximumDimension(t *testing.T) {
	previousMaximum := MaxImageDimension
	MaxImageDimension = 100
	t.Cleanup(func() {
		MaxImageDimension = previousMaximum
	})

	source := image.NewRGBA(image.Rect(0, 0, 400, 200))
	source.Set(10, 10, color.White)

	pixels, width, height := prepareImage(source)
	if width != 100 || height != 50 {
		t.Fatalf("size = %dx%d, expected 100x50", width, height)
	}
	if len(pixels) != width*height*4 {
		t.Fatalf("pixel buffer length = %d, expected %d", len(pixels), width*height*4)
	}
}
