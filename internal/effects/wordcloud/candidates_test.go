package wordcloud

import (
	"image"
	"reflect"
	"testing"

	"github.com/santiagoa58/image-play/internal/imageutil"
	"gocv.io/x/gocv"
)

type Peak struct {
	x, y  int
	depth float32
}

type TestCenter struct {
	name                string
	wantErr             bool
	nilMask             bool
	minCenterDepthRatio float64
	suppressionRadius   int
	peaks               []Peak
	want                []Center
}

func TestFindCenters(t *testing.T) {
	tests := []TestCenter{
		{
			name:    "returns an error for a nil mask",
			nilMask: true,
			wantErr: true,
		},
		{
			name: "returns separated peaks above the depth threshold",
			peaks: []Peak{
				{x: 1, y: 1, depth: 10},
				{x: 6, y: 6, depth: 8},
				{x: 4, y: 4, depth: 7},
			},
			minCenterDepthRatio: 0.8, // makes the min depth 80% of max depth. Here that's 80% of 10 = 8
			suppressionRadius:   2,
			want: []Center{
				{Point: image.Pt(1, 1), Depth: 10},
				{Point: image.Pt(6, 6), Depth: 8},
				// peak with depth 7 should be omitted because its depth is less than the min depth
			},
		},
		{
			name: "suppresses nearby peaks within suppression radius",
			peaks: []Peak{
				{x: 2, y: 2, depth: 10},
				{x: 3, y: 2, depth: 9},
				{x: 7, y: 7, depth: 8},
			},
			suppressionRadius:   2, // suppression radius of 2 means that peaks within radius of 2 will be suppressed
			minCenterDepthRatio: 0.8,
			want: []Center{
				{Point: image.Pt(2, 2), Depth: 10},
				{Point: image.Pt(7, 7), Depth: 8},
			},
		},
		{
			name:                "returns no centers for an empty distance map",
			minCenterDepthRatio: 0.01,
			suppressionRadius:   2,
			want:                []Center{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mask := makeTestMask(tc)
			if mask != nil {
				defer mask.Close()
			}

			cfg := NewConfig(
				WithMinCenterDepthRatio(tc.minCenterDepthRatio),
				WithCenterSuppressionRadius(tc.suppressionRadius),
			)

			got, err := FindCenters(mask, cfg)
			if tc.wantErr {
				if err == nil {
					t.Fatal("FindCenters() error = nil, want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("FindCenters() unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("FindCenters() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func makeTestMask(tc TestCenter) *imageutil.Mask {
	if tc.nilMask {
		return nil
	}

	const size = 9
	dist := gocv.NewMatWithSize(size, size, gocv.MatTypeCV32F)
	for _, peak := range tc.peaks {
		dist.SetFloatAt(peak.y, peak.x, peak.depth)
	}

	return &imageutil.Mask{
		Width:   size,
		Height:  size,
		DistMat: &dist,
	}
}
