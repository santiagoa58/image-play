package wordcloud

import (
	"image"
	"testing"

	"github.com/santiagoa58/image-play/internal/imageutil"
	"github.com/santiagoa58/image-play/internal/textutil"
	"gocv.io/x/gocv"
)

func TestRectanglePlacementSucceedsInsideSafeZone(t *testing.T) {
	ctx := newPlacementTestContext(
		t,
		image.Rect(0, 0, 100, 100),
		[]image.Point{image.Pt(50, 50)},
		0,
	)

	placed, ok := ctx.TryPlace(testWord(20, 10))
	if !ok {
		t.Fatal("TryPlace() = false, want true")
	}
	if placed.X != 50 || placed.Y != 50 {
		t.Errorf(
			"placed center = (%v,%v), want (50,50)",
			placed.X,
			placed.Y,
		)
	}
}

func TestRectanglePlacementRejectsSafeZoneBoundaryCrossing(t *testing.T) {
	ctx := newPlacementTestContext(
		t,
		image.Rect(20, 20, 80, 80),
		[]image.Point{image.Pt(25, 50)},
		0,
	)

	if _, ok := ctx.TryPlace(testWord(20, 10)); ok {
		t.Fatal("TryPlace() = true for a word crossing the safe-zone boundary")
	}
}

func TestRectanglePlacementRejectsImageBoundsCrossing(t *testing.T) {
	ctx := newPlacementTestContext(
		t,
		image.Rect(0, 0, 100, 100),
		[]image.Point{image.Pt(5, 50)},
		0,
	)

	if _, ok := ctx.TryPlace(testWord(20, 10)); ok {
		t.Fatal("TryPlace() = true for a word crossing the image bounds")
	}
}

func TestRectanglePlacementRejectsOccupiedCenter(t *testing.T) {
	ctx := newPlacementTestContext(
		t,
		image.Rect(0, 0, 100, 100),
		[]image.Point{image.Pt(50, 50)},
		0,
	)
	word := testWord(20, 10)

	if _, ok := ctx.TryPlace(word); !ok {
		t.Fatal("first TryPlace() = false, want true")
	}
	if _, ok := ctx.TryPlace(word); ok {
		t.Fatal("second TryPlace() = true at an occupied center")
	}
}

func TestRectanglePlacementUsesSeparatedCenters(t *testing.T) {
	ctx := newPlacementTestContext(
		t,
		image.Rect(0, 0, 100, 100),
		[]image.Point{
			image.Pt(30, 50),
			image.Pt(70, 50),
		},
		0,
	)
	word := testWord(20, 10)

	first, ok := ctx.TryPlace(word)
	if !ok {
		t.Fatal("first TryPlace() = false, want true")
	}
	second, ok := ctx.TryPlace(word)
	if !ok {
		t.Fatal("second TryPlace() = false, want true")
	}

	if first.X != 30 || first.Y != 50 {
		t.Errorf(
			"first center = (%v,%v), want (30,50)",
			first.X,
			first.Y,
		)
	}
	if second.X != 70 || second.Y != 50 {
		t.Errorf(
			"second center = (%v,%v), want (70,50)",
			second.X,
			second.Y,
		)
	}
}

func TestRectanglePlacementPaddingIncreasesOccupiedArea(t *testing.T) {
	withoutPadding := newPlacementTestContext(
		t,
		image.Rect(0, 0, 100, 100),
		[]image.Point{image.Pt(50, 50)},
		0,
	)
	withPadding := newPlacementTestContext(
		t,
		image.Rect(0, 0, 100, 100),
		[]image.Point{image.Pt(50, 50)},
		3,
	)
	word := testWord(20, 10)

	if _, ok := withoutPadding.TryPlace(word); !ok {
		t.Fatal("TryPlace() without padding = false, want true")
	}
	if _, ok := withPadding.TryPlace(word); !ok {
		t.Fatal("TryPlace() with padding = false, want true")
	}

	withoutArea := gocv.CountNonZero(*withoutPadding.occupancy)
	withArea := gocv.CountNonZero(*withPadding.occupancy)

	if withoutArea != 20*10 {
		t.Errorf("occupied area without padding = %d, want 200", withoutArea)
	}
	if withArea != 26*16 {
		t.Errorf("occupied area with padding = %d, want 416", withArea)
	}
	if withArea <= withoutArea {
		t.Errorf(
			"occupied area with padding = %d, want greater than %d",
			withArea,
			withoutArea,
		)
	}
}

func TestSafeZoneErodeSizeChangesSafeZoneArea(t *testing.T) {
	const size = 100

	binary := gocv.NewMatWithSize(
		size,
		size,
		gocv.MatTypeCV8UC1,
	)
	defer binary.Close()

	shape := binary.Region(image.Rect(10, 10, 90, 90))
	shape.SetTo(gocv.NewScalar(255, 0, 0, 0))
	shape.Close()

	mask := &imageutil.Mask{BinaryMat: &binary}

	safe3, occupancy3, err := newValidationMasks(mask, 3)
	if err != nil {
		t.Fatalf("newValidationMasks(..., 3) error = %v", err)
	}
	defer safe3.Close()
	defer occupancy3.Close()

	safe7, occupancy7, err := newValidationMasks(mask, 7)
	if err != nil {
		t.Fatalf("newValidationMasks(..., 7) error = %v", err)
	}
	defer safe7.Close()
	defer occupancy7.Close()

	area3 := gocv.CountNonZero(*safe3)
	area7 := gocv.CountNonZero(*safe7)

	if area7 >= area3 {
		t.Errorf(
			"safe-zone area with 7x7 erosion = %d, want less than 3x3 area %d",
			area7,
			area3,
		)
	}
}

func newPlacementTestContext(
	t *testing.T,
	safeRect image.Rectangle,
	points []image.Point,
	padding float64,
) *PlacementContext {
	t.Helper()

	const size = 100

	safeZone := gocv.NewMatWithSize(
		size,
		size,
		gocv.MatTypeCV8UC1,
	)
	safeROI := safeZone.Region(safeRect)
	safeROI.SetTo(gocv.NewScalar(255, 0, 0, 0))
	safeROI.Close()

	occupancy := gocv.NewMatWithSize(
		size,
		size,
		gocv.MatTypeCV8UC1,
	)

	centers := make([]Center, len(points))
	for i, point := range points {
		centers[i] = Center{Point: point}
	}

	ctx := &PlacementContext{
		safeZone:    &safeZone,
		occupancy:   &occupancy,
		centers:     centers,
		maxAttempts: 1,
		wordPadding: padding,
	}
	t.Cleanup(ctx.Close)

	return ctx
}

func testWord(width, height float64) textutil.Word {
	return textutil.Word{
		Text:   "test",
		Width:  width,
		Height: height,
	}
}
