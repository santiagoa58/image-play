package imageutil

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

// Mask represents a prepared binary shape mask along with its
// Euclidean distance transform. It uses 1D slices for memory efficiency.
type Mask struct {
	// Binary indicates whether each pixel is inside the shape.
	// true = inside (allowed), false = outside.
	Binary []bool
	// Distance stores the Euclidean distance from each pixel to the nearest edge.
	// Higher values are farther from the boundary (better locations for large words).
	Distance         []float32
	Width            int
	Height           int
	ThresholdedImage *gocv.Mat
}

// At reports whether the pixel at (x, y) is inside the allowed shape.
func (m *Mask) At(x, y int) bool {
	if x < 0 || y < 0 || x >= m.Width || y >= m.Height {
		return false
	}
	// use standard row-major ordering for 1D indexing
	point := y*m.Width + x
	return m.Binary[point]
}

// DistanceAt returns the Euclidean distance value at (x, y).
// Higher values mean the pixel is farther from the edge of the shape.
func (m *Mask) DistanceAt(x, y int) float32 {
	if x < 0 || y < 0 || x >= m.Width || y >= m.Height {
		return 0
	}
	point := y*m.Width + x
	return m.Distance[point]
}

func (m *Mask) IMWrite(out string) error {
	ok := gocv.IMWrite(out, *m.ThresholdedImage)
	if !ok {
		return fmt.Errorf("failed to write thresholded image to %q", out)
	}
	return nil
}

// Close closes the underlying gocv.Mat objects
func (m *Mask) Close() error {
	defer m.ThresholdedImage.Close()
	return nil
}

// PrepareMask loads an image, converts it to a clean binary mask,
// applies morphological cleaning, computes the distance transform,
// and returns a Mask.
func PrepareMask(path string) (*Mask, error) {
	// Load the image from file
	img, err := ReadImage(path)
	if err != nil {
		return nil, err
	}
	defer img.Close()

	// Load and threshold the image to create a binary mask
	// will be closed as part of the Close func
	thImg := BinaryThreshold(*img)

	// Apply morphological cleaning
	if err := CleanMask(thImg); err != nil {
		return nil, err
	}

	// Compute distance transform
	dist, err := ComputeDistanceTransform(*thImg)
	if err != nil {
		return nil, err
	}
	defer dist.Close()

	return createMask(thImg, dist)
}

// ReadImage loads the image from disk as grayscale.
func ReadImage(path string) (*gocv.Mat, error) {
	img := gocv.IMRead(path, gocv.IMReadGrayScale)
	if img.Empty() {
		return nil, fmt.Errorf("failed to read image: %s", path)
	}
	return &img, nil
}

// BinaryThreshold converts a grayscale image to a binary mask using Otsu's method.
func BinaryThreshold(img gocv.Mat) *gocv.Mat {
	th := gocv.NewMat()
	gocv.Threshold(img, &th, 0, 255, gocv.ThresholdBinary|gocv.ThresholdOtsu)
	return &th
}

// CleanMask applies morphological opening followed by closing on the binary mask
// to remove small noise specks and fill small holes.
func CleanMask(th *gocv.Mat) error {
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Point{3, 3})
	defer kernel.Close()
	// remove small white specks
	if err := gocv.MorphologyEx(*th, th, gocv.MorphOpen, kernel); err != nil {
		return err
	}
	// fill small black holes inside shape
	if err := gocv.MorphologyEx(*th, th, gocv.MorphClose, kernel); err != nil {
		return err
	}
	return nil
}

// ComputeDistanceTransform computes the Euclidean distance transform of the binary mask.
func ComputeDistanceTransform(th gocv.Mat) (*gocv.Mat, error) {
	dist := gocv.NewMat()
	labels := gocv.NewMat()
	defer labels.Close()
	if err := gocv.DistanceTransform(th, &dist, &labels, gocv.DistL2, gocv.DistanceMaskPrecise, gocv.DistanceLabelPixel); err != nil {
		return nil, err
	}
	return &dist, nil
}

// createMask
func createMask(binaryMat, distMat *gocv.Mat) (*Mask, error) {
	width := binaryMat.Cols()
	height := binaryMat.Rows()
	total := width * height

	binary := make([]bool, total)
	distance := make([]float32, total)

	imgData, err := binaryMat.DataPtrUint8()
	if err != nil {
		return nil, fmt.Errorf("failed to get binary data pointer: %w", err)
	}

	distData, err := distMat.DataPtrFloat32()
	if err != nil {
		return nil, fmt.Errorf("failed to get distance data pointer: %w", err)
	}
	// convert the 2D OpenCV Mats into 1D slices for efficiency
	for i := range total {
		binary[i] = imgData[i] > 128
		distance[i] = distData[i]
	}

	return &Mask{
		Binary:           binary,
		Distance:         distance,
		Width:            width,
		Height:           height,
		ThresholdedImage: binaryMat,
	}, nil
}
