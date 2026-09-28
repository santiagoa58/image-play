package wordcloud

import (
	"errors"
	"fmt"
	"math"

	"github.com/santiagoa58/image-play/internal/imageutil"
	"github.com/santiagoa58/image-play/internal/textutil"
)

// minimumFontSize uses the larger of 8px or 1% of the image's shortest side
// unless the caller sets an override.
func minimumFontSize(cfg Config, width, height int) float64 {
	if cfg.MinFontSize > 0 {
		return cfg.MinFontSize
	}
	shortestSide := min(width, height)
	return math.Max(8, math.Ceil(0.01*float64(shortestSide)))
}

// resolveMaxFontSize returns the maximum font size used for frequency scaling.
//
// The layout calibrates itself by trying to place the most important candidate words in a
// fresh placement context. The probe starts at the image height, mirroring the
// general strategy used by amueller/word_cloud, then uses the harmonic mean of
// the first two successful placed sizes so one unusually easy word does not
// dominate the final scale.
func resolveMaxFontSize(
	mask *imageutil.Mask,
	wordHeap textutil.WordCounts,
	cfg Config,
) (float64, error) {
	if mask == nil || mask.Height <= 0 {
		return 0, errors.New("cannot determine maximum font size without a valid mask")
	}
	minSize := minimumFontSize(cfg, mask.Width, mask.Height)

	candidates := wordHeap.ToSortedSlice()
	if len(candidates) == 0 {
		return minSize, nil
	}
	if len(candidates) > cfg.WordLimit {
		candidates = candidates[:cfg.WordLimit]
	}

	probe, err := NewPlacementContext(mask, cfg)
	if err != nil {
		return 0, fmt.Errorf("create font-size probe layout: %w", err)
	}
	defer probe.Close()

	trialMax := math.Max(minSize, float64(mask.Height))
	placedSizes := make([]float64, 0, 2)

	for _, candidate := range candidates {
		measured, err := textutil.MeasureWord(
			candidate.Word,
			candidate.Count,
			cfg.FontPath,
			trialMax,
		)
		if err != nil {
			return 0, fmt.Errorf("measure font-size probe word %q: %w", candidate.Word, err)
		}

		placed, err := probe.Place(measured, minSize)
		if err != nil {
			if errors.Is(err, errNoPlacement) {
				continue
			}
			return 0, fmt.Errorf("probe font size with %q: %w", candidate.Word, err)
		}

		placedSizes = append(placedSizes, placed.Word.FontSize)
		if len(placedSizes) == 2 {
			break
		}
	}

	switch len(placedSizes) {
	case 0:
		return minSize, nil
	case 1:
		return math.Max(minSize, math.Round(placedSizes[0])), nil
	default:
		a, b := placedSizes[0], placedSizes[1]
		resolved := 2 * a * b / (a + b)
		return math.Max(minSize, math.Round(resolved)), nil
	}
}
