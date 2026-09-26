package mathutil

import (
	"image"
	"math"
)

// GenerateSpiralPosition returns a candidate center along the deterministic
// Archimedean spiral used by word placement. attempt zero returns center
// exactly; later attempts move progressively outward.
func GenerateSpiralPosition(center image.Point, attempt int) (x, y float64) {
	theta := float64(attempt) * 0.35
	r := 3.0 * float64(attempt) * 0.08

	x = float64(center.X) + r*math.Cos(theta)
	y = float64(center.Y) + r*math.Sin(theta)
	return x, y
}
