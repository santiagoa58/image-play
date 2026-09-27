package wordcloud

import (
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/santiagoa58/image-play/internal/imageutil"
)

// Exercise orchestration, font measurement, placement, color sampling, and PNG
// output for every fixture. Full candidate-pool visual checks use the CLI script.
func TestGenerateEveryFixtureWithRegularAndBoldUppercase(t *testing.T) {
	inputs, err := filepath.Glob("../../../testdata/images/*")
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) == 0 {
		t.Fatal("no image fixtures")
	}
	for _, input := range inputs {
		for _, uppercase := range []bool{false, true} {
			name := filepath.Base(input)
			if uppercase {
				name += "-bold-uppercase"
			}
			t.Run(name, func(t *testing.T) {
				font := wordcloudTestFontPath(t)
				if uppercase {
					font = "../../../fonts/NotoSans-Bold.ttf"
				}
				output := filepath.Join(t.TempDir(), "cloud.png")
				cfg := NewConfig(WithInputPath(input), WithTextPath("../../../testdata/text/darth_vader.txt"), WithFontPath(font), WithOutputPath(output), WithWordLimit(60), WithUppercase(uppercase))
				if err := Generate(cfg); err != nil {
					t.Fatal(err)
				}
				file, err := os.Open(output)
				if err != nil {
					t.Fatal(err)
				}
				rendered, err := png.Decode(file)
				file.Close()
				if err != nil {
					t.Fatal(err)
				}
				mask, err := imageutil.PrepareMask(input, 8)
				if err != nil {
					t.Fatal(err)
				}
				defer mask.Close()
				if rendered.Bounds().Dx() != mask.Width || rendered.Bounds().Dy() != mask.Height {
					t.Fatal("output size differs from source")
				}
				ink := 0
				for y := 0; y < mask.Height; y++ {
					for x := 0; x < mask.Width; x++ {
						c := color.NRGBAModel.Convert(rendered.At(x, y)).(color.NRGBA)
						distance := max(absChannel(int(c.R)-int(mask.Background.R)), absChannel(int(c.G)-int(mask.Background.G)), absChannel(int(c.B)-int(mask.Background.B)))
						if distance <= 12 {
							continue
						}
						ink++
						if mask.SubjectMat.GetUCharAt(y, x) == 0 {
							t.Fatalf("visible text outside subject at (%d,%d)", x, y)
						}
					}
				}
				if ink == 0 {
					t.Fatal("output has no visible text")
				}
			})
		}
	}
}
