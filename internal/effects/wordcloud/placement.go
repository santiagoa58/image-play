package wordcloud

import (
	"fmt"
	"image"
	"image/color"

	"github.com/santiagoa58/image-play/internal/imageutil"
	"github.com/santiagoa58/image-play/internal/mathutil"
	"github.com/santiagoa58/image-play/internal/textutil"
	"gocv.io/x/gocv"
)

type PlacedWord struct {
	Word  textutil.Word
	X, Y  float64 // center position
	Color color.RGBA
}

// PlacementContext holds everything needed during placement.
// This reduces parameter passing and makes testing easier.
type PlacementContext struct {
	safeZone    *gocv.Mat
	occupancy   *gocv.Mat
	centers     []Center
	maxAttempts int
	wordPadding float64
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

	if len(ctx.centers) == 0 {
		return PlacedWord{}, false
	}

	boxW := word.Width + 2*ctx.wordPadding
	boxH := word.Height + 2*ctx.wordPadding

	for attempt := 0; attempt < ctx.maxAttempts; attempt++ {
		for _, center := range ctx.centers {
			cx, cy := mathutil.GenerateSpiralPosition(center.Point, attempt)
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
				Word: word,
				X:    cx,
				Y:    cy,
			}, true
		}
	}

	return PlacedWord{}, false

}

func NewPlacementContext(mask *imageutil.Mask, cfg Config) (*PlacementContext, error) {
	safe, occ, err := newValidationMasks(mask, cfg.SafeZoneErodeSize)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare masks for placement: %w", err)
	}
	centers, err := FindCenters(mask, cfg)
	if err != nil {
		safe.Close()
		occ.Close()
		return nil, fmt.Errorf("find placement centers: %w", err)
	}
	return &PlacementContext{
		safeZone:    safe,
		occupancy:   occ,
		centers:     centers,
		maxAttempts: cfg.SpiralStepsPerCenter,
		wordPadding: float64(cfg.WordPadding),
	}, nil
}

// newValidationMasks creates the fixed safe zone and mutable occupancy map
// used while placing words. The caller owns both returned matrices.
func newValidationMasks(
	mask *imageutil.Mask,
	erodeSize int,
) (*gocv.Mat, *gocv.Mat, error) {
	if mask == nil || mask.BinaryMat == nil || mask.BinaryMat.Empty() {
		return nil, nil, fmt.Errorf("binary placement mask is unavailable")
	}
	if erodeSize <= 0 || erodeSize%2 == 0 {
		return nil, nil, fmt.Errorf(
			"safe-zone erosion size must be positive and odd",
		)
	}

	safeZone := gocv.NewMat()
	kernel := gocv.GetStructuringElement(
		gocv.MorphRect,
		image.Point{X: erodeSize, Y: erodeSize},
	)
	defer kernel.Close()

	if err := gocv.Erode(
		*mask.BinaryMat,
		&safeZone,
		kernel,
	); err != nil {
		safeZone.Close()
		return nil, nil, fmt.Errorf("create safe zone: %w", err)
	}

	occupancy := gocv.NewMatWithSize(
		mask.BinaryMat.Rows(),
		mask.BinaryMat.Cols(),
		gocv.MatTypeCV8UC1,
	)

	return &safeZone, &occupancy, nil
}
