package mathutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollides(t *testing.T) {
	tests := []struct {
		name       string
		x1, y1, r1 float64
		x2, y2, r2 float64
		want       bool
	}{
		{
			name: "overlapping circles collide",
			x1:   0, y1: 0, r1: 3,
			x2: 4, y2: 0, r2: 2,
			want: true,
		},
		{
			name: "separate circles do not collide",
			x1:   0, y1: 0, r1: 2,
			x2: 5.1, y2: 0, r2: 3,
			want: false,
		},
		{
			name: "externally tangent circles do not overlap",
			x1:   0, y1: 0, r1: 2,
			x2: 5, y2: 0, r2: 3,
			want: false,
		},
		{
			name: "same center with positive radii collide",
			x1:   10, y1: -3, r1: 1,
			x2: 10, y2: -3, r2: 4,
			want: true,
		},
		{
			name: "zero radius points at same center do not overlap",
			x1:   1, y1: 1, r1: 0,
			x2: 1, y2: 1, r2: 0,
			want: false,
		},
		{
			name: "zero radius point inside positive radius collides",
			x1:   1, y1: 1, r1: 0,
			x2: 1, y2: 1, r2: 2,
			want: true,
		},
		{
			name: "negative radius is invalid and does not collide",
			x1:   0, y1: 0, r1: -1,
			x2: 0, y2: 0, r2: 2,
			want: false,
		},
		{
			name: "diagonal distance uses pythagorean length",
			x1:   0, y1: 0, r1: 2.5,
			x2: 3, y2: 4, r2: 2.6,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Collides(tt.x1, tt.y1, tt.r1, tt.x2, tt.y2, tt.r2)
			assert.Equal(t, tt.want, got, "Collides()")

			// Verify symmetry (commutativity)
			reversed := Collides(tt.x2, tt.y2, tt.r2, tt.x1, tt.y1, tt.r1)
			assert.Equal(t, tt.want, reversed, "Collides() should be symmetric")
		})
	}
}

func TestCollidesRect(t *testing.T) {
	tests := []struct {
		name           string
		x1, y1, w1, h1 float64
		x2, y2, w2, h2 float64
		want           bool
	}{
		{
			name: "partially overlapping rectangles collide",
			x1:   0, y1: 0, w1: 10, h1: 10,
			x2: 8, y2: 2, w2: 5, h2: 5,
			want: true,
		},
		{
			name: "contained rectangle collides",
			x1:   0, y1: 0, w1: 10, h1: 10,
			x2: 2, y2: 3, w2: 4, h2: 5,
			want: true,
		},
		{
			name: "separated horizontally",
			x1:   0, y1: 0, w1: 4, h1: 4,
			x2: 5, y2: 0, w2: 4, h2: 4,
			want: false,
		},
		{
			name: "separated vertically",
			x1:   0, y1: 0, w1: 4, h1: 4,
			x2: 0, y2: 4.5, w2: 4, h2: 4,
			want: false,
		},
		{
			name: "touching vertical edges do not overlap",
			x1:   0, y1: 0, w1: 4, h1: 4,
			x2: 4, y2: 0, w2: 4, h2: 4,
			want: false,
		},
		{
			name: "touching horizontal edges do not overlap",
			x1:   0, y1: 0, w1: 4, h1: 4,
			x2: 0, y2: 4, w2: 4, h2: 4,
			want: false,
		},
		{
			name: "touching only at corner does not overlap",
			x1:   0, y1: 0, w1: 4, h1: 4,
			x2: 4, y2: 4, w2: 4, h2: 4,
			want: false,
		},
		{
			name: "negative coordinates overlap",
			x1:   -5, y1: -5, w1: 4, h1: 4,
			x2: -3, y2: -3, w2: 4, h2: 4,
			want: true,
		},
		{
			name: "zero width rectangle has no area to overlap",
			x1:   0, y1: 0, w1: 0, h1: 4,
			x2: 0, y2: 0, w2: 4, h2: 4,
			want: false,
		},
		{
			name: "zero height rectangle has no area to overlap",
			x1:   0, y1: 0, w1: 4, h1: 0,
			x2: 0, y2: 0, w2: 4, h2: 4,
			want: false,
		},
		{
			name: "negative width rectangle is invalid",
			x1:   10, y1: 0, w1: -2, h1: 4,
			x2: 5, y2: 0, w2: 10, h2: 4,
			want: false,
		},
		{
			name: "negative height rectangle is invalid",
			x1:   0, y1: 10, w1: 4, h1: -2,
			x2: 0, y2: 5, w2: 4, h2: 10,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CollidesRect(tt.x1, tt.y1, tt.w1, tt.h1, tt.x2, tt.y2, tt.w2, tt.h2)
			assert.Equal(t, tt.want, got, "CollidesRect()")

			// Verify symmetry
			reversed := CollidesRect(tt.x2, tt.y2, tt.w2, tt.h2, tt.x1, tt.y1, tt.w1, tt.h1)
			assert.Equal(t, tt.want, reversed, "CollidesRect() should be symmetric")
		})
	}
}
