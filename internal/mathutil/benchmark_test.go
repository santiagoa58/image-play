package mathutil

import "testing"

var (
	benchmarkBoolSink    bool
	benchmarkFloat64Sink float64
)

func BenchmarkCollides(b *testing.B) {
	b.ReportAllocs()

	var hit bool
	for i := 0; b.Loop(); i++ {
		offset := float64(i & 7)
		hit = Collides(10+offset, 20, 12, 17, 24-offset, 9)
	}
	benchmarkBoolSink = hit
}

func BenchmarkCollidesRect(b *testing.B) {
	b.ReportAllocs()

	var hit bool
	for i := 0; b.Loop(); i++ {
		offset := float64(i & 15)
		hit = CollidesRect(offset, 0, 20, 16, 14, 8-offset, 10, 12)
	}
	benchmarkBoolSink = hit
}

func BenchmarkScaleLog(b *testing.B) {
	b.ReportAllocs()

	input := Range{Min: 1, Max: 250}
	output := Range{Min: 10, Max: 72}
	var size float64
	for i := 0; b.Loop(); i++ {
		size = ScaleLog(float64((i%250)+1), input, output)
	}
	benchmarkFloat64Sink = size
}

func BenchmarkGradient(b *testing.B) {
	b.ReportAllocs()

	field := func(x, y int) float32 {
		return float32(3*x - 2*y + 7)
	}

	var dx, dy float64
	for i := 0; b.Loop(); i++ {
		x := i & 127
		y := (i >> 7) & 127
		dx, dy = Gradient(field, x, y)
	}
	benchmarkFloat64Sink = dx + dy
}
