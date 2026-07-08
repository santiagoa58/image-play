package wordcloud

import (
	"fmt"
	"image"

	"github.com/santiagoa58/image-play/internal/imageutil"
	"github.com/santiagoa58/image-play/internal/mathutil"
	"github.com/santiagoa58/image-play/internal/textutil"
	"gocv.io/x/gocv"
)

type PlacedWord struct {
	Word string
	X, Y float64 // center position
	Size float64 // font size in points
}

// PlacementContext holds everything needed during placement.
// This reduces parameter passing and makes testing easier.
type PlacementContext struct {
	safeZone  *gocv.Mat
	occupancy *gocv.Mat
	centers   []image.Point
	FontPath  string
	Padding   float64
}

// Close releases the gocv resources held by the context.
func (ctx *PlacementContext) Close() {
	if ctx.safeZone != nil {
		ctx.safeZone.Close()
	}
	if ctx.occupancy != nil {
		ctx.occupancy.Close()
	}
}

// tryPlaceWord attempts to place a single word using spirals from multiple centers.
// It returns the placed word and whether placement was successful.
func (ctx *PlacementContext) TryPlace(word textutil.Word) (PlacedWord, bool) {

	textW, textH, err := textutil.MeasureWord(word.Text, ctx.FontPath, word.Size)
	if err != nil {
		return PlacedWord{}, false
	}

	boxW := textW + ctx.Padding*2
	boxH := textH + ctx.Padding*2

	for _, center := range ctx.centers {
		for attempt := range 5000 {
			cx, cy := mathutil.GenerateSpiralPosition(center, attempt)
			rect := mathutil.CenteredRect(cx, cy, boxW, boxH)

			if !rect.In(image.Rect(0, 0, ctx.safeZone.Cols(), ctx.safeZone.Rows())) {
				continue
			}

			// Check safe zone
			safeROI := ctx.safeZone.Region(rect)
			isSafe := gocv.CountNonZero(safeROI) == rect.Dx()*rect.Dy()
			safeROI.Close()
			if !isSafe {
				continue
			}

			// Check occupancy
			occROI := ctx.occupancy.Region(rect)
			isFree := gocv.CountNonZero(occROI) == 0
			occROI.Close()
			if !isFree {
				continue
			}

			// Valid position found — mark as occupied
			occROI2 := ctx.occupancy.Region(rect)
			occROI2.SetTo(gocv.NewScalar(255, 0, 0, 0))
			occROI2.Close()

			return PlacedWord{
				Word: word.Text,
				X:    cx,
				Y:    cy,
				Size: word.Size,
			}, true
		}
	}

	return PlacedWord{}, false

}

func NewPlacementContext(mask *imageutil.Mask, fontpath string) (PlacementContext, error) {
	const defaultPadding = 5
	safe, occ, err := imageutil.GetValidationMask(*mask, defaultPadding)
	if err != nil {
		return PlacementContext{}, fmt.Errorf("failed to prepare masks for placement: %w", err)
	}
	centers := imageutil.FindCenters(*mask)
	return PlacementContext{
		safeZone:  safe,
		occupancy: occ,
		centers:   centers,
		Padding:   defaultPadding,
		FontPath:  fontpath,
	}, nil
}
