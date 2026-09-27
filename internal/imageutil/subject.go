package imageutil

import (
	"image"
	"image/color"
	"sort"
)

// subjectAnalysis separates geometry from the source's contrast against the
// rendering canvas. Low-contrast shading belongs to the subject even when it
// provides no useful space for visible words.
type subjectAnalysis struct {
	subject    *image.Gray
	detail     *image.Gray
	background color.NRGBA
	method     string
}

func analyzeSubject(source image.Image, alphaThreshold uint8) subjectAnalysis {
	bounds := source.Bounds()
	visible := visibilityMap(source, alphaThreshold)
	background, tolerance, simple := borderBackground(source, visible)
	subject := image.NewGray(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	method := "border"
	switch {
	case hasTransparentBorder(visible, bounds.Dx(), bounds.Dy()) && !(simple && fillsVisibleRectangle(visible, bounds.Dx(), bounds.Dy())):
		copy(subject.Pix, visible)
		background = contrastingCanvas(source, visible)
		method = "alpha"
	case simple:
		removeConnectedBackground(source, visible, subject, background, tolerance)
	default:
		copy(subject.Pix, visible)
		background = contrastingCanvas(source, visible)
		method = "scene"
	}
	// A uniform opaque image has no separable background. Preserve its visible
	// area rather than returning an empty shape.
	if !hasSelectedPixels(subject.Pix) {
		copy(subject.Pix, visible)
		background = contrastingCanvas(source, visible)
		method = "scene"
	}
	return subjectAnalysis{subject: subject, detail: contrastMap(source, background), background: background, method: method}
}

func visibilityMap(source image.Image, threshold uint8) []uint8 {
	bounds := source.Bounds()
	visible := make([]uint8, bounds.Dx()*bounds.Dy())
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			_, _, _, alpha := source.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			if alpha > uint32(threshold)*257 {
				visible[y*bounds.Dx()+x] = 255
			}
		}
	}
	return visible
}

func hasTransparentBorder(visible []uint8, width, height int) bool {
	transparent, total := 0, 0
	visitBorder(width, height, func(x, y int) {
		total++
		if visible[y*width+x] == 0 {
			transparent++
		}
	})
	return transparent*5 >= total && transparent > 0
}

// borderBackground fits the dominant border color, allowing small compression
// noise. Several unrelated border colors indicate a scene, not a simple cutout.
func borderBackground(source image.Image, visible []uint8) (color.NRGBA, int, bool) {
	type bucket struct{ count, r, g, b int }
	buckets := make(map[int]bucket)
	bounds := source.Bounds()
	total := 0
	border := visibleBounds(visible, bounds.Dx(), bounds.Dy())
	visitVisibleBorder := func(visit func(x, y int)) {
		visitBorder(border.Dx(), border.Dy(), func(x, y int) { visit(x+border.Min.X, y+border.Min.Y) })
	}
	visitVisibleBorder(func(x, y int) {
		if visible[y*bounds.Dx()+x] == 0 {
			return
		}
		c := sourceColor(source, bounds.Min.X+x, bounds.Min.Y+y)
		key := int(c.R/16)<<8 | int(c.G/16)<<4 | int(c.B/16)
		b := buckets[key]
		b.count++
		b.r += int(c.R)
		b.g += int(c.G)
		b.b += int(c.B)
		buckets[key] = b
		total++
	})
	bestKey, bestCount := -1, 0
	for key, b := range buckets {
		if b.count > bestCount || b.count == bestCount && key < bestKey {
			bestKey, bestCount = key, b.count
		}
	}
	if bestCount == 0 {
		return color.NRGBA{A: 255}, 12, false
	}
	b := buckets[bestKey]
	background := color.NRGBA{R: uint8(b.r / b.count), G: uint8(b.g / b.count), B: uint8(b.b / b.count), A: 255}
	distances := make([]int, 0, total)
	visitVisibleBorder(func(x, y int) {
		if visible[y*bounds.Dx()+x] == 0 {
			return
		}
		distance := colorDistance(sourceColor(source, bounds.Min.X+x, bounds.Min.Y+y), background)
		if distance <= 32 {
			distances = append(distances, distance)
		}
	})
	if len(distances)*100 < total*65 {
		return background, 12, false
	}
	sort.Ints(distances)
	tolerance := max(12, distances[(len(distances)-1)*9/10]+4)
	return background, tolerance, true
}

