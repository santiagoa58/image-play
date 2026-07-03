package mathutil

import "math"

// ShapeAwareStep performs one step of a differential-style shape-aware spiral.
// It combines the spiral tangent with the distance field gradient.
func ShapeAwareStep(x, y, angle, radiusGrowth, angleSpeed, gradientWeight float64,
	gradientX, gradientY float64) (newX, newY, newAngle float64) {

	t := angle
	r := 4.0 + t*radiusGrowth

	baseX := x + r*math.Cos(t)
	baseY := y + r*math.Sin(t)

	// Tangent of the spiral
	tx := -math.Sin(t)
	ty := math.Cos(t)

	// Normalize gradient if possible
	gx, gy := gradientX, gradientY
	glen := math.Sqrt(gx*gx + gy*gy)
	if glen > 0.001 {
		gx /= glen
		gy /= glen
	}

	// Combined direction (differential idea)
	dx := tx + gradientWeight*gx
	dy := ty + gradientWeight*gy

	newX = baseX + dx*1.8
	newY = baseY + dy*1.8
	newAngle = t + angleSpeed*0.8

	return newX, newY, newAngle
}
