package wordcloud

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/santiagoa58/image-play/internal/imageutil"
	"github.com/santiagoa58/image-play/internal/layout"
	"github.com/santiagoa58/image-play/internal/mathutil"
	"github.com/santiagoa58/image-play/internal/textutil"
	"gocv.io/x/gocv"
)

var errNoPlacement = errors.New("no valid placement")

// PlacedWord is the layout result consumed by the renderer.
type PlacedWord struct {
	Word textutil.Word

	// X and Y are the center coordinates of the placed word.
	X, Y float64

	// Angle is the clockwise rotation in degrees. Current placement uses 0 or 90.
	Angle int

	// Color is reserved for per-word color styling. The current renderer draws
	// words in black.
	Color color.RGBA
}

// PlacementContext owns the mutable state for one placement run.
//
// Artistic search policy stays here: font-size fallback, preferred angles,
// candidate centers, and spiral traversal. Low-level mask containment and
// collision detection are delegated to layout.Space.
type PlacementContext struct {
	space       *layout.Space
	safeZone    *gocv.Mat
	occupancy   *gocv.Mat
	centers     []Center
	maxAttempts int
	wordPadding float64
	angles      []int
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

// Place attempts to place word, shrinking it until it fits or reaches
// minFontSize.
//
// maxFontSize also preserves the visual hierarchy: callers cap each word by
// the previously placed word's actual size, preventing a later, less-important
// word from becoming larger after an earlier word had to shrink.
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
	lastTried := word

	for fontSize >= minFontSize {
		resized, err := textutil.Resize(word, fontSize)
		if err != nil {
			return PlacedWord{}, fmt.Errorf("resize word: %w", err)
		}
		lastTried = resized

		if placed, ok := ctx.tryPlaceAtSize(resized); ok {
			return placed, nil
		}

		if fontSize == minFontSize {
			break
		}

		decrement := math.Max(1, math.Round(fontSize*stepRatio))
		fontSize = math.Max(minFontSize, fontSize-decrement)
	}

	return PlacedWord{}, fmt.Errorf(
		"%w at %.1fpx (%.1fx%.1f)",
		errNoPlacement,
		lastTried.FontSize,
		lastTried.Width,
		lastTried.Height,
	)
}

// tryPlaceAtSize searches every configured position for the preferred angle
// before falling back to the next angle.
//
// This intentionally gives horizontal text a global preference: all candidate
// positions are exhausted at 0 degrees before the 90-degree fallback begins.
// layout.Space.TryPlace performs and commits the geometry check atomically.
func (ctx *PlacementContext) tryPlaceAtSize(word textutil.Word) (PlacedWord, bool) {
	for _, angle := range ctx.angles {
		boxW := word.Width + 2*ctx.wordPadding
		boxH := word.Height + 2*ctx.wordPadding
		if angle == 90 {
			boxW, boxH = boxH, boxW
		}

		for attempt := range ctx.maxAttempts {
			for _, center := range ctx.centers {
				cx, cy := mathutil.GenerateSpiralPosition(center.Point, attempt)
				rect := mathutil.CenteredRect(cx, cy, boxW, boxH)

				if !ctx.space.TryPlace(rect) {
					continue
				}

				// Retain a bitmap occupancy map for optional debug output. Placement
				// itself is handled by layout.Space.
				occupiedROI := ctx.occupancy.Region(rect)
				occupiedROI.SetTo(gocv.NewScalar(255, 0, 0, 0))
				occupiedROI.Close()

				return PlacedWord{
					Word:  word,
					X:     cx,
					Y:     cy,
					Angle: angle,
				}, true
			}
		}
	}

	return PlacedWord{}, false
}

// NewPlacementContext builds the safe zone, fast layout index, search centers,
// and debug occupancy map used to place a complete word cloud.
//
// The returned context owns OpenCV matrices and must be closed.
func NewPlacementContext(mask *imageutil.Mask, cfg Config) (*PlacementContext, error) {
	if len(cfg.Angles) == 0 {
		return nil, errors.New("at least one placement angle is required")
	}
	for _, angle := range cfg.Angles {
		if angle != 0 && angle != 90 {
			return nil, fmt.Errorf("unsupported placement angle %d: only 0 and 90 are supported", angle)
		}
	}

	safe, occ, err := newValidationMasks(mask, cfg.SafeZoneErodeSize)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare masks for placement: %w", err)
	}

	safePixels, err := safe.DataPtrUint8()
	if err != nil {
		safe.Close()
		occ.Close()
		return nil, fmt.Errorf("read safe-zone pixels: %w", err)
	}
	space, err := layout.NewSpace(safe.Cols(), safe.Rows(), safePixels)
	if err != nil {
		safe.Close()
		occ.Close()
		return nil, fmt.Errorf("create layout space: %w", err)
	}

	centers, err := FindCenters(mask, cfg)
	if err != nil {
		safe.Close()
		occ.Close()
		return nil, fmt.Errorf("find placement centers: %w", err)
	}
	return &PlacementContext{
		space:       space,
		safeZone:    safe,
		occupancy:   occ,
		centers:     centers,
		maxAttempts: cfg.SpiralStepsPerCenter,
		wordPadding: float64(cfg.WordPadding),
		angles:      append([]int(nil), cfg.Angles...),
	}, nil
}

// newValidationMasks creates the eroded safe zone and an initially empty
// occupancy bitmap.
//
// The occupancy bitmap is retained for diagnostics only; layout.Space is the
// source of truth for collision detection. The caller owns both matrices.
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
