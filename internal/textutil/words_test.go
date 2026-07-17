package textutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/santiagoa58/image-play/internal/mathutil"
)

func TestMeasureWordsScalesSelectedFrequencies(t *testing.T) {
	const (
		minFontSize = 10.0
		maxFontSize = 50.0
	)

	heap := WordHeap{
		{Word: "high", Count: 100},
		{Word: "middle", Count: 10},
		{Word: "low", Count: 1},
	}
	fontPath := testWordFontPath(t)

	words, err := MeasureWords(
		heap,
		WordMeasurementConfig{
			FontPath:    fontPath,
			MinFontSize: minFontSize,
			MaxFontSize: maxFontSize,
			Limit:       3,
		},
	)
	if err != nil {
		t.Fatalf("MeasureWords() error = %v", err)
	}
	if len(words) != 3 {
		t.Fatalf("MeasureWords() returned %d words, want 3", len(words))
	}

	countRange := mathutil.Range{Min: 1, Max: 100}
	fontRange := mathutil.Range{Min: minFontSize, Max: maxFontSize}

	for _, word := range words {
		wantSize := mathutil.ScaleLog(
			float64(word.Weight),
			countRange,
			fontRange,
		)
		if word.FontSize != wantSize {
			t.Errorf(
				"word %q font size = %v, want %v",
				word.Text,
				word.FontSize,
				wantSize,
			)
		}
		if word.Width <= 0 || word.Height <= 0 {
			t.Errorf(
				"word %q dimensions = %vx%v, want positive values",
				word.Text,
				word.Width,
				word.Height,
			)
		}
	}
}

func TestMeasureWordsUsesLimitedFrequencyRange(t *testing.T) {
	heap := WordHeap{
		{Word: "high", Count: 100},
		{Word: "middle", Count: 10},
		{Word: "excluded", Count: 1},
	}

	words, err := MeasureWords(
		heap,
		WordMeasurementConfig{
			FontPath:    testWordFontPath(t),
			MinFontSize: 10,
			MaxFontSize: 50,
			Limit:       2,
		},
	)
	if err != nil {
		t.Fatalf("MeasureWords() error = %v", err)
	}
	if len(words) != 2 {
		t.Fatalf("MeasureWords() returned %d words, want 2", len(words))
	}

	if words[0].Text != "high" || words[0].FontSize != 50 {
		t.Errorf(
			"highest-frequency word = %#v, want text high at size 50",
			words[0],
		)
	}
	if words[1].Text != "middle" || words[1].FontSize != 10 {
		t.Errorf(
			"lowest selected word = %#v, want text middle at size 10",
			words[1],
		)
	}
}

func testWordFontPath(t *testing.T) string {
	t.Helper()

	path := filepath.Join(
		"..",
		"..",
		"fonts",
		"NotoSansMono-VariableFont_wdth,wght.ttf",
	)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("test font missing at %q: %v", path, err)
	}

	return path
}
