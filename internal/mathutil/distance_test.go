package mathutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGradient(t *testing.T) {
	tests := []struct {
		name   string
		field  func(x, y int) float32
		x, y   int
		wantDX float64
		wantDY float64
	}{
		{
			name: "constant field has zero gradient",
			field: func(x, y int) float32 {
				return 12
			},
			x: 9, y: -4,
			wantDX: 0, wantDY: 0,
		},
		{
			name: "linear field recovers exact slope",
			field: func(x, y int) float32 {
				return float32(3*x - 2*y + 7)
			},
			x: 5, y: 11,
			wantDX: 3, wantDY: -2,
		},
		{
			name: "quadratic field recovers central derivative at integer point",
			field: func(x, y int) float32 {
				return float32(x*x + 2*y*y)
			},
			x: 4, y: -3,
			wantDX: 8, wantDY: -12,
		},
		{
			name: "cross term gradient includes other coordinate",
			field: func(x, y int) float32 {
				return float32(x*y + y)
			},
			x: -6, y: 5,
			wantDX: 5, wantDY: -5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDX, gotDY := Gradient(tt.field, tt.x, tt.y)
			assert.InDelta(t, tt.wantDX, gotDX, 1e-9, "dx")
			assert.InDelta(t, tt.wantDY, gotDY, 1e-9, "dy")
		})
	}
}