// removeConnectedBackground removes only background-like pixels reachable from
// the perimeter. Enclosed shadows remain part of the subject geometry.
func removeConnectedBackground(source image.Image, visible []uint8, subject *image.Gray, background color.NRGBA, tolerance int) {
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	exterior := make([]bool, len(visible))
	queue := make([]int, 0, len(visible)/2)
	enqueue := func(x, y int) {
		index := y*width + x
		if exterior[index] {
			return
		}
		if visible[index] != 0 && colorDistance(sourceColor(source, bounds.Min.X+x, bounds.Min.Y+y), background) > tolerance {
			return
		}
		exterior[index] = true
		queue = append(queue, index)
	}
	visitBorder(width, height, enqueue)
	for head := 0; head < len(queue); head++ {
		index := queue[head]
		x, y := index%width, index/width
		if x > 0 {
			enqueue(x-1, y)
		}
		if x+1 < width {
			enqueue(x+1, y)
		}
		if y > 0 {
			enqueue(x, y-1)
		}
		if y+1 < height {
			enqueue(x, y+1)
		}
	}
	for i, v := range visible {
		if !exterior[i] {
			subject.Pix[i] = v
		}
	}
}

// contrastingCanvas chooses black or white for alpha cutouts and busy scenes,
// maximizing average channel contrast over visible pixels.
func contrastingCanvas(source image.Image, visible []uint8) color.NRGBA {
	bounds := source.Bounds()
	var onBlack, onWhite uint64
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			if visible[y*bounds.Dx()+x] == 0 {
				continue
			}
			c := sourceColor(source, bounds.Min.X+x, bounds.Min.Y+y)
			onBlack += uint64(max(c.R, c.G, c.B))
			onWhite += uint64(255 - min(c.R, c.G, c.B))
		}
	}
	if onBlack > onWhite {
		return color.NRGBA{A: 255}
	}
	return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
}

func contrastMap(source image.Image, background color.NRGBA) *image.Gray {
	bounds := source.Bounds()
	detail := image.NewGray(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			c := sourceColor(source, bounds.Min.X+x, bounds.Min.Y+y)
			detail.Pix[y*bounds.Dx()+x] = uint8(colorDistance(c, background) * int(c.A) / 255)
		}
	}
	return detail
}

func sourceColor(source image.Image, x, y int) color.NRGBA {
	return color.NRGBAModel.Convert(source.At(x, y)).(color.NRGBA)
}

func colorDistance(a, b color.NRGBA) int {
	return max(abs(int(a.R)-int(b.R)), abs(int(a.G)-int(b.G)), abs(int(a.B)-int(b.B)))
}
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
func hasSelectedPixels(pixels []uint8) bool {
	for _, v := range pixels {
		if v != 0 {
			return true
		}
	}
	return false
}

func visitBorder(width, height int, visit func(x, y int)) {
	for x := 0; x < width; x++ {
		visit(x, 0)
		if height > 1 {
			visit(x, height-1)
		}
	}
	for y := 1; y < height-1; y++ {
		visit(0, y)
		if width > 1 {
			visit(width-1, y)
		}
	}
}

// Some PNGs are opaque rectangular panels with transparent rounded corners,
// rather than subject cutouts. Those panels still need background removal.
func fillsVisibleRectangle(visible []uint8, width, height int) bool {
	bounds := visibleBounds(visible, width, height)
	count := 0
	for _, pixel := range visible {
		if pixel != 0 {
			count++
		}
	}
	return count > 0 && count*100 >= bounds.Dx()*bounds.Dy()*95
}

func visibleBounds(visible []uint8, width, height int) image.Rectangle {
	left, top, right, bottom := width, height, 0, 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if visible[y*width+x] == 0 {
				continue
			}
			left, top = min(left, x), min(top, y)
			right, bottom = max(right, x+1), max(bottom, y+1)
		}
	}
	if right <= left || bottom <= top {
		return image.Rectangle{}
	}
	return image.Rect(left, top, right, bottom)
}
