package wordcloud

import (
	"errors"
	"fmt"
	"math"

	"github.com/santiagoa58/image-play/internal/imageutil"
	"github.com/santiagoa58/image-play/internal/textutil"
)

const placementStepRatio = 0.1

// resolveMaxFontSize returns the maximum font size used for frequency scaling.
//
// A positive Config.MaxFontSize is an explicit override. Otherwise the layout
// calibrates itself by trying to place the most important candidate words in a
// fresh placement context. The probe starts at the image height, mirroring the
// general strategy used by amueller/word_cloud, then uses the harmonic mean of
// the first two successful placed sizes so one unusually easy word does not
// dominate the final scale.
func resolveMaxFontSize(
	mask *imageutil.Mask,
	wordHeap textutil.WordHeap,
	cfg Config,
) (float64, error) {
	if cfg.MaxFontSize > 0 {
		return cfg.MaxFontSize, nil
	}
	if mask == nil || mask.Height <= 0 {
		return 0, errors.New("cannot determine maximum font size without a valid mask")
	}

	candidates := wordHeap.ToSortedSlice()
	if len(candidates) == 0 {
		return cfg.MinFontSize, nil
	}
	if len(candidates) > cfg.WordLimit {
		candidates = candidates[:cfg.WordLimit]
	}

	probe, err := NewPlacementContext(mask, cfg)
	if err != nil {
		return 0, fmt.Errorf("create font-size probe layout: %w", err)
	}
	defer probe.Close()

	trialMax := math.Max(cfg.MinFontSize, float64(mask.Height))
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

		maxForWord := trialMax
		if len(placedSizes) > 0 {
			maxForWord = placedSizes[len(placedSizes)-1]
		}

		placed, err := probe.Place(
			measured,
			maxForWord,
			cfg.MinFontSize,
			placementStepRatio,
		)
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
		return cfg.MinFontSize, nil
	case 1:
		return math.Max(cfg.MinFontSize, math.Round(placedSizes[0])), nil
	default:
		a, b := placedSizes[0], placedSizes[1]
		resolved := 2 * a * b / (a + b)
		return math.Max(cfg.MinFontSize, math.Round(resolved)), nil
	}
}
