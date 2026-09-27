package wordcloud

import (
	"image"
	"testing"

	"github.com/santiagoa58/image-play/internal/imageutil"
	"gocv.io/x/gocv"
)

func TestRegionsCoverDisconnectedShapeWithoutCrossingBackground(t *testing.T) {
	safe := gocv.NewMatWithSize(8, 14, gocv.MatTypeCV8UC1)
	defer safe.Close()
	for _, rect := range []image.Rectangle{
		image.Rect(1, 1, 5, 7), image.Rect(9, 1, 13, 7),
	} {
		part := safe.Region(rect)
		part.SetTo(gocv.NewScalar(255, 0, 0, 0))
		part.Close()
	}
	distance, err := imageutil.ComputeDistanceTransform(safe)
	if err != nil {
		t.Fatal(err)
	}
	defer distance.Close()
	regions, err := newRegionPolicy(safe, *distance)
	if err != nil {
		t.Fatal(err)
	}
	if regions.labels[2*14+6] != -1 {
		t.Error("background pixel was assigned a region")
	}
	left, right := regions.labels[3*14+2], regions.labels[3*14+10]
	if left < 0 || right < 0 || left == right {
		t.Fatalf("separate components have labels %d and %d", left, right)
	}
	area := 0
	for _, count := range regions.area {
		area += count
	}
	if area != gocv.CountNonZero(safe) {
		t.Fatalf("region area %d differs from silhouette area", area)
	}
}

func TestRegionChoiceBalancesCoverageBeforeOrientation(t *testing.T) {
	policy := &regionPolicy{
		width: 6, height: 2,
		labels: []int32{0, 0, 0, 1, 1, 1, 0, 0, 0, 1, 1, 1},
		area:   []int{6, 6}, occupied: []int{4, 0},
		depth: []float32{1, 2, 3, 4, 5, 6, 1, 2, 3, 4, 5, 6},
	}
	horizontal := gocv.NewMatWithSize(2, 6, gocv.MatTypeCV8UC1)
	defer horizontal.Close()
	horizontal.SetUCharAt(0, 2, 255) // only the fuller left region
	vertical := gocv.NewMatWithSize(2, 6, gocv.MatTypeCV8UC1)
	defer vertical.Close()
	vertical.SetUCharAt(0, 4, 255)
	vertical.SetUCharAt(0, 5, 255) // deepest point in the emptier right region

	point, angle, found, err := policy.choose([]orientedCenters{
		{angle: 0, centers: horizontal}, {angle: 90, centers: vertical},
	})
	if err != nil || !found || angle != 90 || point != image.Pt(5, 0) {
		t.Fatalf("choice = %v, %d, %v, %v; want right region vertically", point, angle, found, err)
	}
	horizontal.SetUCharAt(0, 3, 255)
	point, angle, found, err = policy.choose([]orientedCenters{
		{angle: 0, centers: horizontal}, {angle: 90, centers: vertical},
	})
	if err != nil || !found || angle != 0 || point != image.Pt(3, 0) {
		t.Fatalf("choice = %v, %d, %v, %v; want horizontal in right region", point, angle, found, err)
	}

	policy.reserve(image.Rect(2, 0, 5, 1))
	if policy.occupied[0] != 5 || policy.occupied[1] != 2 {
		t.Fatalf("cross-region reservation = %v, want [5 2]", policy.occupied)
	}
}
