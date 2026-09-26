package mathutil

import (
	"image"
	"math"
	"testing"
)

func TestGenerateSpiralPositionStartsAtCenter(t *testing.T) {
	center := image.Pt(25, 40)
	x, y := GenerateSpiralPosition(center, 0)

	if x != 25 || y != 40 {
		t.Fatalf("attempt 0 = (%v,%v), want center (25,40)", x, y)
	}
}

func TestGenerateSpiralPositionMovesOutward(t *testing.T) {
	center := image.Pt(0, 0)
	x1, y1 := GenerateSpiralPosition(center, 10)
	x2, y2 := GenerateSpiralPosition(center, 100)

	r1 := math.Hypot(x1, y1)
	r2 := math.Hypot(x2, y2)
	if r2 <= r1 {
		t.Fatalf("radius at attempt 100 = %v, want greater than attempt 10 radius %v", r2, r1)
	}
}
