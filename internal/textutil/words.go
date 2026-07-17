package textutil

import (
	"fmt"
	"strings"

	"github.com/fogleman/gg"
	"github.com/santiagoa58/image-play/internal/mathutil"
)

type Word struct {
	Text     string
	Weight   int
	FontSize float64
	Width    float64
	Height   float64
}

// WordMeasurementConfig controls frequency-based sizing and font measurement.
type WordMeasurementConfig struct {
	FontPath    string
	MinFontSize float64
	MaxFontSize float64
	Limit       int
}

// MeasureWords returns the most frequent words with logarithmically scaled
// font sizes and measured rendering bounds.
func MeasureWords(h WordHeap, cfg WordMeasurementConfig) ([]Word, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	if h.Len() == 0 {
		return nil, nil
	}

	limit := cfg.Limit
	if limit > h.Len() {
		limit = h.Len()
	}

	// A tiny context is sufficient because gg only needs its font face for
	// measurement.
	dc := gg.NewContext(1, 1)
	sortedWords := h.ToSortedSlice()[:limit]

	countRange := mathutil.Range{
		Min: float64(sortedWords[len(sortedWords)-1].Count),
		Max: float64(sortedWords[0].Count),
	}
	fontSizeRange := mathutil.Range{
		Min: cfg.MinFontSize,
		Max: cfg.MaxFontSize,
	}

	words := make([]Word, len(sortedWords))
	for i, w := range sortedWords {
		size := mathutil.ScaleLog(
			float64(w.Count),
			countRange,
			fontSizeRange,
		)
		width, height, err := measureWord(dc, w.Word, cfg.FontPath, size)
		if err != nil {
			return nil, fmt.Errorf("measure word: %w", err)
		}
		words[i] = Word{
			Text:     w.Word,
			Weight:   w.Count,
			FontSize: size,
			Width:    width,
			Height:   height,
		}
	}

	return words, nil
}

func (cfg WordMeasurementConfig) validate() error {
	switch {
	case strings.TrimSpace(cfg.FontPath) == "":
		return fmt.Errorf("font path is required")
	case cfg.MinFontSize <= 0:
		return fmt.Errorf("minimum font size must be positive")
	case cfg.MaxFontSize < cfg.MinFontSize:
		return fmt.Errorf("maximum font size must be at least the minimum")
	case cfg.Limit <= 0:
		return fmt.Errorf("word limit must be positive")
	default:
		return nil
	}
}

// measureWord returns the rendered bounds of text at fontSize.
func measureWord(ctx *gg.Context, txt, fontPath string, fontSize float64) (w, h float64, err error) {
	if err := ctx.LoadFontFace(fontPath, fontSize); err != nil {
		return 0, 0, fmt.Errorf("failed to load font %q at size %.1f: %w", fontPath, fontSize, err)
	}
	w, h = ctx.MeasureString(txt)
	return w, h, nil
}
