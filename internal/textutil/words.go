package textutil

import (
	"fmt"
	"sort"
	"strings"

	"github.com/fogleman/gg"
	"github.com/santiagoa58/image-play/internal/mathutil"
)

// Word is a measured word-cloud candidate.
//
// FontSize is the desired or currently retried size, while Width and Height are
// the measured rendering bounds at that exact size.
type Word struct {
	Text     string
	Weight   int
	FontSize float64
	Width    float64
	Height   float64
	fontpath string
}

// WordMeasurementConfig controls frequency-based sizing and font measurement.
//
// Limit is a candidate limit, not a required number of final placements.
type WordMeasurementConfig struct {
	FontPath    string
	MinFontSize float64
	MaxFontSize float64
	Limit       int
}

// MeasureWord measures one word at an explicit font size.
func MeasureWord(text string, weight int, fontPath string, fontSize float64) (Word, error) {
	if strings.TrimSpace(fontPath) == "" {
		return Word{}, fmt.Errorf("font path is required")
	}
	if fontSize <= 0 {
		return Word{}, fmt.Errorf("font size must be positive")
	}

	dc := gg.NewContext(1, 1)
	width, height, err := measureWord(dc, text, fontPath, fontSize)
	if err != nil {
		return Word{}, err
	}

	return Word{
		Text:     text,
		Weight:   weight,
		FontSize: fontSize,
		Width:    width,
		Height:   height,
		fontpath: fontPath,
	}, nil
}

// Resize returns w remeasured at font size f.
//
// The original word is unchanged. Resize reuses the private font path captured
// by MeasureWords so placement code does not need to know font details.
func Resize(w Word, f float64) (Word, error) {
	if w.FontSize == f {
		return w, nil
	}
	dc := gg.NewContext(1, 1)
	width, height, err := measureWord(dc, w.Text, w.fontpath, f)
	if err != nil {
		return w, fmt.Errorf("resize word: %w", err)
	}
	return Word{
		Text:     w.Text,
		Weight:   w.Weight,
		FontSize: f,
		Width:    width,
		Height:   height,
		fontpath: w.fontpath,
	}, nil
}

// MeasureWords returns up to cfg.Limit candidates ordered by frequency, with
// logarithmically scaled font sizes and measured rendering bounds.
//
// Log scaling preserves frequency hierarchy without letting a few very common
// words consume the entire visual size range.
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
			fontpath: cfg.FontPath,
		}
	}

	// Frequency remains the primary hierarchy. When frequencies tie, place the
	// physically larger/harder-to-fit rectangle first and leave smaller words to
	// fill fragmented space later.
	sort.Slice(words, func(i, j int) bool {
		if words[i].Weight != words[j].Weight {
			return words[i].Weight > words[j].Weight
		}
		areaI := words[i].Width * words[i].Height
		areaJ := words[j].Width * words[j].Height
		if areaI != areaJ {
			return areaI > areaJ
		}
		return words[i].Text < words[j].Text
	})

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
