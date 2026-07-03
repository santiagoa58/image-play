package mathutil

import (
	"math"
	"testing"
)

func TestCalculateFontSize(t *testing.T) {
	const (
		minFontSize = 12.0
		maxFontSize = 72.0
		maxCount    = 100.0
	)

	tests := []struct {
		name     string
		count    int
		maxCount float64
		want     float64
	}{
		{
			name:     "zero max count falls back to minimum",
			count:    10,
			maxCount: 0,
			want:     minFontSize,
		},
		{
			name:     "zero count maps to minimum",
			count:    0,
			maxCount: maxCount,
			want:     minFontSize,
		},
		{
			name:     "max count maps to maximum",
			count:    int(maxCount),
			maxCount: maxCount,
			want:     maxFontSize,
		},
		{
			name:     "intermediate count follows logarithmic scale",
			count:    9,
			maxCount: maxCount,
			want:     minFontSize + (maxFontSize-minFontSize)*math.Log(10)/math.Log(101),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateFontSize(tt.count, tt.maxCount, maxFontSize, minFontSize)
			assertClose(t, "font size", got, tt.want)
		})
	}
}

func TestCalculateFontSizeIsMonotonic(t *testing.T) {
	const (
		minFontSize = 8.0
		maxFontSize = 64.0
		maxCount    = 250.0
	)

	previous := CalculateFontSize(0, maxCount, maxFontSize, minFontSize)
	for count := 1; count <= int(maxCount); count++ {
		current := CalculateFontSize(count, maxCount, maxFontSize, minFontSize)
		if current <= previous {
			t.Fatalf("font size for count %d = %f; want greater than previous %f", count, current, previous)
		}
		if current < minFontSize || current > maxFontSize {
			t.Fatalf("font size for count %d = %f; want within [%f, %f]", count, current, minFontSize, maxFontSize)
		}
		previous = current
	}
}
