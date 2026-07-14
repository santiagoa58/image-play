package imageutil

import (
	"errors"
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

type MaskSource int

const (
	MaskSourceLuminance MaskSource = iota
	MaskSourceAlpha
)

// Mask holds a prepared binary shape and its distance transform.
// Binary and Distance are stored as 2D slices for ergonomic access.
type Mask struct {
	// permanent definition of where text may exist; true = inside shape.
	Binary [][]bool
	// Euclidean distance to nearest boundary
	Distance  [][]float32
	Width     int
	Height    int
	// permanent definition of where text may exist
	BinaryMat *gocv.Mat
	DistMat   *gocv.Mat
}

// At returns whether the pixel at (x, y) is inside the shape.
func (m *Mask) At(x, y int) bool {
	if x < 0 || y < 0 || x >= m.Width || y >= m.Height {
		return false
	}
	return m.Binary[y][x]
}

// DistanceAt returns the distance value at (x, y).
// Higher values = farther from the boundary.
func (m *Mask) DistanceAt(x, y int) float32 {
	if x < 0 || y < 0 || x >= m.Width || y >= m.Height {
		return 0
	}
	return m.Distance[y][x]
}

func (m *Mask) Close() {
	if m.BinaryMat != nil {
		m.BinaryMat.Close()
		m.BinaryMat = nil
	}
	if m.DistMat != nil {
		m.DistMat.Close()
		m.DistMat = nil
	}
}

// IMWrite writes the img to disk as a PNG file.
func IMWrite(path string, img *gocv.Mat) error {
	if img == nil {
		return fmt.Errorf("no thresholded image available")
	}
	if ok := gocv.IMWrite(path, *img); !ok {
		return fmt.Errorf("failed to write image to %s", path)
	}
	return nil
}

// PrepareMask loads an image, creates a clean binary mask using Otsu,
// applies morphological cleaning, computes the distance transform,
// and returns a Mask with 2D slices.
func PrepareMask(path string, source MaskSource, alphaThreshold uint8) (*Mask, error) {
	img, err := readImage(path, source == MaskSourceLuminance)
	if err != nil {
		return nil, fmt.Errorf("read image from %q: %w", path, err)
	}
	defer img.Close()
	var thImg *gocv.Mat
	if source == MaskSourceAlpha {
		thImg, err = applyAlphaThreshold(*img, alphaThreshold)
	} else {
		thImg = applyBinaryThreshold(*img)
	}
	if err != nil {
		return nil, err
	}

	if err := cleanMask(thImg); err != nil {
		return nil, fmt.Errorf("clean mask: %w", err)
	}

	dist, err := ComputeDistanceTransform(*thImg)
	if err != nil {
		return nil, fmt.Errorf("compute distance transform: %w", err)
	}

	mask, err := createMask(thImg, dist)
	if err != nil {
		return nil, fmt.Errorf("create mask: %w", err)
	}

	return mask, nil
}

// readImage loads an image as grayscale.
func readImage(path string, grayscale bool) (*gocv.Mat, error) {
	flag := gocv.IMReadUnchanged
	if grayscale {
		flag = gocv.IMReadGrayScale
	}
	img := gocv.IMRead(path, flag)
	if img.Empty() {
		return nil, fmt.Errorf("failed to read image: %s", path)
	}
	return &img, nil
}

func applyAlphaThreshold(img gocv.Mat, threshold uint8) (*gocv.Mat, error) {
	alpha := gocv.NewMat()
	defer alpha.Close()

	if img.Channels() != 4 {
		return nil, errors.New("alpha mask requested but image has no alpha channel")
	}
	if err := gocv.ExtractChannel(img, &alpha, 3); err != nil {
		return nil, err
	}
	binary := gocv.NewMat()
	gocv.Threshold(alpha, &binary, float32(threshold), 255, gocv.ThresholdBinary)

	return &binary, nil
}

// BinaryThreshold applies Otsu's method to create a binary mask.
func applyBinaryThreshold(img gocv.Mat) *gocv.Mat {
	th := gocv.NewMat()
	gocv.Threshold(img, &th, 0, 255, gocv.ThresholdBinaryInv|gocv.ThresholdOtsu)
	return &th
}

// cleanMask removes noise and fills small holes using morphological operations.
func cleanMask(th *gocv.Mat) error {
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Point{3, 3})
	defer kernel.Close()

	if err := gocv.MorphologyEx(*th, th, gocv.MorphOpen, kernel); err != nil {
		return err
	}
	if err := gocv.MorphologyEx(*th, th, gocv.MorphClose, kernel); err != nil {
		return err
	}
	return nil
}

// createMask converts gocv.Mat data into 2D Go slices.
func createMask(binaryMat, distMat *gocv.Mat) (*Mask, error) {
	width := binaryMat.Cols()
	height := binaryMat.Rows()

	binary := make([][]bool, height)
	distance := make([][]float32, height)

	binaryData, err := binaryMat.DataPtrUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to get binary data: %w", err)
	}

	distData, err := distMat.DataPtrFloat32()
	if err != nil {
		return nil, fmt.Errorf("failed to get distance data: %w", err)
	}

	for y := range height {
		binary[y] = make([]bool, width)
		distance[y] = make([]float32, width)

		for x := range width {
			idx := y*width + x
			binary[y][x] = binaryData[idx] > 128
			distance[y][x] = distData[idx]
		}
	}

	return &Mask{
		Binary:    binary,
		Distance:  distance,
		Width:     width,
		Height:    height,
		BinaryMat: binaryMat,
		DistMat:   distMat,
	}, nil
}

// getValidationMask creates two masks used for fast validation during placement.
//
// safeZone: A shrunk version of the binary mask. Any rectangle that fits
//	completely inside this mask is guaranteed to have enough
//	breathing room from the actual shape boundary.
//
// occupancy: Starts empty. We mark areas as occupied when we place words.
//	This lets us quickly check if a candidate position overlaps
//	any already placed word.
func GetValidationMask(mask *Mask) (*gocv.Mat, *gocv.Mat, error) {
	safeZone := gocv.NewMat()

	// Create a kernel size based on padding.
	// A kernel of size (padding*2 + 1) roughly erodes by `padding` pixels.
	kSize := 3
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Point{kSize, kSize})
	defer kernel.Close()

	if err := gocv.Erode(*mask.BinaryMat, &safeZone, kernel); err != nil {
		safeZone.Close()
		return nil, nil, fmt.Errorf("failed to create safe zone: %w", err)
	}

	occupancy := gocv.NewMatWithSize(
		mask.BinaryMat.Rows(),
		mask.BinaryMat.Cols(),
		gocv.MatTypeCV8UC1,
	)

	return &safeZone, &occupancy, nil
}
