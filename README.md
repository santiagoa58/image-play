# image-play

`image-play` is a Go image-effects playground with multiple supported effects,
including image-shaped word clouds and text mosaics.

The CLI selects an effect explicitly with `-effect`:

```text
-effect wordcloud
-effect textmosaic
```

Both effects share the same required inputs: an image, a text file, and a font.

## Example effects

The same source image and text can be rendered with either supported effect.
These previews were generated from the current implementation using
`testdata/images/gen-img-couple.png`, the sample text, and the included
Noto Sans Mono font.

<table>
  <tr>
    <th>Original</th>
    <th>Text Mosaic</th>
    <th>Word Cloud</th>
  </tr>
  <tr>
    <td><img src="testdata/images/gen-img-couple.png" width="280" alt="Original couple image"></td>
    <td><img src="docs/assets/examples/gen-img-couple-textmosaic.webp" width="280" alt="Text mosaic effect generated from the couple image"></td>
    <td><img src="docs/assets/examples/gen-img-couple-wordcloud.webp" width="280" alt="Word cloud effect generated from the couple image"></td>
  </tr>
</table>

The README previews are resized for display; the effects themselves render at
the configured/source resolution.

## Word cloud

The word-cloud pipeline turns a source image into a placement silhouette, sizes
words by frequency, then packs those words inside the silhouette.

```bash
go run ./cmd/mosaic \
  -effect wordcloud \
  -in testdata/images/deepseek-logo-icon.png \
  -text testdata/text/sample_text_message.txt \
  -font "fonts/NotoSansMono-VariableFont_wdth,wght.ttf" \
  -out output.png
```

Current defaults include 0/90-degree placement, logarithmic frequency scaling,
rectangular word footprints, an automatic maximum font size, and up to 500
candidate words. The default minimum size is 6 px for every image. The
candidate limit is a source pool: words that cannot fit at the minimum size in
either orientation are skipped.

### Pipeline

```text
source image
    ↓
binary silhouette + distance transform
    ↓
word counts + measured target font sizes
    ↓
current free-space mask
    ↓
exact legal centers for each word footprint
    ↓
shape-region and orientation choice
    ↓
reserve the padded footprint
    ↓
accepted PlacedWord values
    ↓
PNG renderer
```

The main package boundaries are:

- `internal/imageutil`: image loading, alpha-aware segmentation, morphology,
  and distance transforms.
- `internal/textutil`: tokenization, stop-word filtering, frequency counting,
  font-size scaling, and word measurement.
- `internal/effects/wordcloud`: the complete word-cloud effect pipeline and
  artistic placement policy.
- `internal/effects/textmosaic`: the complete text-mosaic effect pipeline,
  including source preparation, font-grid measurement, rendering, and output.
- `internal/layout`: generic exact rectangular placement geometry.
- `internal/mathutil`: small deterministic geometry and scaling helpers.
- `cmd/mosaic`: CLI parsing and effect selection only.

## Placement geometry

The low-level geometry is isolated in `internal/layout`. `FreeSpace` keeps a
binary map of the silhouette minus the rectangles already placed. OpenCV's
rectangular erosion returns every integer center at which a measured, padded
rectangle fits wholly within that map. The same rectangle is removed when the
word is placed. Tests compare the result pixel for pixel against exhaustive
rectangle checks on small irregular shapes. The exactness claim applies to
these padded rectangles, not individual glyph outlines.

The separate region policy draws on [ShapeWordle's](https://www.microsoft.com/en-us/research/publication/shapewordle-tailoring-wordles-using-shape-aware-archimedean-spirals/)
use of distance and shape parts. Deep, separated points seed regions, and a
flood through the silhouette assigns every usable pixel to a region. The
region's occupied fraction ranks legal positions; it never rules out a fit.

## Placement behavior

Words are processed in frequency order. The default minimum is 6 px for every
image; a positive `MinFontSize` overrides it. The default maximum is calibrated
against the image by probing the most important words in a fresh layout; a
positive `MaxFontSize` overrides that calibration. Frequency
maps words logarithmically into this range. When a desired size does not fit,
placement binary-searches whole-pixel font sizes down to the minimum.

The maximum starting size for each new word is capped by the previous
successfully placed word's actual size. This keeps rendered sizes
non-increasing even when an important word had to shrink to fit.

When words have equal frequency, their measured rectangle area breaks the tie:
larger, harder-to-fit words are attempted before smaller gap-filling words.

Fit checks consider both configured orientations across the complete remaining
free space. At the chosen size, placement favors a less-filled shape region,
then the configured orientation order (horizontal first by default), then the
deepest legal center in that region. Each reserved rectangle contributes its
actual occupied area to every region it crosses. A skipped word has no legal
position in either orientation at the permitted minimum size.

## Text mosaic

The text-mosaic effect rebuilds an image from repeated text. The source is
optionally resized and contrast-adjusted, converted to grayscale, then sampled
at each text-grid position so the rendered characters reproduce the source
luminance on a transparent canvas.

```bash
go run ./cmd/mosaic \
  -effect textmosaic \
  -in testdata/images/gen-img-couple.png \
  -text testdata/text/sample_text_message.txt \
  -font "fonts/NotoSansMono-VariableFont_wdth,wght.ttf" \
  -out output.png
```

Text Mosaic owns its file loading, image preparation, font-grid measurement,
rendering, and output. The CLI dispatches to it in the same way it dispatches
to Word Cloud.

## Output paths

Both effects write PNG files. If `-out` is omitted, output is written beside
the input as:

```text
<input>_wordcloud.png
<input>_textmosaic.png
```

Existing directories and paths ending in a separator are treated as output
directories.

## Debugging

Set `Debug` in the word-cloud configuration to write intermediate images for:

- the binary mask,
- the distance transform,
- shape regions,
- the eroded safe zone,
- occupied rectangles,
- and the final rendered cloud.

Normal word-cloud generation logs the prepared, placed, skipped, horizontal,
and vertical word counts plus placement and total latency.

## Development

Requirements:

- Go 1.26.3+
- OpenCV development/runtime libraries for GoCV
- a TTF or OTF font

Run the test suite:

```bash
go test ./...
```

Build the CLI:

```bash
go build -o ./bin/mosaic ./cmd/mosaic
```

CI checks formatting, module tidiness, `go vet`, the full test suite, the
deployable Docker image, and all combinations of supported effects and images
under `testdata/images`. The generated PNGs are uploaded as a short-lived
workflow artifact so visual output can be reviewed.

The repository Dockerfile provides the OpenCV toolchain and runtime stages used
for local and container builds.

## License

MIT License. See [LICENSE](LICENSE).

Copyright (c) 2026 Santiago Gomez
