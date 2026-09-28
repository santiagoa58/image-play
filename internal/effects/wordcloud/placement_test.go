package wordcloud

import (
	"errors"
	"fmt"
	"image"
	"math"
	"testing"

	"github.com/fogleman/gg"
	"github.com/santiagoa58/image-play/internal/imageutil"
	"github.com/santiagoa58/image-play/internal/layout"
	"github.com/santiagoa58/image-play/internal/textutil"
	"gocv.io/x/gocv"
)

func TestPlaceFindsAPositionOutsideOldSearchCenters(t *testing.T) {
	ctx := placementContextForShape(t, image.Rect(70, 70, 92, 90), 0)
	word := textutil.Word{Text: "test", FontSize: 10, Width: 12, Height: 7}
	placed, err := ctx.Place(word, 10)
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
	placed, err := ctx.Place(word, 10)
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
	_, err := ctx.Place(word, 10)
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
	placed, err := ctx.Place(word, 6)
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

func TestLargeWordReservesGlyphWhileSmallWordReservesRectangle(t *testing.T) {
	for _, size := range []float64{24, 60} {
		t.Run(fmt.Sprintf("%.0fpx", size), func(t *testing.T) {
			ctx := placementContextForShape(t, image.Rect(0, 0, 100, 100), 1)
			word, err := textutil.MeasureWord("I", 1, wordcloudTestFontPath(t), size)
			if err != nil {
				t.Fatal(err)
			}
			placed, err := ctx.Place(word, size)
			if err != nil {
				t.Fatal(err)
			}
			if placed.Word.FontSize != size {
				t.Fatalf("font size = %v, want %v", placed.Word.FontSize, size)
			}
			occupied := gocv.CountNonZero(*ctx.occupancy)
			rect := ctx.footprint(word, placed.Angle)
			fit, err := ctx.fitFootprint(word, placed.Angle)
			if err != nil {
				t.Fatal(err)
			}
			regionCredit := 0
			for _, count := range ctx.regions.occupied {
				regionCredit += count
			}
			if regionCredit != fit.X*fit.Y {
				t.Fatalf("region credited %d pixels, want fit rectangle of %d", regionCredit, fit.X*fit.Y)
			}
			if size <= 24 && occupied != rect.X*rect.Y {
				t.Fatalf("small word occupied %d pixels, want rectangle of %d", occupied, rect.X*rect.Y)
			}
			if size > 24 && occupied >= rect.X*rect.Y {
				t.Fatalf("large word occupied %d pixels, want less than rectangle of %d", occupied, rect.X*rect.Y)
			}
		})
	}
}

func TestGlyphReservationCoversRenderedPixelsAtBothAngles(t *testing.T) {
	for _, angle := range []int{0, 90} {
		t.Run(fmt.Sprintf("%d-degrees", angle), func(t *testing.T) {
			ctx := placementContextForShape(t, image.Rect(0, 0, 100, 100), 1)
			ctx.angles = []int{angle}
			fontPath := wordcloudTestFontPath(t)
			word, err := textutil.MeasureWord("I", 1, fontPath, 60)
			if err != nil {
				t.Fatal(err)
			}
			placed, err := ctx.Place(word, 60)
			if err != nil {
				t.Fatal(err)
			}
			dc := gg.NewContext(100, 100)
			if err := dc.LoadFontFace(fontPath, 60); err != nil {
				t.Fatal(err)
			}
			dc.SetRGBA(1, 1, 1, 1)
			dc.Translate(placed.X, placed.Y)
			dc.Rotate(float64(angle) * math.Pi / 180)
			dc.DrawStringAnchored(word.Text, 0, 0, 0.5, 0.5)
			ink := dc.Image().(*image.RGBA)
			for y := 0; y < 100; y++ {
				for x := 0; x < 100; x++ {
					if ink.Pix[y*ink.Stride+x*4+3] != 0 && ctx.occupancy.GetUCharAt(y, x) == 0 {
						t.Fatalf("rendered glyph pixel (%d,%d) was not reserved", x, y)
					}
				}
			}
		})
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
	ctx, err := NewPlacementContext(mask, NewConfig(WithFontPath(wordcloudTestFontPath(t))))
	if err != nil {
		t.Fatal(err)
	}
	ctx.wordPadding = padding
	t.Cleanup(ctx.Close)
	return ctx
}
