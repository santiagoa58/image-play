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
	regions     *regionPolicy
	fontPath    string
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
func (ctx *PlacementContext) Place(word textutil.Word, minFontSize float64) (PlacedWord, error) {
	minimum := int(math.Ceil(minFontSize))
	desired := int(math.Floor(word.FontSize))
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

	fittingWord, err := ctx.findSmallerFittingWord(word, minimum, desired-1)
	if err != nil {
		return PlacedWord{}, err
	}
	placed, ok, err := ctx.tryPlaceAtSize(fittingWord)
	if err != nil {
		return PlacedWord{}, err
	}
	if !ok {
		return PlacedWord{}, errors.New("fitting size became unavailable before reservation")
	}
	return placed, nil
}

// findSmallerFittingWord searches without reserving space. The desired size
// has already failed, so first establish whether the minimum can fit.
func (ctx *PlacementContext) findSmallerFittingWord(word textutil.Word, minimum, maximum int) (textutil.Word, error) {
	atMinimum, err := textutil.Resize(word, float64(minimum))
	if err != nil {
		return textutil.Word{}, fmt.Errorf("measure minimum word: %w", err)
	}
	fits, err := ctx.fits(atMinimum)
	if err != nil {
		return textutil.Word{}, err
	}
	if !fits {
		return textutil.Word{}, fmt.Errorf("%w at %dpx", errNoPlacement, minimum)
	}

	fontSize, err := ctx.largestFittingFontSize(word, minimum, maximum)
	if err != nil {
		return textutil.Word{}, err
	}
	fittingWord, err := textutil.Resize(word, float64(fontSize))
	if err != nil {
		return textutil.Word{}, fmt.Errorf("measure fitting word: %w", err)
	}
	return fittingWord, nil
}

// largestFittingFontSize uses binary search with a minimum already known to fit.
func (ctx *PlacementContext) largestFittingFontSize(word textutil.Word, minimum, maximum int) (int, error) {
	low, high := minimum, maximum
	for low < high {
		candidateSize := low + (high-low+1)/2
		candidateWord, err := textutil.Resize(word, float64(candidateSize))
		if err != nil {
			return 0, fmt.Errorf("measure word at %dpx: %w", candidateSize, err)
		}
		fits, err := ctx.fits(candidateWord)
		if err != nil {
			return 0, err
		}
		if fits {
			low = candidateSize
		} else {
			high = candidateSize - 1
		}
	}
	return low, nil
}

