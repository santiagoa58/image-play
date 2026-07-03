package mathutil

import "math"

// Collides reports whether two circles strictly overlap.
// Negative radii are invalid, and tangent circles are not considered overlap.
func Collides(x1, y1, r1, x2, y2, r2 float64) bool {
	if r1 < 0 || r2 < 0 {
		return false
	}

	dx := x1 - x2
	dy := y1 - y2
	return math.Hypot(dx, dy) < (r1 + r2)
}

// CollidesRect reports whether two axis-aligned rectangles overlap.
// Non-positive sizes are invalid, and edge contact is not considered overlap.
func CollidesRect(x1, y1, w1, h1, x2, y2, w2, h2 float64) bool {
	if w1 <= 0 || h1 <= 0 || w2 <= 0 || h2 <= 0 {
		return false
	}

	return !(x1+w1 <= x2 || x2+w2 <= x1 || y1+h1 <= y2 || y2+h2 <= y1)
}
