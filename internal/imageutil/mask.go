package imageutil

import (
	"errors"
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

// Mask contains a binary placement shape and its distance transform.
type Mask struct {
	// Binary indicates where text may be placed.
	Binary [][]bool

	// Distance stores each pixel's distance from the nearest shape boundary.
	Distance [][]float32

	Width  int
	Height int

	// BinaryMat and DistMat are retained for OpenCV operations.
	BinaryMat *gocv.Mat
	DistMat   *gocv.Mat
}

// At reports whether (x, y) is inside the placement shape.
func (m *Mask) At(x, y int) bool {
	if x < 0 || y < 0 || x >= m.Width || y >= m.Height {
		return false
	}
	return m.Binary[y][x]
}

// DistanceAt returns the distance from (x, y) to the nearest shape boundary.
func (m *Mask) DistanceAt(x, y int) float32 {
	if x < 0 || y < 0 || x >= m.Width || y >= m.Height {
		return 0
	}
	return m.Distance[y][x]
}

// Close releases the OpenCV matrices owned by the mask.
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

// IMWrite writes img to disk.
func IMWrite(path string, img *gocv.Mat) error {
	if img == nil {
		return errors.New("no image available")
	}
	if ok := gocv.IMWrite(path, *img); !ok {
		return fmt.Errorf("failed to write image to %q", path)
	}
	return nil
}

// PrepareMask builds the placement mask and distance transform for an image.
// Dark visible pixels become placeable; transparent pixels are excluded.
func PrepareMask(path string, alphaThreshold uint8) (*Mask, error) {
	img, err := readImage(path)
	if err != nil {
		return nil, fmt.Errorf("read image from %q: %w", path, err)
	}
	defer img.Close()

	binary, err := buildBinaryMask(*img, alphaThreshold)
	if err != nil {
		return nil, fmt.Errorf("build binary mask: %w", err)
	}

	dist, err := ComputeDistanceTransform(*binary)
	if err != nil {
		binary.Close()
		return nil, fmt.Errorf("compute distance transform: %w", err)
	}

	mask, err := createMask(binary, dist)
	if err != nil {
		binary.Close()
		dist.Close()
		return nil, fmt.Errorf("create mask: %w", err)
	}

	return mask, nil
}

// readImage loads an image without discarding its alpha channel.
func readImage(path string) (*gocv.Mat, error) {
	img := gocv.IMRead(path, gocv.IMReadUnchanged)
	if img.Empty() {
		return nil, fmt.Errorf("failed to read image from %q", path)
	}
	return &img, nil
}

// buildBinaryMask returns a cleaned binary placement mask.
// When alpha is present, invisible pixels are excluded.
// The caller owns the returned matrix.
func buildBinaryMask(
	img gocv.Mat,
	alphaThreshold uint8,
) (*gocv.Mat, error) {
	gray, err := grayscale(img)
	if err != nil {
		return nil, err
	}
	defer gray.Close()

	thresholdInput := gray

	var visibleMask *gocv.Mat
	if img.Channels() == 4 {
		visibleMask, err = buildAlphaVisibilityMask(img, alphaThreshold)
		if err != nil {
			return nil, err
		}
		defer visibleMask.Close()

		visibleGray, err := visibleLuminance(*gray, *visibleMask)
		if err != nil {
			return nil, err
		}
		defer visibleGray.Close()

		thresholdInput = visibleGray
	}

	binary := applyBinaryThreshold(*thresholdInput)

	if err := cleanMask(binary); err != nil {
		binary.Close()
		return nil, fmt.Errorf("clean binary mask: %w", err)
	}

	if visibleMask == nil {
		return binary, nil
	}

	final := gocv.NewMat()

	// Cleaning can fill transparent holes, so enforce visibility again.
	if err := gocv.BitwiseAnd(*binary, *visibleMask, &final); err != nil {
		final.Close()
		binary.Close()
		return nil, fmt.Errorf(
			"constrain binary mask to visible pixels: %w",
			err,
		)
	}

	binary.Close()
	return &final, nil
}

// buildAlphaVisibilityMask converts alpha into a binary visibility mask.
// Alpha values greater than threshold become 255; all others become 0.
// The caller owns the returned matrix.
func buildAlphaVisibilityMask(
	img gocv.Mat,
	threshold uint8,
) (*gocv.Mat, error) {
	if img.Channels() != 4 {
		return nil, errors.New(
			"alpha visibility mask requires a four-channel image",
		)
	}

	alpha := gocv.NewMat()
	defer alpha.Close()

	if err := gocv.ExtractChannel(img, &alpha, 3); err != nil {
		return nil, fmt.Errorf("extract alpha channel: %w", err)
	}

	visible := gocv.NewMat()
	gocv.Threshold(
		alpha,
		&visible,
		float32(threshold),
		255,
		gocv.ThresholdBinary,
	)

	return &visible, nil
}

// visibleLuminance returns a grayscale image with invisible pixels set to white.
// This prevents hidden color data from affecting Otsu's threshold.
// The caller owns the returned matrix.
func visibleLuminance(
	grayImg gocv.Mat,
	visibleMask gocv.Mat,
) (*gocv.Mat, error) {
	result := gocv.NewMatWithSizeFromScalar(
		gocv.NewScalar(255, 0, 0, 0),
		grayImg.Rows(),
		grayImg.Cols(),
		gocv.MatTypeCV8UC1,
	)

	if err := grayImg.CopyToWithMask(&result, visibleMask); err != nil {
		result.Close()
		return nil, fmt.Errorf(
			"copy visible pixels into grayscale image: %w",
			err,
		)
	}

	return &result, nil
}

// grayscale returns a single-channel luminance matrix.
// The caller owns the returned matrix.
func grayscale(img gocv.Mat) (*gocv.Mat, error) {
	channels := img.Channels()

	if channels == 1 {
		gray := img.Clone()
		return &gray, nil
	}

	gray := gocv.NewMat()

	var code gocv.ColorConversionCode
	switch channels {
	case 3:
		code = gocv.ColorBGRToGray
	case 4:
		code = gocv.ColorBGRAToGray
	default:
		return nil, fmt.Errorf(
			"unsupported image channel count: %d",
			channels,
		)
	}

	if err := gocv.CvtColor(img, &gray, code); err != nil {
		gray.Close()
		return nil, fmt.Errorf("convert image to grayscale: %w", err)
	}

	return &gray, nil
}

// applyBinaryThreshold uses inverse Otsu thresholding to select dark pixels.
// The caller owns the returned matrix.
func applyBinaryThreshold(img gocv.Mat) *gocv.Mat {
	binary := gocv.NewMat()
	gocv.Threshold(
		img,
		&binary,
		0,
		255,
		gocv.ThresholdBinaryInv|gocv.ThresholdOtsu,
	)
	return &binary
}

// cleanMask removes small artifacts and fills small gaps.
func cleanMask(binary *gocv.Mat) error {
	kernel := gocv.GetStructuringElement(
		gocv.MorphRect,
		image.Point{X: 3, Y: 3},
	)
	defer kernel.Close()

	if err := gocv.MorphologyEx(
		*binary,
		binary,
		gocv.MorphOpen,
		kernel,
	); err != nil {
		return fmt.Errorf("open binary mask: %w", err)
	}

	if err := gocv.MorphologyEx(
		*binary,
		binary,
		gocv.MorphClose,
		kernel,
	); err != nil {
		return fmt.Errorf("close binary mask: %w", err)
	}

	return nil
}

// createMask converts OpenCV matrices into the Mask representation.
// Ownership of both matrices transfers to the returned Mask.
func createMask(
	binaryMat,
	distMat *gocv.Mat,
) (*Mask, error) {
	width := binaryMat.Cols()
	height := binaryMat.Rows()

	binary := make([][]bool, height)
	distance := make([][]float32, height)

	binaryData, err := binaryMat.DataPtrUint8()
	if err != nil {
		return nil, fmt.Errorf("read binary mask data: %w", err)
	}

	distData, err := distMat.DataPtrFloat32()
	if err != nil {
		return nil, fmt.Errorf("read distance transform data: %w", err)
	}

	for y := range height {
		binary[y] = make([]bool, width)
		distance[y] = make([]float32, width)

		for x := range width {
			index := y*width + x
			binary[y][x] = binaryData[index] > 128
			distance[y][x] = distData[index]
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
