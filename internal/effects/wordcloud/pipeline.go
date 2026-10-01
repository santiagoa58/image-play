package wordcloud

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/santiagoa58/image-play/internal/imageutil"
	"github.com/santiagoa58/image-play/internal/textutil"
)

// prepareWords turns text frequencies into measured candidates for this image.
func prepareWords(mask *imageutil.Mask, cfg Config, logger *slog.Logger) ([]textutil.Word, float64, error) {
	minFontSize := minimumFontSize(cfg, mask.Width, mask.Height)

	logger.Info("counting words", "path", cfg.TextPath)
	wordCounts, err := textutil.CountWords(cfg.TextPath)
	if err != nil {
		return nil, 0, fmt.Errorf("count words: %w", err)
	}
	wordCounts = displayWordCounts(wordCounts, cfg.Uppercase)

	maxFontSize, err := resolveMaxFontSize(mask, wordCounts, cfg)
	if err != nil {
		return nil, 0, fmt.Errorf("resolve maximum font size: %w", err)
	}
	logger.Info("resolved font range", "min", minFontSize, "max", maxFontSize)

	words, err := textutil.MeasureWords(wordCounts, textutil.WordMeasurementConfig{
		FontPath:    cfg.FontPath,
		MinFontSize: minFontSize,
		MaxFontSize: maxFontSize,
		Limit:       cfg.WordLimit,
	})
	if err != nil {
		return nil, 0, err
	}
	logger.Info("prepared words", "count", len(words))
	return words, minFontSize, nil
}

// placeWords reserves the chosen words and reports the same progress and
// placement summary as the CLI pipeline.
func placeWords(ctx *PlacementContext, words []textutil.Word, minFontSize float64, debug bool, logger *slog.Logger) ([]PlacedWord, error) {
	logger.Info("placing words")
	started := time.Now()
	var placed []PlacedWord
	skipped, horizontal, vertical := 0, 0, 0

	for i, w := range words {
		percent := 100 * (i + 1) / len(words)
		fmt.Printf("\rProgress: [%3d%%] %d/%d", percent, i+1, len(words))
		p, err := ctx.Place(w, minFontSize)
		if err != nil {
			if errors.Is(err, errNoPlacement) {
				skipped++
				if debug {
					logger.Info("word skipped", "word", w.Text, "reason", err)
				}
				continue
			}
			return nil, fmt.Errorf("place word %q: %w", w.Text, err)
		}

		placed = append(placed, p)
		if p.Angle == 90 {
			vertical++
		} else {
			horizontal++
		}
	}
	duration := time.Since(started)
	fmt.Println()

	logger.Info(
		"placement complete",
		"prepared", len(words),
		"placed", len(placed),
		"skipped", skipped,
		"horizontal", horizontal,
		"vertical", vertical,
		"duration", duration,
	)
	return placed, nil
}

func displayWordCounts(counts textutil.WordCounts, uppercase bool) textutil.WordCounts {
	if !uppercase {
		return counts
	}
	result := make(textutil.WordCounts, len(counts))
	copy(result, counts)
	for i := range result {
		result[i].Word = strings.ToUpper(result[i].Word)
	}
	return result
}
