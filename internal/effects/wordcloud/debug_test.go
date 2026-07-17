package wordcloud

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/santiagoa58/image-play/internal/imageutil"
	"gocv.io/x/gocv"
)

func TestDebugWritersCreateNumberedImages(t *testing.T) {
	const size = 9

	outputPath := filepath.Join(t.TempDir(), "result.jpg")

	binary := gocv.NewMatWithSizeFromScalar(
		gocv.NewScalar(255, 0, 0, 0),
		size,
		size,
		gocv.MatTypeCV8UC1,
	)
	defer binary.Close()

	distance := gocv.NewMatWithSize(
		size,
		size,
		gocv.MatTypeCV32F,
	)
	defer distance.Close()
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			distance.SetFloatAt(y, x, float32(y*size+x))
		}
	}

	mask := &imageutil.Mask{
		BinaryMat: &binary,
		DistMat:   &distance,
	}
	centers := []Center{{Point: image.Pt(size/2, size/2)}}

	safeZone := binary.Clone()
	defer safeZone.Close()
	occupancy := gocv.NewMatWithSize(
		size,
		size,
		gocv.MatTypeCV8UC1,
	)
	defer occupancy.Close()
	occupancy.SetUCharAt(size/2, size/2, 255)

	rendered := image.NewRGBA(image.Rect(0, 0, size, size))

	if err := writeMaskDebug(true, outputPath, mask); err != nil {
		t.Fatalf("writeMaskDebug() error = %v", err)
	}
	if err := writeDistanceDebug(true, outputPath, mask); err != nil {
		t.Fatalf("writeDistanceDebug() error = %v", err)
	}
	if err := writeCentersDebug(
		true,
		outputPath,
		mask,
		centers,
	); err != nil {
		t.Fatalf("writeCentersDebug() error = %v", err)
	}
	if err := writePlacementDebug(
		true,
		outputPath,
		safeZone,
		occupancy,
	); err != nil {
		t.Fatalf("writePlacementDebug() error = %v", err)
	}
	if err := writeWordcloudDebug(
		true,
		outputPath,
		rendered,
	); err != nil {
		t.Fatalf("writeWordcloudDebug() error = %v", err)
	}

	for _, label := range []string{
		"01-mask",
		"02-distance",
		"03-centers",
		"04-safe-zone",
		"05-occupancy",
		"06-wordcloud",
	} {
		path := debugPath(outputPath, label)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("debug image %q was not created: %v", path, err)
		}
	}

	distanceImage := gocv.IMRead(
		debugPath(outputPath, "02-distance"),
		gocv.IMReadGrayScale,
	)
	defer distanceImage.Close()
	if distanceImage.Empty() {
		t.Fatal("distance debug image could not be read")
	}
	_, maxDistance, _, _ := gocv.MinMaxLoc(distanceImage)
	if maxDistance != 255 {
		t.Errorf(
			"normalized distance maximum = %v, want 255",
			maxDistance,
		)
	}

	centersImage := gocv.IMRead(
		debugPath(outputPath, "03-centers"),
		gocv.IMReadColor,
	)
	defer centersImage.Close()
	if centersImage.Empty() {
		t.Fatal("centers debug image could not be read")
	}
	if got := centersImage.Channels(); got != 3 {
		t.Errorf("centers debug channels = %d, want 3", got)
	}

	centerPixel := centersImage.GetVecbAt(size/2, size/2)
	if centerPixel[0] != 0 ||
		centerPixel[1] != 0 ||
		centerPixel[2] != 255 {
		t.Errorf(
			"center marker BGR = %v, want [0 0 255]",
			centerPixel,
		)
	}
}

func TestDebugWritersAreNoOpWhenDisabled(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "result.png")

	if err := writeMaskDebug(false, outputPath, nil); err != nil {
		t.Fatalf("writeMaskDebug(false) error = %v", err)
	}
	if err := writeDistanceDebug(false, outputPath, nil); err != nil {
		t.Fatalf("writeDistanceDebug(false) error = %v", err)
	}
	if err := writeCentersDebug(
		false,
		outputPath,
		nil,
		nil,
	); err != nil {
		t.Fatalf("writeCentersDebug(false) error = %v", err)
	}

	empty := gocv.NewMat()
	defer empty.Close()
	if err := writePlacementDebug(
		false,
		outputPath,
		empty,
		empty,
	); err != nil {
		t.Fatalf("writePlacementDebug(false) error = %v", err)
	}
	if err := writeWordcloudDebug(
		false,
		outputPath,
		nil,
	); err != nil {
		t.Fatalf("writeWordcloudDebug(false) error = %v", err)
	}

	if _, err := os.Stat(debugPath(outputPath, "01-mask")); !os.IsNotExist(err) {
		t.Errorf("disabled debug output exists or stat failed: %v", err)
	}
}
