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
	if got < minimumFontSize(cfg, mask.Width, mask.Height) {
		t.Fatalf("resolveMaxFontSize() = %v, want >= minimum %v", got, minimumFontSize(cfg, mask.Width, mask.Height))
	}
	if got > float64(mask.Height) {
		t.Fatalf("resolveMaxFontSize() = %v, want <= probe upper bound %d", got, mask.Height)
	}
}

func TestMinimumFontSizeUsesImageScaleOrExplicitOverride(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
		want   float64
	}{
		{name: "eight pixel floor", width: 600, height: 400, want: 8},
		{name: "one percent boundary", width: 1200, height: 800, want: 8},
		{name: "rounds scaled minimum up", width: 1200, height: 801, want: 9},
		{name: "uses shortest side", width: 1600, height: 900, want: 9},
		{name: "scales larger images", width: 2048, height: 1024, want: 11},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := minimumFontSize(NewConfig(), tt.width, tt.height); got != tt.want {
				t.Errorf("minimumFontSize() = %v, want %v", got, tt.want)
			}
		})
	}

	if got := minimumFontSize(NewConfig(WithMinFontSize(9)), 2048, 2048); got != 9 {
		t.Errorf("explicit minimum = %v, want 9", got)
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
