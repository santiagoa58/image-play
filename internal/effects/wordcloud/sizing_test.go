package wordcloud

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/santiagoa58/image-play/internal/imageutil"
	"github.com/santiagoa58/image-play/internal/textutil"
	"gocv.io/x/gocv"
)

func TestResolveMaxFontSizeUsesConfiguredOverride(t *testing.T) {
	cfg := NewConfig(WithMaxFontSize(37))

	got, err := resolveMaxFontSize(nil, nil, cfg)
	if err != nil {
		t.Fatalf("resolveMaxFontSize() error = %v, want nil", err)
	}
	if got != 37 {
		t.Fatalf("resolveMaxFontSize() = %v, want 37", got)
	}
}

func TestResolveMaxFontSizeUsesLayout(t *testing.T) {
	mask := newSizingTestMask(t, 240, 160)
	cfg := NewConfig(
		WithFontPath(wordcloudTestFontPath(t)),
		WithWordLimit(3),
	)
	heap := textutil.WordHeap{
		{Word: "important", Count: 10},
		{Word: "secondary", Count: 8},
		{Word: "fallback", Count: 5},
	}

	got, err := resolveMaxFontSize(mask, heap, cfg)
	if err != nil {
		t.Fatalf("resolveMaxFontSize() error = %v", err)
	}
	if got < cfg.MinFontSize {
		t.Fatalf("resolveMaxFontSize() = %v, want >= minimum %v", got, cfg.MinFontSize)
	}
	if got > float64(mask.Height) {
		t.Fatalf("resolveMaxFontSize() = %v, want <= probe upper bound %d", got, mask.Height)
	}
}

func newSizingTestMask(t *testing.T, width, height int) *imageutil.Mask {
	t.Helper()

	binary := gocv.NewMatWithSize(height, width, gocv.MatTypeCV8UC1)
	margin := 8
	roi := binary.Region(image.Rect(margin, margin, width-margin, height-margin))
	roi.SetTo(gocv.NewScalar(255, 0, 0, 0))
	roi.Close()

	dist, err := imageutil.ComputeDistanceTransform(binary)
	if err != nil {
		binary.Close()
		t.Fatalf("ComputeDistanceTransform() error = %v", err)
	}

	mask := &imageutil.Mask{
		Width:     width,
		Height:    height,
		BinaryMat: &binary,
		DistMat:   dist,
	}
	t.Cleanup(mask.Close)
	return mask
}

func wordcloudTestFontPath(t *testing.T) string {
	t.Helper()

	path := filepath.Join(
		"..",
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
