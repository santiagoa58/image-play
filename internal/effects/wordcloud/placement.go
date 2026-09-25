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

// Place attempts to place a word, shrinking it when its current size cannot fit.
func (ctx *PlacementContext) Place(
	word textutil.Word,
	maxFontSize, minFontSize, stepRatio float64,
) (PlacedWord, error) {
	if stepRatio <= 0 || stepRatio >= 1 {
		return PlacedWord{}, errors.New("step ratio must be between 0 and 1")
	}
	if len(ctx.centers) == 0 {
		return PlacedWord{}, errors.New("expected non-empty centers")
	}

	fontSize := math.Min(word.FontSize, maxFontSize)

	for fontSize >= minFontSize {
		resized, err := textutil.Resize(word, fontSize)
		if err != nil {
			return PlacedWord{}, fmt.Errorf("resize word: %w", err)
		}

		if placed, ok := ctx.tryPlaceAtSize(resized); ok {
			return placed, nil
		}

		if fontSize == minFontSize {
			break
		}

		decrement := math.Max(1, math.Round(fontSize*stepRatio))
		fontSize = math.Max(minFontSize, fontSize-decrement)
	}

	return PlacedWord{}, errors.New("failed to place word")
}

// tryPlaceAtSize searches the configured centers and spiral positions for a
// valid location for a word at one specific measured size.
func (ctx *PlacementContext) tryPlaceAtSize(word textutil.Word) (PlacedWord, bool) {
	boxW := word.Width + 2*ctx.wordPadding
	boxH := word.Height + 2*ctx.wordPadding

	for attempt := range ctx.maxAttempts {
		for _, center := range ctx.centers {
			cx, cy := mathutil.GenerateSpiralPosition(center.Point, attempt)
			rect := mathutil.CenteredRect(cx, cy, boxW, boxH)

			if !rect.In(image.Rect(0, 0, ctx.safeZone.Cols(), ctx.safeZone.Rows())) {
				continue
			}

			safeROI := ctx.safeZone.Region(rect)
			isSafe := gocv.CountNonZero(safeROI) == rect.Dx()*rect.Dy()
			safeROI.Close()
			if !isSafe {
				continue
			}

			occROI := ctx.occupancy.Region(rect)
			isFree := gocv.CountNonZero(occROI) == 0
			occROI.Close()
			if !isFree {
				continue
			}

			occROI := ctx.occupancy.Region(rect)
			occROI.SetTo(gocv.NewScalar(255, 0, 0, 0))
			occROI.Close()

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
