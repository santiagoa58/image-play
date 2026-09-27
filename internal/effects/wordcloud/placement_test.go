package wordcloud

import (
	"errors"
	"image"
	"testing"

	"github.com/santiagoa58/image-play/internal/imageutil"
	"github.com/santiagoa58/image-play/internal/layout"
	"github.com/santiagoa58/image-play/internal/textutil"
	"gocv.io/x/gocv"
)

func TestPlaceFindsAPositionOutsideOldSearchCenters(t *testing.T) {
	ctx := placementContextForShape(t, image.Rect(70, 70, 92, 90), 0)
	word := textutil.Word{Text: "test", FontSize: 10, Width: 12, Height: 7}
	placed, err := ctx.Place(word, 10, 10)
	if err != nil {
		t.Fatal(err)
	}
	if !layout.RectAt(image.Pt(int(placed.X), int(placed.Y)), ctx.footprint(word, placed.Angle)).In(image.Rect(70, 70, 92, 90)) {
		t.Fatalf("word placed outside the only available region: %+v", placed)
	}
}

func TestPlaceUsesVerticalWhenHorizontalCannotFit(t *testing.T) {
	ctx := placementContextForShape(t, image.Rect(46, 20, 54, 80), 0)
	word := textutil.Word{Text: "test", FontSize: 10, Width: 30, Height: 6}
	placed, err := ctx.Place(word, 10, 10)
	if err != nil {
		t.Fatal(err)
	}
	if placed.Angle != 90 {
		t.Fatalf("angle = %d, want vertical", placed.Angle)
	}
}

func TestPlaceSkipsOnlyWhenMinimumCannotFit(t *testing.T) {
	ctx := placementContextForShape(t, image.Rect(40, 40, 45, 45), 0)
	word := textutil.Word{Text: "test", FontSize: 10, Width: 30, Height: 8}
	_, err := ctx.Place(word, 10, 10)
	if !errors.Is(err, errNoPlacement) {
		t.Fatalf("error = %v, want errNoPlacement", err)
	}
}

func TestPlaceFindsLargestFittingSize(t *testing.T) {
	ctx := placementContextForShape(t, image.Rect(20, 30, 80, 70), 1)
	word, err := textutil.MeasureWord("example", 10, wordcloudTestFontPath(t), 80)
	if err != nil {
		t.Fatal(err)
	}
	placed, err := ctx.Place(word, 80, 6)
	if err != nil {
		t.Fatal(err)
	}
	if placed.Word.FontSize >= 80 || placed.Word.FontSize < 6 {
		t.Fatalf("size = %v, expected a smaller fitting size", placed.Word.FontSize)
	}
	// Query a fresh copy of the shape: placement has already reserved space in ctx.
	fresh := placementContextForShape(t, image.Rect(20, 30, 80, 70), 1)
	next, err := textutil.Resize(word, placed.Word.FontSize+1)
	if err != nil {
		t.Fatal(err)
	}
	fits, err := fresh.fits(next)
	if err != nil {
		t.Fatal(err)
	}
	if fits {
		t.Fatalf("size %v also fits; selected %v", next.FontSize, placed.Word.FontSize)
	}
}

func TestFootprintAccountsForPaddingAndRotation(t *testing.T) {
	ctx := placementContextForShape(t, image.Rect(0, 0, 100, 100), 3)
	word := textutil.Word{Width: 20, Height: 10}
	if got := ctx.footprint(word, 0); got != image.Pt(26, 16) {
		t.Errorf("horizontal footprint = %v, want (26,16)", got)
	}
	if got := ctx.footprint(word, 90); got != image.Pt(16, 26) {
		t.Errorf("vertical footprint = %v, want (16,26)", got)
	}
}

func TestSafeZoneErodeSizeChangesSafeZoneArea(t *testing.T) {
	binary := gocv.NewMatWithSize(100, 100, gocv.MatTypeCV8UC1)
	defer binary.Close()
	shape := binary.Region(image.Rect(10, 10, 90, 90))
	shape.SetTo(gocv.NewScalar(255, 0, 0, 0))
	shape.Close()
	mask := &imageutil.Mask{BinaryMat: &binary}

	safe3, occ3, err := newValidationMasks(mask, 3)
	if err != nil {
		t.Fatal(err)
	}
	defer safe3.Close()
	defer occ3.Close()
	safe7, occ7, err := newValidationMasks(mask, 7)
	if err != nil {
		t.Fatal(err)
	}
	defer safe7.Close()
	defer occ7.Close()
	if gocv.CountNonZero(*safe7) >= gocv.CountNonZero(*safe3) {
		t.Fatal("a larger safe-zone margin should leave less room")
	}
}

func placementContextForShape(t *testing.T, rect image.Rectangle, padding int) *PlacementContext {
	t.Helper()
	binary := gocv.NewMatWithSize(100, 100, gocv.MatTypeCV8UC1)
	shape := binary.Region(rect)
	shape.SetTo(gocv.NewScalar(255, 0, 0, 0))
	shape.Close()
	depth, err := imageutil.ComputeDistanceTransform(binary)
	if err != nil {
		binary.Close()
		t.Fatal(err)
	}
	mask := &imageutil.Mask{BinaryMat: &binary, DistMat: depth}
	t.Cleanup(mask.Close)
	ctx, err := NewPlacementContext(mask, NewConfig())
	if err != nil {
		t.Fatal(err)
	}
	ctx.wordPadding = padding
	t.Cleanup(ctx.Close)
	return ctx
}

func TestCandidateCanGrowAfterAnAwkwardWordShrinks(t *testing.T) {
	ctx := placementContextForShape(t, image.Rect(10, 10, 90, 90), 1)
	font := wordcloudTestFontPath(t)
	long, err := textutil.MeasureWord("extraordinarilylong", 20, font, 40)
	if err != nil {
		t.Fatal(err)
	}
	first, err := placeCandidate(ctx, long, 6)
	if err != nil {
		t.Fatal(err)
	}
	short, err := textutil.MeasureWord("I", 10, font, 35)
	if err != nil {
		t.Fatal(err)
	}
	second, err := placeCandidate(ctx, short, 6)
	if err != nil {
		t.Fatal(err)
	}
	if first.Word.FontSize >= 35 {
		t.Fatalf("long word did not establish shrinkage: %v", first.Word.FontSize)
	}
	if second.Word.FontSize != 35 {
		t.Fatalf("short word inherited shrinkage: %v, want 35", second.Word.FontSize)
	}
}