// fits checks the complete free-space mask, without changing it.
func (ctx *PlacementContext) fits(word textutil.Word) (bool, error) {
	for _, angle := range ctx.angles {
		size, err := ctx.fitFootprint(word, angle)
		if err != nil {
			return false, err
		}
		centers, err := ctx.space.ValidCenters(size)
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
	center, angle, found, err := ctx.choosePlacementCenter(word)
	if err != nil || !found {
		return PlacedWord{}, false, err
	}
	if word.FontSize > 24 {
		glyph, err := rasterizeGlyph(word, ctx.fontPath, angle, ctx.wordPadding)
		if err != nil {
			return PlacedWord{}, false, err
		}
		defer glyph.Close()
		if err := ctx.reserveGlyph(glyph, ctx.footprint(word, angle), center); err != nil {
			return PlacedWord{}, false, err
		}
	} else {
		if err := ctx.reserveFootprint(ctx.footprint(word, angle), center); err != nil {
			return PlacedWord{}, false, err
		}
	}
	return PlacedWord{Word: word, X: float64(center.X), Y: float64(center.Y), Angle: angle}, true, nil
}

func (ctx *PlacementContext) choosePlacementCenter(word textutil.Word) (image.Point, int, bool, error) {
	options := make([]orientedCenters, 0, len(ctx.angles))
	defer func() {
		for _, option := range options {
			option.centers.Close()
		}
	}()
	for _, angle := range ctx.angles {
		size, err := ctx.fitFootprint(word, angle)
		if err != nil {
			return image.Point{}, 0, false, err
		}
		centers, err := ctx.space.ValidCenters(size)
		if err != nil {
			return image.Point{}, 0, false, err
		}
		options = append(options, orientedCenters{angle: angle, centers: centers})
	}
	return ctx.regions.choose(options)
}

func (ctx *PlacementContext) fitFootprint(word textutil.Word, angle int) (image.Point, error) {
	if word.FontSize <= 24 {
		return ctx.footprint(word, angle), nil
	}
	glyph, err := rasterizeGlyph(word, ctx.fontPath, angle, ctx.wordPadding)
	if err != nil {
		return image.Point{}, err
	}
	defer glyph.Close()
	return glyph.fitSize(), nil
}

func (ctx *PlacementContext) reserveGlyph(glyph glyphFootprint, rectangle, center image.Point) error {
	origin := center.Add(glyph.offset)
	reserved, err := ctx.space.ReserveMask(glyph.mask, origin)
	if err != nil {
		return err
	}
	if !reserved {
		return errors.New("selected glyph could not be reserved")
	}
	// Keep the original rectangular region score. Only collision space uses the
	// glyph shape; region ranking is independent of the font's stroke weight.
	ctx.regions.reserve(layout.RectAt(center, rectangle))
	rect := image.Rectangle{Min: origin, Max: origin.Add(image.Pt(glyph.mask.Cols(), glyph.mask.Rows()))}
	occupied := ctx.occupancy.Region(rect)
	defer occupied.Close()
	updated := gocv.NewMat()
	defer updated.Close()
	if err := gocv.BitwiseOr(occupied, glyph.mask, &updated); err != nil {
		return err
	}
	return updated.CopyTo(&occupied)
}

// reserveFootprint keeps free space, region usage, and diagnostics in sync.
func (ctx *PlacementContext) reserveFootprint(size, center image.Point) error {
	if !ctx.space.Reserve(size, center) {
		return errors.New("selected center could not be reserved")
	}
	rect := layout.RectAt(center, size)
	ctx.regions.reserve(rect)
	occupied := ctx.occupancy.Region(rect)
	occupied.SetTo(gocv.NewScalar(255, 0, 0, 0))
	occupied.Close()
	return nil
}

func (ctx *PlacementContext) footprint(word textutil.Word, angle int) image.Point {
	width := int(math.Ceil(word.Width + 2*float64(ctx.wordPadding)))
	height := int(math.Ceil(word.Height + 2*float64(ctx.wordPadding)))
	if angle == 90 {
		width, height = height, width
	}
	return image.Pt(max(1, width), max(1, height))
}

func NewPlacementContext(mask *imageutil.Mask, cfg Config) (*PlacementContext, error) {
	if err := validatePlacementInputs(mask, cfg); err != nil {
		return nil, err
	}

	safeZone, occupancy, err := newValidationMasks(mask, cfg.SafeZoneErodeSize)
	if err != nil {
		return nil, fmt.Errorf("prepare placement masks: %w", err)
	}
	ctx := &PlacementContext{
		safeZone:    safeZone,
		occupancy:   occupancy,
		fontPath:    cfg.FontPath,
		wordPadding: cfg.WordPadding,
		angles:      append([]int(nil), cfg.Angles...),
	}
	ctx.space, err = layout.NewFreeSpace(*safeZone)
	if err != nil {
		ctx.Close()
		return nil, fmt.Errorf("create free space: %w", err)
	}
	ctx.regions, err = newRegionPolicy(*safeZone, *mask.DistMat)
	if err != nil {
		ctx.Close()
		return nil, fmt.Errorf("create shape regions: %w", err)
	}
	return ctx, nil
}

func validatePlacementInputs(mask *imageutil.Mask, cfg Config) error {
	if len(cfg.Angles) == 0 {
		return errors.New("at least one placement angle is required")
	}
	for _, angle := range cfg.Angles {
		if angle != 0 && angle != 90 {
			return fmt.Errorf("unsupported placement angle %d: only 0 and 90 are supported", angle)
		}
	}
	if mask == nil || mask.DistMat == nil || mask.DistMat.Empty() {
		return errors.New("distance map is unavailable")
	}
	return nil
}

// newValidationMasks preserves the existing silhouette safety margin and
// creates an occupancy bitmap for optional diagnostics.
func newValidationMasks(mask *imageutil.Mask, erodeSize int) (*gocv.Mat, *gocv.Mat, error) {
	safeZone, err := newSafeZone(mask, erodeSize)
	if err != nil {
		return nil, nil, err
	}
	occupancy := gocv.NewMatWithSize(safeZone.Rows(), safeZone.Cols(), gocv.MatTypeCV8UC1)
	return safeZone, &occupancy, nil
}

func newSafeZone(mask *imageutil.Mask, erodeSize int) (*gocv.Mat, error) {
	if mask == nil || mask.BinaryMat == nil || mask.BinaryMat.Empty() {
		return nil, errors.New("binary placement mask is unavailable")
	}
	if erodeSize <= 0 || erodeSize%2 == 0 {
		return nil, errors.New("safe-zone erosion size must be positive and odd")
	}
	safeZone := gocv.NewMat()
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(erodeSize, erodeSize))
	defer kernel.Close()
	if err := gocv.Erode(*mask.BinaryMat, &safeZone, kernel); err != nil {
		safeZone.Close()
		return nil, fmt.Errorf("create safe zone: %w", err)
	}
	return &safeZone, nil
}
