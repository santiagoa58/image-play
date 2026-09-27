package imageutil

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"gocv.io/x/gocv"
)

func TestSubjectGeometryPreservesShadowsIndependentlyOfDetail(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 40, 30))
	for y := 0; y < 30; y++ {
		for x := 0; x < 40; x++ {
			c := color.NRGBA{A: 255}
			if x >= 5 && x < 35 && y >= 5 && y < 25 {
				c.R = 35
			}
			if x >= 15 && x < 25 && y >= 10 && y < 20 {
				c.R = 0
			}
			source.SetNRGBA(x, y, c)
		}
	}
	analysis := analyzeSubject(source, 8)
	if analysis.method != "border" {
		t.Fatalf("selection = %q", analysis.method)
	}
	if got := analysis.subject.GrayAt(20, 15).Y; got != 255 {
		t.Fatalf("enclosed shadow excluded from subject: %d", got)
	}
	if got := analysis.detail.GrayAt(20, 15).Y; got != 0 {
		t.Fatalf("shadow contrast = %d", got)
	}
	if got := analysis.subject.GrayAt(8, 8).Y; got != 255 {
		t.Fatalf("dark red subject excluded: %d", got)
	}
	if got := analysis.detail.GrayAt(8, 8).Y; got != 35 {
		t.Fatalf("dark red contrast = %d", got)
	}
	if got := analysis.subject.GrayAt(0, 0).Y; got != 0 {
		t.Fatalf("exterior background included: %d", got)
	}
}

func TestAlphaSubjectKeepsLightAndDarkPixelsAndStrictThreshold(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	source.SetNRGBA(3, 3, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	source.SetNRGBA(4, 3, color.NRGBA{A: 255})
	source.SetNRGBA(5, 3, color.NRGBA{R: 255, A: 8})
	source.SetNRGBA(6, 3, color.NRGBA{R: 255, A: 9})
	analysis := analyzeSubject(source, 8)
	for _, p := range []struct {
		x    int
		want uint8
	}{{3, 255}, {4, 255}, {5, 0}, {6, 255}} {
		if got := analysis.subject.GrayAt(p.x, 3).Y; got != p.want {
			t.Errorf("subject(%d,3) = %d, want %d", p.x, got, p.want)
		}
	}
}

func TestBusyBorderRetainsSceneWithoutPretendingToFindSubject(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 20, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			c := color.NRGBA{R: 220, G: 30, A: 255}
			if x >= 10 {
				c = color.NRGBA{G: 30, B: 220, A: 255}
			}
			source.SetNRGBA(x, y, c)
		}
	}
	analysis := analyzeSubject(source, 8)
	if analysis.method != "scene" {
		t.Fatalf("method = %q, want scene", analysis.method)
	}
	for _, v := range analysis.subject.Pix {
		if v != 255 {
			t.Fatal("scene geometry was thresholded")
		}
	}
}

func TestSubjectMaskRequiresMatchingDimensionsAndRespectsSourceAlpha(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: 200, A: 255})
		}
	}
	source.SetNRGBA(5, 5, color.NRGBA{R: 255, A: 0})
	analysis := analyzeSubject(source, 8)
	mask := gocv.NewMatWithSizeFromScalar(gocv.NewScalar(255, 0, 0, 0), 10, 10, gocv.MatTypeCV8UC1)
	defer mask.Close()
	mask.SetUCharAt(3, 3, 0)
	path := filepath.Join(t.TempDir(), "mask.png")
	if !gocv.IMWrite(path, mask) {
		t.Fatal("write mask")
	}
	if err := applySubjectMask(&analysis, source, path, 8); err != nil {
		t.Fatal(err)
	}
	if analysis.method != "mask" || analysis.subject.GrayAt(3, 3).Y != 0 || analysis.subject.GrayAt(5, 5).Y != 0 || analysis.subject.GrayAt(4, 4).Y != 255 {
		t.Fatal("mask or visibility not respected")
	}
	smaller := gocv.NewMatWithSize(2, 2, gocv.MatTypeCV8UC1)
	defer smaller.Close()
	if !gocv.IMWrite(path, smaller) {
		t.Fatal("write smaller mask")
	}
	if err := applySubjectMask(&analysis, source, path, 8); err == nil {
		t.Fatal("mismatched mask accepted")
	}
}

