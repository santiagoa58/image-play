package imageutil

import (
	"errors"
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

// Mask contains a binary placement shape and its distance transform.
type Mask struct {
	Width  int
	Height int

	// Source preserves the original colors for rendering, independently of OpenCV.
	Source image.Image
	// DarkBackground selects bright foreground and a black rendering canvas.
	DarkBackground bool

	// BinaryMat is the placeable silhouette; DistMat is its distance transform.
	BinaryMat *gocv.Mat
	DistMat   *gocv.Mat
}

// Close releases the OpenCV matrices and drops the preserved source image.
func (m *Mask) Close() {
	m.Source = nil
	if m.BinaryMat != nil {
		m.BinaryMat.Close()
		m.BinaryMat = nil
	}
	if m.DistMat != nil {
		m.DistMat.Close()
		m.DistMat = nil
	}
}

// PrepareMask builds the placement silhouette and distance transform for an
// image.
//
// The current segmentation is intentionally simple and deterministic:
//   - preserve alpha when loading;
//   - convert visible pixels to luminance;
//   - use Otsu thresholding, selecting the brightness group opposite the border;
//   - use the strongest RGB channel for colored detail against dark backgrounds;
//   - clean small artifacts with morphological open/close operations;
//   - reapply alpha so transparent pixels can never become placeable;
//   - compute a Euclidean distance transform for placement-center discovery.
//
// The caller owns the returned Mask and must close it.
func PrepareMask(path string, alphaThreshold uint8) (*Mask, error) {
	img, err := readImage(path)
	if err != nil {
		return nil, fmt.Errorf("read image from %q: %w", path, err)
	}
	defer img.Close()

	binary, darkBackground, err := prepareBinaryMask(*img, alphaThreshold)
	if err != nil {
		return nil, fmt.Errorf("build binary mask: %w", err)
	}

	source, err := img.ToImage()
	if err != nil {
		binary.Close()
		return nil, fmt.Errorf("preserve source colors: %w", err)
	}

	dist, err := ComputeDistanceTransform(*binary)
	if err != nil {
		binary.Close()
		return nil, fmt.Errorf("compute distance transform: %w", err)
	}

	return &Mask{
		Source:         source,
		DarkBackground: darkBackground,
		Width:          binary.Cols(),
		Height:         binary.Rows(),
		BinaryMat:      binary,
		DistMat:        dist,
	}, nil
}

// readImage loads an image without discarding its alpha channel.
func readImage(path string) (*gocv.Mat, error) {
	img := gocv.IMRead(path, gocv.IMReadUnchanged)
	if img.Empty() {
		img.Close()
		return nil, fmt.Errorf("failed to read image from %q", path)
	}
	return &img, nil
}

// buildBinaryMask returns a cleaned binary placement mask.
// When alpha is present, invisible pixels are excluded.
// The caller owns the returned matrix.
func buildBinaryMask(img gocv.Mat, alphaThreshold uint8) (*gocv.Mat, error) {
	binary, _, err := prepareBinaryMask(img, alphaThreshold)
	return binary, err
}

func prepareBinaryMask(
	img gocv.Mat,
	alphaThreshold uint8,
) (*gocv.Mat, bool, error) {
	gray, err := grayscale(img)
	if err != nil {
		return nil, false, err
	}
	defer gray.Close()

	var visibleMask *gocv.Mat
	if img.Channels() == 4 {
		visibleMask, err = buildAlphaVisibilityMask(img, alphaThreshold)
		if err != nil {
			return nil, false, err
		}
		defer visibleMask.Close()
	}

	binary, darkBackground, err := thresholdForeground(img, *gray, visibleMask)
	if err != nil {
		return nil, false, err
	}

	if err := cleanMask(binary); err != nil {
		binary.Close()
		return nil, false, fmt.Errorf("clean binary mask: %w", err)
	}

	if visibleMask == nil {
		return binary, darkBackground, nil
	}

	final := gocv.NewMat()

	// Cleaning can fill transparent holes, so enforce visibility again.
	if err := gocv.BitwiseAnd(*binary, *visibleMask, &final); err != nil {
		final.Close()
		binary.Close()
		return nil, false, fmt.Errorf(
			"constrain binary mask to visible pixels: %w",
			err,
		)
	}

	binary.Close()
	return &final, darkBackground, nil
}

// thresholdForeground chooses the brightness group opposite the visible border.
func thresholdForeground(img, gray gocv.Mat, visibility *gocv.Mat) (*gocv.Mat, bool, error) {
	if visibility != nil {
		visibleGray, err := visibleLuminance(gray, *visibility)
		if err != nil {
			return nil, false, err
		}
		defer visibleGray.Close()
		gray = *visibleGray
	}
	binary := applyBinaryThreshold(gray)
	if hasUniformLuminance(gray, visibility) || !borderSelectsDarkPixels(*binary, visibility) {
		return binary, false, nil
	}
	binary.Close()
	bright, err := thresholdBrightColors(img, visibility)
	return bright, true, err
}

// thresholdBrightColors selects colored detail against a dark background.
func thresholdBrightColors(img gocv.Mat, visibility *gocv.Mat) (*gocv.Mat, error) {
	intensity, err := colorIntensity(img)
	if err != nil {
		return nil, err
	}
	defer intensity.Close()
	input := intensity
	if visibility != nil {
		visibleIntensity, err := luminanceOnBackground(*intensity, *visibility, 0)
		if err != nil {
			return nil, err
		}
		defer visibleIntensity.Close()
		input = visibleIntensity
	}
	binary := gocv.NewMat()
	gocv.Threshold(*input, &binary, 0, 255, gocv.ThresholdBinary|gocv.ThresholdOtsu)
	return &binary, nil
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
	return luminanceOnBackground(grayImg, visibleMask, 255)
}

func luminanceOnBackground(grayImg, visibleMask gocv.Mat, background uint8) (*gocv.Mat, error) {
	result := gocv.NewMatWithSizeFromScalar(
		gocv.NewScalar(float64(background), 0, 0, 0),
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

// borderSelectsDarkPixels infers background from the perimeter, counting each
// visible border pixel once. Transparent borders and uniform images retain the
// dark-foreground fallback: there is no evidence for switching polarity.
func borderSelectsDarkPixels(inverse gocv.Mat, visibility *gocv.Mat) bool {
	selected := gocv.CountNonZero(inverse)
	if selected == 0 || selected == inverse.Rows()*inverse.Cols() {
		return false
	}
	dark, visible := 0, 0
	count := func(x, y int) {
		if visibility != nil && visibility.GetUCharAt(y, x) == 0 {
			return
		}
		visible++
		if inverse.GetUCharAt(y, x) != 0 {
			dark++
		}
	}
	for x := 0; x < inverse.Cols(); x++ {
		count(x, 0)
		if inverse.Rows() > 1 {
			count(x, inverse.Rows()-1)
		}
	}
	for y := 1; y < inverse.Rows()-1; y++ {
		count(0, y)
		if inverse.Cols() > 1 {
			count(inverse.Cols()-1, y)
		}
	}
	return visible > 0 && dark*2 > visible
}

// colorIntensity uses the strongest RGB channel so saturated colors, especially
// red and blue, are not mistaken for dark background by grayscale weighting.
// Alpha is handled separately by the visibility mask.
func colorIntensity(img gocv.Mat) (*gocv.Mat, error) {
	if img.Channels() == 1 {
		result := img.Clone()
		return &result, nil
	}
	channels := gocv.Split(img)
	defer func() {
		for i := range channels {
			channels[i].Close()
		}
	}()
	intensity := channels[0].Clone()
	for _, channel := range channels[1:3] {
		if err := gocv.Max(intensity, channel, &intensity); err != nil {
			intensity.Close()
			return nil, fmt.Errorf("combine color intensity channels: %w", err)
		}
	}
	return &intensity, nil
}

// hasUniformLuminance avoids inferring a background from hidden RGB or from an
// image with no visible brightness contrast.
func hasUniformLuminance(gray gocv.Mat, visibility *gocv.Mat) bool {
	var low, high float32
	if visibility == nil {
		low, high, _, _ = gocv.MinMaxLoc(gray)
	} else {
		low, high, _, _ = gocv.MinMaxLocWithMask(gray, *visibility)
	}
	return low == high
}
