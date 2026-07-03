package mathutil

import "math"

// ShapeAwareStep performs one local step of a shape-aware Archimedean spiral.
// It blends the spiral tangent with the supplied distance-field gradient.
func ShapeAwareStep(x, y, angle, radiusGrowth, angleSpeed, gradientWeight float64,
	gradientX, gradientY float64) (newX, newY, newAngle float64) {

	const (
		baseRadius = 4.0
		stepLength = 1.8
	)

	t := angle
	r := baseRadius + t*radiusGrowth

	baseX := x + r*math.Cos(t)
	baseY := y + r*math.Sin(t)

	// Tangent of the Archimedean spiral r = baseRadius + radiusGrowth*t.
	tx := radiusGrowth*math.Cos(t) - r*math.Sin(t)
	ty := radiusGrowth*math.Sin(t) + r*math.Cos(t)
	tlen := math.Hypot(tx, ty)
	if tlen > 0 {
		tx /= tlen
		ty /= tlen
	}

	// Avoid amplifying numerical noise in nearly flat regions.
	gx, gy := gradientX, gradientY
	glen := math.Hypot(gx, gy)
	if glen > 0.001 {
		gx /= glen
		gy /= glen
	}

	dx := tx + gradientWeight*gx
	dy := ty + gradientWeight*gy
	dlen := math.Hypot(dx, dy)
	if dlen > 0 {
		dx /= dlen
		dy /= dlen
	}

	newX = baseX + dx*stepLength
	newY = baseY + dy*stepLength
	newAngle = t + angleSpeed*0.8

	return newX, newY, newAngle
}
