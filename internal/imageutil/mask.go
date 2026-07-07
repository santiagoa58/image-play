package imageutil

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

// Mask holds a prepared binary shape and its distance transform.
// Binary and Distance are stored as 2D slices for ergonomic access.
type Mask struct {
	Binary           [][]bool    // true = inside the shape
	Distance         [][]float32 // Euclidean distance to nearest boundary
	Width            int
	Height           int
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

// IMWrite writes the img to disk as a PNG file.
func  IMWrite(path string, img *gocv.Mat) error {
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
func PrepareMask(path string) (*Mask, error) {
	img, err := ReadImage(path)
	if err != nil {
		return nil, err
	}
	defer img.Close()

	thImg := BinaryThreshold(*img)
	defer thImg.Close()

	if err := cleanMask(thImg); err != nil {
		return nil, err
	}

	dist, err := computeDistanceTransform(*thImg)
	if err != nil {
		return nil, err
	}
	defer dist.Close()

	mask, err := createMask(thImg, dist)
	if err != nil {
		return nil, err
	}

	return mask, nil
}

// ReadImage loads an image as grayscale.
func ReadImage(path string) (*gocv.Mat, error) {
	img := gocv.IMRead(path, gocv.IMReadGrayScale)
	if img.Empty() {
		return nil, fmt.Errorf("failed to read image: %s", path)
	}
	return &img, nil
}

// BinaryThreshold applies Otsu's method to create a binary mask.
func BinaryThreshold(img gocv.Mat) *gocv.Mat {
	th := gocv.NewMat()
	gocv.Threshold(img, &th, 0, 255, gocv.ThresholdBinary|gocv.ThresholdOtsu)
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

// computeDistanceTransform runs the Euclidean distance transform.
func computeDistanceTransform(th gocv.Mat) (*gocv.Mat, error) {
	dist := gocv.NewMat()
	labels := gocv.NewMat()
	defer labels.Close()
	if err := gocv.DistanceTransform(th, &dist, &labels, gocv.DistL2, gocv.DistanceMaskPrecise, gocv.DistanceLabelPixel); err != nil {
		dist.Close()
		return nil, err
	}
	return &dist, nil
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
		Binary:   binary,
		Distance: distance,
		Width:    width,
		Height:   height,
	}, nil
}
