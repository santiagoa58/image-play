package textutil

import (
	"fmt"

	"github.com/fogleman/gg"
)

type Word struct {
	Text     string
	Weight   int
	FontSize float64
	Width    float64
	Height   float64
}

func (w *Word) Padding() float64 {
	return min(w.FontSize*0.25, 5)
}

func WordsWithMeasurements(h WordHeap, minSize, maxSize float64, fontpath string) ([]Word, error) {
	if h.Len() == 0 {
		return nil, nil
	}
	words := make([]Word, h.Len())
	dc := gg.NewContext(1, 1)                  // dummy context is enough for measuring
	maxCount := float64(max(h.Top().Count, 1)) // Avoid division by zero
	for i, w := range h.ToSortedSlice() {
		countPercent := float64(w.Count) / maxCount
		size := minSize + (maxSize-minSize)*countPercent
		width, height, err := measureWord(dc, w.Word, fontpath, size)
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

// MeasureWord returns the measured width and height in pixels for a given word when rendered at its corresponding font size.
func measureWord(ctx *gg.Context, txt, fontpath string, fontsize float64) (w, h float64, err error) {
	if err := ctx.LoadFontFace(fontpath, fontsize); err != nil {
		return 0, 0, fmt.Errorf("failed to load font %q at size %.1f: %w", fontpath, fontsize, err)
	}
	w, h = ctx.MeasureString(txt)
	return w, h, nil
}
