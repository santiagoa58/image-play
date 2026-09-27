package imageutil

import (
	"fmt"
	"image"
	"image/color"

	"gocv.io/x/gocv"
)

// Mask keeps subject geometry separate from source shading and placement space.
// The caller owns its OpenCV matrices and must call Close.
type Mask struct {
	Width, Height   int
	Source          image.Image
	Background      color.NRGBA
	DarkBackground  bool
	SelectionMethod string
	// SubjectMat includes enclosed shadows and visible alpha-cutout pixels.
	SubjectMat *gocv.Mat
	// DetailMat records source contrast against Background, independently of shape.
	DetailMat *gocv.Mat
	// BinaryMat admits visible, contrasting detail within the subject.
	BinaryMat *gocv.Mat
	DistMat   *gocv.Mat
}

func (m *Mask) Close() {
	m.Source = nil
	for _, mat := range []*gocv.Mat{m.SubjectMat, m.DetailMat, m.BinaryMat, m.DistMat} {
		if mat != nil {
			mat.Close()
		}
	}
	m.SubjectMat, m.DetailMat, m.BinaryMat, m.DistMat = nil, nil, nil, nil
}

func PrepareMask(path string, alphaThreshold uint8) (*Mask, error) {
	return PrepareMaskWithSubject(path, "", alphaThreshold)
}

// PrepareMaskWithSubject optionally uses a user-supplied mask: white means
// subject, black means background. The mask must match the source dimensions.
func PrepareMaskWithSubject(path, subjectPath string, alphaThreshold uint8) (*Mask, error) {
	img, err := readImage(path)
	if err != nil {
		return nil, err
	}
	defer img.Close()
	source, err := img.ToImage()
	if err != nil {
		return nil, fmt.Errorf("preserve source colors: %w", err)
	}
	analysis := analyzeSubject(source, alphaThreshold)
	if subjectPath != "" {
		if err := applySubjectMask(&analysis, source, subjectPath, alphaThreshold); err != nil {
			return nil, err
		}
	}
	return maskFromAnalysis(source, analysis)
}

func readImage(path string) (*gocv.Mat, error) {
	img := gocv.IMRead(path, gocv.IMReadUnchanged)
	if img.Empty() {
		img.Close()
		return nil, fmt.Errorf("failed to read image from %q", path)
	}
	return &img, nil
}

func applySubjectMask(analysis *subjectAnalysis, source image.Image, path string, alphaThreshold uint8) error {
	mat, err := readImage(path)
	if err != nil {
		return fmt.Errorf("read subject mask: %w", err)
	}
	defer mat.Close()
	supplied, err := mat.ToImage()
	if err != nil {
		return fmt.Errorf("decode subject mask: %w", err)
	}
	if supplied.Bounds().Size() != source.Bounds().Size() {
		return fmt.Errorf("subject mask dimensions %v must match source %v", supplied.Bounds().Size(), source.Bounds().Size())
	}
	visible := visibilityMap(source, alphaThreshold)
	bounds := supplied.Bounds()
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			c := sourceColor(supplied, bounds.Min.X+x, bounds.Min.Y+y)
			index := y*bounds.Dx() + x
			analysis.subject.Pix[index] = 0
			if c.A > alphaThreshold && color.GrayModel.Convert(c).(color.Gray).Y > 127 {
				analysis.subject.Pix[index] = visible[index]
			}
		}
	}
	if analysis.method == "scene" || analysis.method == "alpha" {
		analysis.background = contrastingCanvas(source, analysis.subject.Pix)
		analysis.detail = contrastMap(source, analysis.background)
	}
	analysis.method = "mask"
	return nil
}

func maskFromAnalysis(source image.Image, analysis subjectAnalysis) (*Mask, error) {
	bounds := source.Bounds()
	mask := &Mask{Width: bounds.Dx(), Height: bounds.Dy(), Source: source, Background: analysis.background, SelectionMethod: analysis.method}
	mask.DarkBackground = color.GrayModel.Convert(analysis.background).(color.Gray).Y < 128
	subject, err := gocv.NewMatFromBytes(mask.Height, mask.Width, gocv.MatTypeCV8UC1, analysis.subject.Pix)
	if err != nil {
		return nil, err
	}
	mask.SubjectMat = &subject
	detail, err := gocv.NewMatFromBytes(mask.Height, mask.Width, gocv.MatTypeCV8UC1, analysis.detail.Pix)
	if err != nil {
		mask.Close()
		return nil, err
	}
	mask.DetailMat = &detail
	pixels := make([]byte, len(analysis.subject.Pix))
	for i, allowed := range analysis.subject.Pix {
		if allowed != 0 && analysis.detail.Pix[i] > 12 {
			pixels[i] = 255
		}
	}
	binary, err := gocv.NewMatFromBytes(mask.Height, mask.Width, gocv.MatTypeCV8UC1, pixels)
	if err != nil {
		mask.Close()
		return nil, err
	}
	mask.BinaryMat = &binary
	excluded := append([]byte(nil), pixels...)
	if err := cleanMask(mask.BinaryMat); err != nil {
		mask.Close()
		return nil, err
	}
	// Cleanup may bridge tiny holes; enforce geometry and contrast again.
	for i, pixel := range excluded {
		if pixel == 0 {
			binary.SetUCharAt(i/mask.Width, i%mask.Width, 0)
		}
	}
	mask.DistMat, err = ComputeDistanceTransform(binary)
	if err != nil {
		mask.Close()
		return nil, err
	}
	return mask, nil
}

// buildBinaryMask is also used by synthetic-image regression tests.
func buildBinaryMask(img gocv.Mat, alphaThreshold uint8) (*gocv.Mat, error) {
	binary, _, err := prepareBinaryMask(img, alphaThreshold)
	return binary, err
}

func prepareBinaryMask(img gocv.Mat, alphaThreshold uint8) (*gocv.Mat, bool, error) {
	source, err := img.ToImage()
	if err != nil {
		return nil, false, err
	}
	mask, err := maskFromAnalysis(source, analyzeSubject(source, alphaThreshold))
	if err != nil {
		return nil, false, err
	}
	binary, dark := mask.BinaryMat, mask.DarkBackground
	mask.BinaryMat = nil
	mask.Close()
	return binary, dark, nil
}

func cleanMask(binary *gocv.Mat) error {
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
	defer kernel.Close()
	if err := gocv.MorphologyEx(*binary, binary, gocv.MorphOpen, kernel); err != nil {
		return fmt.Errorf("open binary mask: %w", err)
	}
	if err := gocv.MorphologyEx(*binary, binary, gocv.MorphClose, kernel); err != nil {
		return fmt.Errorf("close binary mask: %w", err)
	}
	return nil
}
