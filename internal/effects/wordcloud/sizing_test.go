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
	heap := textutil.WordCounts{
		{Word: "important", Count: 10},
		{Word: "secondary", Count: 8},
		{Word: "fallback", Count: 5},
	}

	got, err := resolveMaxFontSize(mask, heap, cfg)
	if err != nil {
		t.Fatalf("resolveMaxFontSize() error = %v", err)
	}
	if got < minimumFontSize(mask, cfg) {
		t.Fatalf("resolveMaxFontSize() = %v, want >= minimum %v", got, minimumFontSize(mask, cfg))
	}
	if got > float64(mask.Height) {
		t.Fatalf("resolveMaxFontSize() = %v, want <= probe upper bound %d", got, mask.Height)
	}
}

func TestMinimumFontSizeUsesImageScaleOrExplicitOverride(t *testing.T) {
	for _, tc := range []struct {
		width, height int
		want          float64
	}{
		{300, 300, 6},
		{1000, 800, 8},
		{4000, 4000, 40},
	} {
		mask := &imageutil.Mask{Width: tc.width, Height: tc.height}
		if got := minimumFontSize(mask, NewConfig()); got != tc.want {
			t.Errorf("image %dx%d minimum = %v, want %v", tc.width, tc.height, got, tc.want)
		}
		if got := minimumFontSize(mask, NewConfig(WithMinFontSize(9))); got != 9 {
			t.Errorf("explicit minimum = %v, want 9", got)
		}
	}
}

func TestMeasuredFootprintsGrowWithFontSize(t *testing.T) {
	// Binary search relies on this property for the bundled font. A regression
	// in font measurement could otherwise miss the largest fitting size.
	font := wordcloudTestFontPath(t)
	ctx := &PlacementContext{wordPadding: 1}
	for _, text := range []string{"I", "example", "WAITING", "gyp"} {
		previous := image.Point{}
		for size := 6; size <= 200; size++ {
			word, err := textutil.MeasureWord(text, 1, font, float64(size))
			if err != nil {
				t.Fatal(err)
			}
			current := ctx.footprint(word, 0)
			if current.X < previous.X || current.Y < previous.Y {
				t.Fatalf("%q footprint shrank at %dpx: %v after %v", text, size, current, previous)
			}
			previous = current
		}
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
