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
		if got := mask.BinaryMat.GetUCharAt(point[1], point[0]); got != 0 {
			t.Errorf(
				"transparent corner (%d,%d) = %d, want 0",
				point[0],
				point[1],
				got,
			)
		}
	}

	if got := gocv.CountNonZero(*mask.BinaryMat); got == 0 {
		t.Fatal("PrepareMask() produced an empty logo mask")
	}
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

func TestBuildBinaryMaskRemovesLightBorderBackground(t *testing.T) {
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
	if got := mask.GetUCharAt(7, 23); got != 255 {
		t.Errorf("visible light region = %d, want 255", got)
	}
	if got := mask.GetUCharAt(7, 15); got != 0 {
		t.Errorf("transparent region = %d, want 0", got)
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

func TestBuildBinaryMaskSelectsBrightSubjectOnDarkBackground(t *testing.T) {
	const width, height = 40, 30
	pixels := make([]byte, width*height*3)
	// A bright subject with a large dark interior hole, surrounded by black.
	for y := 5; y < 25; y++ {
		for x := 5; x < 35; x++ {
			if x >= 15 && x < 25 && y >= 10 && y < 20 {
				continue
			}
			i := (y*width + x) * 3
			pixels[i], pixels[i+1], pixels[i+2] = 20, 100, 240
		}
	}
	img := newTestMat(t, height, width, gocv.MatTypeCV8UC3, pixels)
	defer img.Close()
	binary, dark, err := prepareBinaryMask(img, 8)
	if err != nil {
		t.Fatal(err)
	}
	defer binary.Close()
	if !dark {
		t.Fatal("black border should select a dark background")
	}
	for _, p := range []struct {
		x, y int
		want uint8
	}{{8, 8, 255}, {0, 0, 0}, {20, 15, 0}} {
		if got := binary.GetUCharAt(p.y, p.x); got != p.want {
			t.Errorf("mask(%d,%d) = %d, want %d", p.x, p.y, got, p.want)
		}
	}
}

func TestPrepareMaskVaderPreservesColorsAndExcludesBackground(t *testing.T) {
	mask, err := PrepareMask("../../testdata/images/darth_vader_og.jpg", 8)
	if err != nil {
		t.Fatal(err)
	}
	defer mask.Close()
	if !mask.DarkBackground {
		t.Fatal("Vader should have a dark background")
	}
	if mask.Source.Bounds().Dx() != mask.Width || mask.Source.Bounds().Dy() != mask.Height {
		t.Fatal("source and mask dimensions differ")
	}
	if got := mask.BinaryMat.GetUCharAt(0, 0); got != 0 {
		t.Fatalf("background corner = %d, want 0", got)
	}
	if selected := gocv.CountNonZero(*mask.BinaryMat); selected == 0 || selected > mask.Width*mask.Height/2 {
		t.Fatalf("selected pixels = %d; expected helmet highlights, not background", selected)
	}
	redPixels := 0
	for y := 0; y < mask.Height; y++ {
		for x := 0; x < mask.Width; x++ {
			if mask.BinaryMat.GetUCharAt(y, x) == 0 {
				continue
			}
			r, g, b, _ := mask.Source.At(x, y).RGBA()
			if r > g && r > b {
				redPixels++
			}
		}
	}
	if redPixels == 0 {
		t.Fatal("source colors were lost")
	}
}

func TestDarkBackgroundPreservesSaturatedRedBesideBrightYellow(t *testing.T) {
	const width, height = 40, 30
	pixels := make([]byte, width*height*3)
	for y := 5; y < 25; y++ {
		for x := 5; x < 35; x++ {
			i := (y*width + x) * 3
			pixels[i+2] = 180
			if x >= 20 {
				pixels[i+1], pixels[i+2] = 255, 255
			}
		}
	}
	img := newTestMat(t, height, width, gocv.MatTypeCV8UC3, pixels)
	defer img.Close()
	binary, err := buildBinaryMask(img, 8)
	if err != nil {
		t.Fatal(err)
	}
	defer binary.Close()
	for _, x := range []int{10, 30} {
		if got := binary.GetUCharAt(15, x); got != 255 {
			t.Errorf("colored subject at x=%d excluded", x)
		}
	}
}

func TestDarkBackgroundCleanupKeepsTransparentHoleExcluded(t *testing.T) {
	const width, height = 30, 20
	pixels := make([]byte, width*height*4)
	for i := 3; i < len(pixels); i += 4 {
		pixels[i] = 255
	}
	for y := 4; y < 16; y++ {
		for x := 5; x < 25; x++ {
			i := (y*width + x) * 4
			pixels[i+2] = 220
		}
	}
	hole := (10*width + 15) * 4
	pixels[hole], pixels[hole+1], pixels[hole+2], pixels[hole+3] = 255, 255, 255, 0
	img := newTestMat(t, height, width, gocv.MatTypeCV8UC4, pixels)
	defer img.Close()
	binary, dark, err := prepareBinaryMask(img, 8)
	if err != nil {
		t.Fatal(err)
	}
	defer binary.Close()
	if !dark {
		t.Fatal("expected dark background")
	}
	for _, p := range []struct {
		x, y int
		want uint8
	}{{0, 0, 0}, {14, 10, 255}, {15, 10, 0}} {
		if got := binary.GetUCharAt(p.y, p.x); got != p.want {
			t.Errorf("mask(%d,%d) = %d, want %d", p.x, p.y, got, p.want)
		}
	}
}
