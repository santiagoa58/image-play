package imageutil

import (
	"path/filepath"
	"testing"

	"gocv.io/x/gocv"
)

func TestPrepareMaskExcludesTransparentLogoCorners(t *testing.T) {
	path := filepath.Join(
		"..",
		"..",
		"testdata",
		"images",
		"deepseek-logo-icon.png",
	)

	mask, err := PrepareMask(path, 8)
	if err != nil {
		t.Fatalf("PrepareMask() error = %v", err)
	}
	defer mask.Close()

	for _, point := range [][2]int{
		{0, 0},
		{mask.Width - 1, 0},
		{0, mask.Height - 1},
		{mask.Width - 1, mask.Height - 1},
	} {
		if mask.At(point[0], point[1]) {
			t.Errorf(
				"transparent corner (%d,%d) is placeable",
				point[0],
				point[1],
			)
		}
	}

	if got := gocv.CountNonZero(*mask.BinaryMat); got == 0 {
		t.Fatal("PrepareMask() produced an empty logo mask")
	}
}

func TestBuildAlphaVisibilityMaskUsesStrictThreshold(t *testing.T) {
	// Alpha values equal to the threshold remain invisible. ThresholdBinary
	// makes only values greater than the threshold visible.
	img := newTestMat(t, 1, 3, gocv.MatTypeCV8UC4, []byte{
		0, 0, 0, 7,
		0, 0, 0, 8,
		0, 0, 0, 9,
	})
	defer img.Close()

	visible, err := buildAlphaVisibilityMask(img, 8)
	if err != nil {
		t.Fatalf("buildAlphaVisibilityMask() error = %v", err)
	}
	defer visible.Close()

	assertRowEquals(t, *visible, 0, []uint8{0, 0, 255})
}

func TestBuildAlphaVisibilityMaskRejectsImageWithoutAlpha(t *testing.T) {
	img := gocv.NewMatWithSize(1, 1, gocv.MatTypeCV8UC3)
	defer img.Close()

	visible, err := buildAlphaVisibilityMask(img, 8)
	if visible != nil {
		visible.Close()
		t.Fatal(
			"buildAlphaVisibilityMask() returned a mask for a three-channel image",
		)
	}
	if err == nil {
		t.Fatal("buildAlphaVisibilityMask() error = nil, want an error")
	}
}

func TestVisibleLuminanceSetsInvisiblePixelsToWhite(t *testing.T) {
	gray := newTestMat(t, 1, 3, gocv.MatTypeCV8UC1, []byte{
		10, 20, 30,
	})
	defer gray.Close()

	visible := newTestMat(t, 1, 3, gocv.MatTypeCV8UC1, []byte{
		0, 255, 0,
	})
	defer visible.Close()

	got, err := visibleLuminance(gray, visible)
	if err != nil {
		t.Fatalf("visibleLuminance() error = %v", err)
	}
	defer got.Close()

	assertRowEquals(t, *got, 0, []uint8{255, 20, 255})
}

func TestApplyBinaryThresholdSelectsDarkPixels(t *testing.T) {
	gray := newTestMat(t, 1, 2, gocv.MatTypeCV8UC1, []byte{
		0, 255,
	})
	defer gray.Close()

	binary := applyBinaryThreshold(gray)
	defer binary.Close()

	assertRowEquals(t, *binary, 0, []uint8{255, 0})
}

func TestCleanMaskFillsSinglePixelHole(t *testing.T) {
	const size = 9

	data := make([]byte, size*size)
	for i := range data {
		data[i] = 255
	}
	data[(size/2)*size+size/2] = 0

	binary := newTestMat(
		t,
		size,
		size,
		gocv.MatTypeCV8UC1,
		data,
	)
	defer binary.Close()

	if err := cleanMask(&binary); err != nil {
		t.Fatalf("cleanMask() error = %v", err)
	}

	if got := binary.GetUCharAt(size/2, size/2); got != 255 {
		t.Errorf("center pixel after cleanMask() = %d, want 255", got)
	}
}

func TestBuildBinaryMaskReappliesVisibilityAfterCleanup(t *testing.T) {
	const size = 9

	data := make([]byte, size*size*4)
	for i := 0; i < size*size; i++ {
		offset := i * 4
		data[offset+3] = 255
	}

	center := ((size/2)*size + size/2) * 4
	data[center+3] = 0

	img := newTestMat(
		t,
		size,
		size,
		gocv.MatTypeCV8UC4,
		data,
	)
	defer img.Close()

	mask, err := buildBinaryMask(img, 8)
	if err != nil {
		t.Fatalf("buildBinaryMask() error = %v", err)
	}
	defer mask.Close()

	if got := mask.GetUCharAt(size/2, size/2); got != 0 {
		t.Errorf("transparent center pixel = %d, want 0", got)
	}
	if got := mask.GetUCharAt(size/2, size/2-1); got != 255 {
		t.Errorf("opaque pixel beside center = %d, want 255", got)
	}
}

