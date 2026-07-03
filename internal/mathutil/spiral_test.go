package mathutil

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShapeAwareStep(t *testing.T) {
	tests := []struct {
		name           string
		x, y           float64
		angle          float64
		radiusGrowth   float64
		angleSpeed     float64
		gradientWeight float64
		gradientX      float64
		gradientY      float64
		wantX          float64
		wantY          float64
		wantAngle      float64
	}{
		{
			name:         "zero angle with no gradient follows spiral tangent",
			x:            10,
			y:            20,
			angle:        0,
			radiusGrowth: 0.7,
			angleSpeed:   1.2,
			wantX:        14.310284597154046,
			wantY:        21.773054840880267,
			wantAngle:    0.96,
		},
		{
			name:           "gradient is normalized before weighting",
			x:              0,
			y:              0,
			angle:          0,
			radiusGrowth:   0,
			angleSpeed:     0,
			gradientWeight: 2,
			gradientX:      3,
			gradientY:      4,
			wantX:          4.754304719431144,
			wantY:          1.634326892100813,
			wantAngle:      0,
		},
		{
			name:           "tiny gradient is not normalized",
			x:              0,
			y:              0,
			angle:          math.Pi / 2,
			radiusGrowth:   0,
			angleSpeed:     0.5,
			gradientWeight: 10,
			gradientX:      0.0003,
			gradientY:      0.0004,
			wantX:          -1.799985513384526,
			wantY:          4.007221606874161,
			wantAngle:      math.Pi/2 + 0.4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotX, gotY, gotAngle := ShapeAwareStep(
				tt.x,
				tt.y,
				tt.angle,
				tt.radiusGrowth,
				tt.angleSpeed,
				tt.gradientWeight,
				tt.gradientX,
				tt.gradientY,
			)
			assert.InDelta(t, tt.wantX, gotX, 1e-9)
			assert.InDelta(t, tt.wantY, gotY, 1e-9)
			assert.InDelta(t, tt.wantAngle, gotAngle, 1e-9)
		})
	}
}

// TestShapeAwareStep_Properties tests mathematical invariants
func TestShapeAwareStep_Properties(t *testing.T) {
	t.Parallel()
	for i := 0; i < 100; i++ {
		angle := float64(i) * 0.1
		_, _, newAngle := ShapeAwareStep(0, 0, angle, 0.5, 1.0, 0.3, 1, 0)
		if newAngle <= angle {
			t.Errorf("angle should always increase")
		}
	}
}
