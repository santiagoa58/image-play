package layout

import (
	"image"
	"testing"
)

func TestSpaceRejectsMaskedPixels(t *testing.T) {
	const width, height = 10, 10
	pixels := make([]uint8, width*height)
	for i := range pixels {
		pixels[i] = 255
	}
	pixels[5*width+5] = 0

	space, err := NewSpace(width, height, pixels)
	if err != nil {
		t.Fatalf("NewSpace() error = %v", err)
	}

	if space.TryPlace(image.Rect(4, 4, 7, 7)) {
		t.Fatal("TryPlace() = true for rectangle containing a masked pixel")
	}
	if !space.TryPlace(image.Rect(0, 0, 4, 4)) {
		t.Fatal("TryPlace() = false for fully allowed rectangle")
	}
}

func TestSpaceRejectsOverlapButAllowsTouchingEdges(t *testing.T) {
	const width, height = 40, 40
	pixels := make([]uint8, width*height)
	for i := range pixels {
		pixels[i] = 255
	}

	space, err := NewSpace(width, height, pixels)
	if err != nil {
		t.Fatalf("NewSpace() error = %v", err)
	}

	if !space.TryPlace(image.Rect(8, 8, 24, 24)) {
		t.Fatal("first TryPlace() = false, want true")
	}
	if space.TryPlace(image.Rect(20, 20, 30, 30)) {
		t.Fatal("TryPlace() = true for overlapping rectangle")
	}
	if !space.TryPlace(image.Rect(24, 8, 32, 24)) {
		t.Fatal("TryPlace() = false for rectangle touching an occupied edge")
	}
}

func TestSpaceRejectsOutOfBounds(t *testing.T) {
	const width, height = 10, 10
	pixels := make([]uint8, width*height)
	for i := range pixels {
		pixels[i] = 255
	}

	space, err := NewSpace(width, height, pixels)
	if err != nil {
		t.Fatalf("NewSpace() error = %v", err)
	}

	if space.TryPlace(image.Rect(-1, 0, 2, 2)) {
		t.Fatal("TryPlace() = true for out-of-bounds rectangle")
	}
}

func TestNewSpaceRejectsShortMask(t *testing.T) {
	if _, err := NewSpace(10, 10, make([]uint8, 99)); err == nil {
		t.Fatal("NewSpace() error = nil for undersized mask")
	}
}
