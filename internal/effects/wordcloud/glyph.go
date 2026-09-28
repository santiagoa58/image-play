package wordcloud

import (
	"errors"
	"fmt"
	"image"
	"math"

	"github.com/fogleman/gg"
	"github.com/santiagoa58/image-play/internal/textutil"
	"gocv.io/x/gocv"
)

// glyphFootprint holds the rendered pixels relative to the word's center.
// Placement still uses the measured rectangle; the raster is only needed once
// a large word has an accepted position.
type glyphFootprint struct {
	mask   gocv.Mat
	offset image.Point
}

func (f *glyphFootprint) Close() { f.mask.Close() }

func rasterizeGlyph(word textutil.Word, fontPath string, angle, padding int) (glyphFootprint, error) {
	margin := int(math.Ceil(word.FontSize)) + padding + 4
	width, height := int(math.Ceil(word.Width))+2*margin, int(math.Ceil(word.Height))+2*margin
	if angle == 90 {
		width, height = height, width
	}
	center := image.Pt(width/2, height/2)
	dc := gg.NewContext(width, height)
	if err := dc.LoadFontFace(fontPath, word.FontSize); err != nil {
		return glyphFootprint{}, fmt.Errorf("load glyph font: %w", err)
	}
	dc.SetRGBA(1, 1, 1, 1)
	dc.Translate(float64(center.X), float64(center.Y))
	dc.Rotate(float64(angle) * math.Pi / 180)
	dc.DrawStringAnchored(word.Text, 0, 0, 0.5, 0.5)

	mask := gocv.NewMatWithSize(height, width, gocv.MatTypeCV8UC1)
	pixels, err := mask.DataPtrUint8()
	if err != nil {
		mask.Close()
		return glyphFootprint{}, err
	}
	img := dc.Image().(*image.RGBA)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if img.Pix[y*img.Stride+x*4+3] != 0 {
				pixels[y*width+x] = 255
			}
		}
	}
	if padding > 0 {
		kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(2*padding+1, 2*padding+1))
		defer kernel.Close()
		grown := gocv.NewMat()
		if err := gocv.Dilate(mask, &grown, kernel); err != nil {
			mask.Close()
			grown.Close()
			return glyphFootprint{}, err
		}
		mask.Close()
		mask = grown
		pixels, err = mask.DataPtrUint8()
		if err != nil {
			mask.Close()
			return glyphFootprint{}, err
		}
	}

	minX, minY, maxX, maxY := width, height, -1, -1
	for index, pixel := range pixels {
		if pixel == 0 {
			continue
		}
		x, y := index%width, index/width
		minX, minY = min(minX, x), min(minY, y)
		maxX, maxY = max(maxX, x), max(maxY, y)
	}
	if maxX < 0 || minX == 0 || minY == 0 || maxX == width-1 || maxY == height-1 {
		mask.Close()
		return glyphFootprint{}, errors.New("glyph raster is empty or clipped")
	}
	rect := image.Rect(minX, minY, maxX+1, maxY+1)
	region := mask.Region(rect)
	trimmed := region.Clone()
	region.Close()
	mask.Close()
	offset := rect.Min.Sub(center)
	return glyphFootprint{
		mask: trimmed, offset: offset,
	}, nil
}
