package wordcloud

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"

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

/**
 * Place attempts to place a single word using spirals from multiple centers.
 * It returns the placed word and whether placement was successful.
 * word - word to try and place
 * minFontsize - minimum fontsize
 * stepsize - value between 0 and 1, used to determine next attempted fontsize
 */
func (ctx *PlacementContext) Place(word textutil.Word, maxFontsize, minFontsize, stepsize float64) (PlacedWord, error) {
	if stepsize < 0 || stepsize > 1 {
		return PlacedWord{}, errors.New("stepsize must be a value between 0 and 1")
	}
	if len(ctx.centers) == 0 {
		return PlacedWord{}, errors.New("expected non-empty centers")
	}
	boxW := word.Width + 2*ctx.wordPadding
	boxH := word.Height + 2*ctx.wordPadding

	for f := math.Min(word.FontSize, maxFontsize); f > minFontsize; f -= math.Round(f * stepsize) {
		w, err := textutil.Resize(word, f)
		if err != nil {
			return PlacedWord{}, fmt.Errorf("failed to resize word: %w", err)
		}
		for attempt := range ctx.maxAttempts {
			for _, center := range ctx.centers {
				cx, cy := mathutil.GenerateSpiralPosition(center.Point, attempt+1)
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
					Word: w,
					X:    cx,
					Y:    cy,
				}, nil
			}
		}
	}
	return PlacedWord{}, errors.New("failed to place word")
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
