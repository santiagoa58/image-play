package mathutil

// Gradient approximates the gradient of a scalar grid field at (x, y).
// It uses central differences and leaves boundary behavior to distField.
func Gradient(distField func(x, y int) float32, x, y int) (float64, float64) {
	dx := (float64(distField(x+1, y)) - float64(distField(x-1, y))) * 0.5
	dy := (float64(distField(x, y+1)) - float64(distField(x, y-1))) * 0.5
	return dx, dy
}
