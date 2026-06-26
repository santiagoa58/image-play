package imageutil

import (
	"image"
	"image/color"
	"testing"
)

func TestFilterForegroundComponentsRemovesSmallDetachedRegions(t *testing.T) {
	mask := image.NewGray(image.Rect(0, 0, 20, 12))
	fillGrayRect(mask, image.Rect(1, 1, 12, 10), 255)
	fillGrayRect(mask, image.Rect(16, 1, 19, 4), 255)

	filtered := filterForegroundComponents(mask, 0.2)

	if filtered.GrayAt(5, 5).Y == 0 {
		t.Fatal("expected the largest foreground component to remain")
	}
	if filtered.GrayAt(17, 2).Y != 0 {
		t.Fatal("expected the small detached foreground component to be removed")
	}
}

func TestFilterForegroundComponentsKeepsComparableDetachedRegions(t *testing.T) {
	mask := image.NewGray(image.Rect(0, 0, 20, 12))
	fillGrayRect(mask, image.Rect(1, 1, 8, 10), 255)
	fillGrayRect(mask, image.Rect(11, 1, 18, 10), 255)

	filtered := filterForegroundComponents(mask, 0.2)

	if filtered.GrayAt(4, 5).Y == 0 || filtered.GrayAt(14, 5).Y == 0 {
		t.Fatal("expected substantial detached foreground components to remain")
	}
}

func fillGrayRect(img *image.Gray, rect image.Rectangle, value uint8) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.SetGray(x, y, color.Gray{Y: value})
		}
	}
}
