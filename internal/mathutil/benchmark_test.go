package mathutil

import "testing"

var benchmarkFloat64Sink float64

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
