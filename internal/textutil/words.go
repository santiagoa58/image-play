package textutil

import (
	"fmt"

	"github.com/fogleman/gg"
)

type Word struct {
	Text   string
	Weight int
	Size   float64
}

func WordsFromCounts(h WordHeap, minSize, maxSize float64) []Word {
	if h.Len() == 0 {
		return nil
	}
	words := make([]Word, h.Len())
	maxCount := float64(max(h.Top().Count, 1)) // Avoid division by zero
	counts := h.ToSortedSlice()
	for i, w := range counts {
		countPercent := float64(w.Count) / maxCount
		size := minSize + (maxSize-minSize)*countPercent
		words[i] = Word{
			Text:   w.Word,
			Weight: w.Count,
			Size:   size,
		}
	}
	return words
}

// MeasureWord returns the width and height of the text in pixels
// when rendered at the given font size.
func MeasureWord(text string, fontPath string, fontSize float64) (width, height float64, err error) {
	dc := gg.NewContext(1, 1) // dummy context is enough for measuring
	
	if err := dc.LoadFontFace(fontPath, fontSize); err != nil {
		return 0, 0, fmt.Errorf("failed to load font %q at size %.1f: %w", fontPath, fontSize, err)
	}

	w, h := dc.MeasureString(text)
	return w, h, nil
}
