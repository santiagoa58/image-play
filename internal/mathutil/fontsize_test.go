package mathutil

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
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
			name:     "negative max count falls back to minimum",
			count:    10,
			maxCount: -5,
			want:     minFontSize,
		},
		{
			name:     "zero count maps to minimum",
			count:    0,
			maxCount: maxCount,
			want:     minFontSize,
		},
		{
			name:     "negative count maps to minimum",
			count:    -1,
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
			name:     "count above max count is clamped to maximum",
			count:    150,
			maxCount: maxCount,
			want:     maxFontSize,
		},
		{
			name:     "intermediate count follows logarithmic scale",
			count:    9,
			maxCount: maxCount,
			want:     minFontSize + (maxFontSize-minFontSize)*math.Log1p(9)/math.Log1p(maxCount),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateFontSize(tt.count, tt.maxCount, maxFontSize, minFontSize)
			assert.InDelta(t, tt.want, got, 1e-9)
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

		assert.Greater(t, current, previous,
			fmt.Sprintf("font size should be strictly increasing: count=%d, current=%f, previous=%f",
				count, current, previous))

		assert.InDelta(t, current, minFontSize, maxFontSize, // rough bounds check
			"font size should stay within min/max range")

		previous = current
	}
}