func TestSubjectMaskChoosesCanvasForSelectedSubject(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 20, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			c := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			if y >= 10 {
				c.B = 0
			}
			if x >= 8 && x < 12 && y >= 8 && y < 12 {
				c = color.NRGBA{A: 255}
			}
			source.SetNRGBA(x, y, c)
		}
	}
	analysis := analyzeSubject(source, 8)
	if analysis.method != "scene" || analysis.background != (color.NRGBA{A: 255}) {
		t.Fatalf("expected bright scene on black, got %s on %v", analysis.method, analysis.background)
	}
	mask := gocv.NewMatWithSize(20, 20, gocv.MatTypeCV8UC1)
	defer mask.Close()
	for y := 8; y < 12; y++ {
		for x := 8; x < 12; x++ {
			mask.SetUCharAt(y, x, 255)
		}
	}
	path := filepath.Join(t.TempDir(), "mask.png")
	if !gocv.IMWrite(path, mask) {
		t.Fatal("write subject mask")
	}
	if err := applySubjectMask(&analysis, source, path, 8); err != nil {
		t.Fatal(err)
	}
	if analysis.background != (color.NRGBA{R: 255, G: 255, B: 255, A: 255}) || analysis.detail.GrayAt(9, 9).Y != 255 {
		t.Fatal("selected dark subject did not get a contrasting canvas")
	}
}

func TestPrepareMaskEveryImageFixture(t *testing.T) {
	inputs, err := filepath.Glob("../../testdata/images/*")
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) == 0 {
		t.Fatal("no image fixtures")
	}
	for _, path := range inputs {
		t.Run(filepath.Base(path), func(t *testing.T) {
			if info, err := os.Stat(path); err != nil || info.IsDir() {
				t.Skip("not an image file")
			}
			mask, err := PrepareMask(path, 8)
			if err != nil {
				t.Fatal(err)
			}
			defer mask.Close()
			if mask.Width != mask.Source.Bounds().Dx() || mask.Height != mask.Source.Bounds().Dy() {
				t.Fatal("dimensions differ")
			}
			if gocv.CountNonZero(*mask.BinaryMat) == 0 {
				t.Fatal("empty placement mask")
			}
			for y := 0; y < mask.Height; y++ {
				for x := 0; x < mask.Width; x++ {
					if mask.BinaryMat.GetUCharAt(y, x) == 0 {
						continue
					}
					if mask.SubjectMat.GetUCharAt(y, x) == 0 {
						t.Fatal("placement escaped subject")
					}
					_, _, _, a := mask.Source.At(x, y).RGBA()
					if a <= 8*257 {
						t.Fatal("transparent placement pixel")
					}
				}
			}
			t.Logf("%s: %dx%d, method=%s, subject=%d, placeable=%d", filepath.Base(path), mask.Width, mask.Height, mask.SelectionMethod, gocv.CountNonZero(*mask.SubjectMat), gocv.CountNonZero(*mask.BinaryMat))
		})
	}
}

func TestTransparentFrameDoesNotMakeOpaqueBackgroundPartOfLogo(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 30, 30))
	for y := 1; y < 29; y++ {
		for x := 1; x < 29; x++ {
			c := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			if x >= 8 && x < 22 && y >= 8 && y < 22 {
				c = color.NRGBA{R: 60, G: 80, B: 255, A: 255}
			}
			source.SetNRGBA(x, y, c)
		}
	}
	analysis := analyzeSubject(source, 8)
	if analysis.subject.GrayAt(2, 2).Y != 0 || analysis.subject.GrayAt(15, 15).Y != 255 {
		t.Fatal("opaque panel confused with subject cutout")
	}
	if analysis.background != (color.NRGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("background = %v", analysis.background)
	}
}

func TestFullyTransparentInputHasNoSubject(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255})
		}
	}
	analysis := analyzeSubject(source, 8)
	if hasSelectedPixels(analysis.subject.Pix) {
		t.Fatal("hidden white became subject geometry")
	}
	for _, v := range analysis.detail.Pix {
		if v != 0 {
			t.Fatal("hidden RGB became detail")
		}
	}
}
