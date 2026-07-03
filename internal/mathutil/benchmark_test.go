package mathutil

import "testing"

var (
	benchmarkBoolSink    bool
	benchmarkFloat64Sink float64
)

func BenchmarkCollides(b *testing.B) {
	b.ReportAllocs()

	var hit bool
	for i := 0; i < b.N; i++ {
		offset := float64(i & 7)
		hit = Collides(10+offset, 20, 12, 17, 24-offset, 9)
	}
	benchmarkBoolSink = hit
}

func BenchmarkCollidesRect(b *testing.B) {
	b.ReportAllocs()

	var hit bool
	for i := 0; i < b.N; i++ {
		offset := float64(i & 15)
		hit = CollidesRect(offset, 0, 20, 16, 14, 8-offset, 10, 12)
	}
	benchmarkBoolSink = hit
}

func BenchmarkCalculateFontSize(b *testing.B) {
	b.ReportAllocs()

	var size float64
	for i := 0; i < b.N; i++ {
		size = CalculateFontSize((i%250)+1, 250, 72, 10)
	}
	benchmarkFloat64Sink = size
}

func BenchmarkGradient(b *testing.B) {
	b.ReportAllocs()

	field := func(x, y int) float32 {
		return float32(3*x - 2*y + 7)
	}

	var dx, dy float64
	for i := 0; i < b.N; i++ {
		x := i & 127
		y := (i >> 7) & 127
		dx, dy = Gradient(field, x, y)
	}
	benchmarkFloat64Sink = dx + dy
}

func BenchmarkShapeAwareStep(b *testing.B) {
	b.ReportAllocs()

	var x, y, angle float64
	for i := 0; i < b.N; i++ {
		x, y, angle = ShapeAwareStep(
			x,
			y,
			angle,
			0.7,
			1.2,
			0.45,
			float64((i&7)-3),
			float64(((i>>3)&7)-3),
		)
	}
	benchmarkFloat64Sink = x + y + angle
}
