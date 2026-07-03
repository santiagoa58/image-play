package mathutil

// Gradient approximates the gradient of a distance field at (x, y)
// using central finite differences.
func Gradient(distField func(x, y int) float32, x, y int) (float64, float64) {
	dx := (distField(x+1, y) - distField(x-1, y)) / 2
	dy := (distField(x, y+1) - distField(x, y-1)) / 2
	return float64(dx), float64(dy)
}