func TestBuildBinaryMaskWithoutAlphaUsesLuminance(t *testing.T) {
	const (
		rows = 15
		cols = 30
	)

	data := make([]byte, rows*cols*3)
	for i := range data {
		data[i] = 255
	}

	setBGRRect(data, cols, 3, 3, 12, 12, 0)

	img := newTestMat(
		t,
		rows,
		cols,
		gocv.MatTypeCV8UC3,
		data,
	)
	defer img.Close()

	mask, err := buildBinaryMask(img, 8)
	if err != nil {
		t.Fatalf("buildBinaryMask() error = %v", err)
	}
	defer mask.Close()

	if got := mask.GetUCharAt(7, 7); got != 255 {
		t.Errorf("dark region center = %d, want 255", got)
	}
	if got := mask.GetUCharAt(7, 20); got != 0 {
		t.Errorf("light background pixel = %d, want 0", got)
	}
}

func TestBuildBinaryMaskIgnoresHiddenRGB(t *testing.T) {
	const (
		rows = 15
		cols = 30
	)

	// The image starts as transparent black. Two visible 9x9 regions contain
	// dark and light pixels. Hidden black must not become placeable or cause
	// the visible dark region to be classified as background.
	data := make([]byte, rows*cols*4)
	setBGRARect(data, cols, 2, 3, 11, 12, 80, 255)
	setBGRARect(data, cols, 19, 3, 28, 12, 240, 255)

	img := newTestMat(
		t,
		rows,
		cols,
		gocv.MatTypeCV8UC4,
		data,
	)
	defer img.Close()

	mask, err := buildBinaryMask(img, 8)
	if err != nil {
		t.Fatalf("buildBinaryMask() error = %v", err)
	}
	defer mask.Close()

	if got := mask.GetUCharAt(7, 6); got != 255 {
		t.Errorf("visible dark region = %d, want 255", got)
	}
	if got := mask.GetUCharAt(7, 23); got != 0 {
		t.Errorf("visible light region = %d, want 0", got)
	}
	if got := mask.GetUCharAt(7, 15); got != 0 {
		t.Errorf("transparent region = %d, want 0", got)
	}
}

func TestGrayscaleRejectsUnsupportedChannelCount(t *testing.T) {
	img := gocv.NewMatWithSize(1, 1, gocv.MatTypeCV8UC2)
	defer img.Close()

	gray, err := grayscale(img)
	if gray != nil {
		gray.Close()
		t.Fatal("grayscale() returned a matrix for an unsupported channel count")
	}
	if err == nil {
		t.Fatal("grayscale() error = nil, want an error")
	}
}

func newTestMat(
	t *testing.T,
	rows,
	cols int,
	matType gocv.MatType,
	data []byte,
) gocv.Mat {
	t.Helper()

	mat, err := gocv.NewMatFromBytes(rows, cols, matType, data)
	if err != nil {
		t.Fatalf("create %dx%d test matrix: %v", cols, rows, err)
	}

	return mat
}

func assertRowEquals(
	t *testing.T,
	mat gocv.Mat,
	row int,
	want []uint8,
) {
	t.Helper()

	for col, wantValue := range want {
		if got := mat.GetUCharAt(row, col); got != wantValue {
			t.Errorf(
				"pixel (%d,%d) = %d, want %d",
				col,
				row,
				got,
				wantValue,
			)
		}
	}
}

func setBGRRect(
	data []byte,
	cols,
	minX,
	minY,
	maxX,
	maxY int,
	value byte,
) {
	for y := minY; y < maxY; y++ {
		for x := minX; x < maxX; x++ {
			offset := (y*cols + x) * 3
			data[offset] = value
			data[offset+1] = value
			data[offset+2] = value
		}
	}
}

func setBGRARect(
	data []byte,
	cols,
	minX,
	minY,
	maxX,
	maxY int,
	value,
	alpha byte,
) {
	for y := minY; y < maxY; y++ {
		for x := minX; x < maxX; x++ {
			offset := (y*cols + x) * 4
			data[offset] = value
			data[offset+1] = value
			data[offset+2] = value
			data[offset+3] = alpha
		}
	}
}
