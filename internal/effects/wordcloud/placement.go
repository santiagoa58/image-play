package wordcloud

import (
	"errors"
	"fmt"
	"image"
	"math"

	"github.com/santiagoa58/image-play/internal/imageutil"
	"github.com/santiagoa58/image-play/internal/layout"
	"github.com/santiagoa58/image-play/internal/textutil"
	"gocv.io/x/gocv"
)

var errNoPlacement = errors.New("no valid placement")

// PlacedWord is the layout result consumed by the renderer.
type PlacedWord struct {
	Word  textutil.Word
	X, Y  float64
	Angle int
}

// PlacementContext owns the mutable state of one word-cloud layout.
type PlacementContext struct {
	space       *layout.FreeSpace
	safeZone    *gocv.Mat
	occupancy   *gocv.Mat
	depth       *gocv.Mat
	wordPadding int
	angles      []int
}

func (ctx *PlacementContext) Close() {
	if ctx.space != nil {
		ctx.space.Close()
	}
	if ctx.safeZone != nil {
		ctx.safeZone.Close()
	}
	if ctx.occupancy != nil {
		ctx.occupancy.Close()
	}
}

// Place selects the largest fitting whole-pixel font size at or above the
// minimum. A failed placement means no footprint fits at the minimum size.
func (ctx *PlacementContext) Place(word textutil.Word, maxFontSize, minFontSize float64) (PlacedWord, error) {
	minimum := int(math.Ceil(minFontSize))
	desired := int(math.Floor(math.Min(word.FontSize, maxFontSize)))
	if minimum <= 0 || desired < minimum {
		return PlacedWord{}, errors.New("invalid font-size range")
	}

	atDesired, err := textutil.Resize(word, float64(desired))
	if err != nil {
		return PlacedWord{}, fmt.Errorf("measure desired word: %w", err)
	}
	if placed, ok, err := ctx.tryPlaceAtSize(atDesired); err != nil || ok {
		return placed, err
	}

	atMinimum, err := textutil.Resize(word, float64(minimum))
	if err != nil {
		return PlacedWord{}, fmt.Errorf("measure minimum word: %w", err)
	}
	fits, err := ctx.fits(atMinimum)
	if err != nil {
		return PlacedWord{}, err
	}
	if !fits {
		return PlacedWord{}, fmt.Errorf("%w at %dpx", errNoPlacement, minimum)
	}

	low, high := minimum, desired-1
	for low < high {
		mid := low + (high-low+1)/2
		measured, err := textutil.Resize(word, float64(mid))
		if err != nil {
			return PlacedWord{}, fmt.Errorf("measure word at %dpx: %w", mid, err)
		}
		fits, err := ctx.fits(measured)
		if err != nil {
			return PlacedWord{}, err
		}
		if fits {
			low = mid
		} else {
			high = mid - 1
		}
	}
	measured, err := textutil.Resize(word, float64(low))
	if err != nil {
		return PlacedWord{}, fmt.Errorf("measure fitting word: %w", err)
	}
	placed, ok, err := ctx.tryPlaceAtSize(measured)
	if err != nil {
		return PlacedWord{}, err
	}
	if !ok {
		return PlacedWord{}, errors.New("fitting size became unavailable before reservation")
	}
	return placed, nil
}

// fits checks the complete free-space mask, without changing it.
func (ctx *PlacementContext) fits(word textutil.Word) (bool, error) {
	for _, angle := range ctx.angles {
		centers, err := ctx.space.ValidCenters(ctx.footprint(word, angle))
		if err != nil {
			return false, err
		}
		found := gocv.CountNonZero(centers) > 0
		centers.Close()
		if found {
			return true, nil
		}
	}
	return false, nil
}

func (ctx *PlacementContext) tryPlaceAtSize(word textutil.Word) (PlacedWord, bool, error) {
	for _, angle := range ctx.angles {
		size := ctx.footprint(word, angle)
		centers, err := ctx.space.ValidCenters(size)
		if err != nil {
			return PlacedWord{}, false, err
		}
		if gocv.CountNonZero(centers) == 0 {
			centers.Close()
			continue
		}
		_, _, _, center := gocv.MinMaxLocWithMask(*ctx.depth, centers)
		centers.Close()
		if !ctx.space.Reserve(size, center) {
			return PlacedWord{}, false, errors.New("selected center could not be reserved")
		}
		rect := layout.RectAt(center, size)
		occupied := ctx.occupancy.Region(rect)
		occupied.SetTo(gocv.NewScalar(255, 0, 0, 0))
		occupied.Close()
		return PlacedWord{Word: word, X: float64(center.X), Y: float64(center.Y), Angle: angle}, true, nil
	}
	return PlacedWord{}, false, nil
}

func (ctx *PlacementContext) footprint(word textutil.Word, angle int) image.Point {
	w := int(math.Ceil(word.Width + 2*float64(ctx.wordPadding)))
	h := int(math.Ceil(word.Height + 2*float64(ctx.wordPadding)))
	if angle == 90 {
		w, h = h, w
	}
	return image.Pt(max(1, w), max(1, h))
}

func NewPlacementContext(mask *imageutil.Mask, cfg Config) (*PlacementContext, error) {
	if len(cfg.Angles) == 0 {
		return nil, errors.New("at least one placement angle is required")
	}
	for _, angle := range cfg.Angles {
		if angle != 0 && angle != 90 {
			return nil, fmt.Errorf("unsupported placement angle %d: only 0 and 90 are supported", angle)
		}
	}
	if mask == nil || mask.DistMat == nil || mask.DistMat.Empty() {
		return nil, errors.New("distance map is unavailable")
	}

	safe, occ, err := newValidationMasks(mask, cfg.SafeZoneErodeSize)
	if err != nil {
		return nil, fmt.Errorf("prepare placement masks: %w", err)
	}
	space, err := layout.NewFreeSpace(*safe)
	if err != nil {
		safe.Close()
		occ.Close()
		return nil, fmt.Errorf("create free space: %w", err)
	}
	return &PlacementContext{
		space: space, safeZone: safe, occupancy: occ, depth: mask.DistMat,
		wordPadding: cfg.WordPadding, angles: append([]int(nil), cfg.Angles...),
	}, nil
}

// newValidationMasks preserves the existing silhouette safety margin and
// creates an occupancy bitmap for optional diagnostics.
func newValidationMasks(mask *imageutil.Mask, erodeSize int) (*gocv.Mat, *gocv.Mat, error) {
	if mask == nil || mask.BinaryMat == nil || mask.BinaryMat.Empty() {
		return nil, nil, errors.New("binary placement mask is unavailable")
	}
	if erodeSize <= 0 || erodeSize%2 == 0 {
		return nil, nil, errors.New("safe-zone erosion size must be positive and odd")
	}
	safe := gocv.NewMat()
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(erodeSize, erodeSize))
	defer kernel.Close()
	if err := gocv.Erode(*mask.BinaryMat, &safe, kernel); err != nil {
		safe.Close()
		return nil, nil, fmt.Errorf("create safe zone: %w", err)
	}
	occ := gocv.NewMatWithSize(safe.Rows(), safe.Cols(), gocv.MatTypeCV8UC1)
	return &safe, &occ, nil
}
