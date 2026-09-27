package wordcloud

import (
	"image"
	"image/color"
	"math"
)

type colorBucket struct {
	weight           uint64
	red, green, blue uint64
	samples          []color.NRGBA
}

// sampleGlyphColor considers only pixels covered by the rendered letters.
// Representative mode chooses a real source color from the dominant local
// color family, weighted by visibility and contrast against the canvas.
func sampleGlyphColor(source image.Image, word PlacedWord, glyph image.Image, background color.NRGBA, mode string) color.NRGBA {
	buckets := make(map[int]*colorBucket)
	mean := &colorBucket{}
	bounds := glyph.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, coverage := glyph.At(x, y).RGBA()
			if coverage == 0 {
				continue
			}
			point := glyphSourcePoint(source.Bounds(), word, bounds, x, y)
			if !point.In(source.Bounds()) {
				continue
			}
			c := color.NRGBAModel.Convert(source.At(point.X, point.Y)).(color.NRGBA)
			weight := uint64(coverage) * uint64(c.A)
			if weight == 0 {
				continue
			}
			mean.add(c, weight)
			key := int(c.R/16)<<8 | int(c.G/16)<<4 | int(c.B/16)
			bucket := buckets[key]
			if bucket == nil {
				bucket = &colorBucket{}
				buckets[key] = bucket
			}
			contrast := max(absChannel(int(c.R)-int(background.R)), absChannel(int(c.G)-int(background.G)), absChannel(int(c.B)-int(background.B)))
			bucket.add(c, weight*uint64(max(1, contrast)))
			bucket.samples = append(bucket.samples, c)
		}
	}
	if mode == "mean" {
		return mean.average()
	}
	bestKey := -1
	var best *colorBucket
	for key, bucket := range buckets {
		if best == nil || bucket.weight > best.weight || bucket.weight == best.weight && key < bestKey {
			bestKey, best = key, bucket
		}
	}
	if best == nil {
		return mean.average()
	}
	return best.representative()
}

func glyphSourcePoint(source image.Rectangle, word PlacedWord, glyph image.Rectangle, x, y int) image.Point {
	dx := float64(x-glyph.Min.X) + 0.5 - float64(glyph.Dx())/2
	dy := float64(y-glyph.Min.Y) + 0.5 - float64(glyph.Dy())/2
	if word.Angle == 90 {
		dx, dy = -dy, dx
	}
	return image.Pt(source.Min.X+int(math.Floor(word.X+dx)), source.Min.Y+int(math.Floor(word.Y+dy)))
}

func (b *colorBucket) add(c color.NRGBA, weight uint64) {
	b.weight += weight
	b.red += uint64(c.R) * weight
	b.green += uint64(c.G) * weight
	b.blue += uint64(c.B) * weight
}
func (b *colorBucket) average() color.NRGBA {
	if b.weight == 0 {
		return color.NRGBA{A: 255}
	}
	return color.NRGBA{R: uint8(b.red / b.weight), G: uint8(b.green / b.weight), B: uint8(b.blue / b.weight), A: 255}
}
func (b *colorBucket) representative() color.NRGBA {
	mean := b.average()
	best := mean
	distance := math.MaxInt
	for _, c := range b.samples {
		d := square(int(c.R)-int(mean.R)) + square(int(c.G)-int(mean.G)) + square(int(c.B)-int(mean.B))
		if d < distance {
			best, distance = c, d
		}
	}
	best.A = 255
	return best
}
func square(v int) int { return v * v }
func absChannel(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
