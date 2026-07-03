package mathutil

import (
	"math"
	"testing"
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

			assertClose(t, "x", gotX, tt.wantX)
			assertClose(t, "y", gotY, tt.wantY)
			assertClose(t, "angle", gotAngle, tt.wantAngle)
		})
	}
}
