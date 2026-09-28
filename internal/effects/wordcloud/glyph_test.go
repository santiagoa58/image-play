package wordcloud

import "testing"

func TestFillGlyphHolesKeepsOnlyExteriorGaps(t *testing.T) {
	pixels := []uint8{
		0, 0, 0, 0, 0,
		0, 255, 255, 255, 0,
		0, 255, 0, 255, 0,
		0, 255, 255, 255, 0,
		0, 0, 0, 0, 0,
	}
	fillGlyphHoles(pixels, 5, 5)
	if pixels[2*5+2] != 255 {
		t.Fatal("enclosed letter hole should be reserved")
	}
	if pixels[0] != 0 {
		t.Fatal("space outside the glyph should stay available")
	}
}
