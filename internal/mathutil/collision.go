package mathutil

import "math"

// Collides returns true if two circles (center + radius) overlap.
func Collides(x1, y1, r1, x2, y2, r2 float64) bool {
	dx := x1 - x2
	dy := y1 - y2
	return math.Sqrt(dx*dx+dy*dy) < (r1 + r2)
}

// CollidesRect returns true if two axis-aligned rectangles overlap.
func CollidesRect(x1, y1, w1, h1, x2, y2, w2, h2 float64) bool {
	return !(x1+w1 <= x2 || x2+w2 <= x1 || y1+h1 <= y2 || y2+h2 <= y1)
}
