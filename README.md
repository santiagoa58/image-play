# image-play

`image-play` is a Go image-effects playground with multiple supported effects,
including image-shaped word clouds and text mosaics.

The CLI selects an effect explicitly with `-effect`:

```text
-effect wordcloud
-effect textmosaic
```

Both effects share the same required inputs: an image, a text file, and a font.

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

Current defaults include horizontal-first 0/90-degree placement, logarithmic
frequency scaling, rectangular word footprints, an automatic maximum font size,
and up to 500 candidate words. The candidate limit is a source pool: words that
cannot fit at the minimum font size are skipped.

### Pipeline

```text
source image
    ↓
binary silhouette + distance transform
    ↓
word counts + measured target font sizes
    ↓
distance-based placement centers
    ↓
spiral candidate positions
    ↓
layout.Space.TryPlace(rect)
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
- `internal/effects/wordcloud`: artistic layout policy: center selection,
  horizontal/vertical preference, spiral search, resizing, progress, and
  rendering.
- `internal/effects/textmosaic`: text-mosaic effect that reconstructs an image
  from repeated text colored from the source image.
- `internal/layout`: generic rectangle containment and collision geometry.
- `internal/mathutil`: small deterministic geometry and scaling helpers.

## Placement geometry

The low-level geometry is deliberately isolated in `internal/layout`. The
word-cloud package decides *where* and *how* to try a word; `layout.Space`
only decides whether that rectangular footprint is valid and reserves it on
success.

Two mature word-cloud implementations informed this design:

1. **amueller/word_cloud** (Python, MIT license) uses an integral/summed-area
   occupancy image to make rectangle-space queries cheap.
   - https://github.com/amueller/word_cloud
   - https://github.com/amueller/word_cloud/blob/master/wordcloud/wordcloud.py
   - https://github.com/amueller/word_cloud/blob/master/wordcloud/query_integral_image.pyx

2. **psykhi/wordclouds** (Go, Apache-2.0 license) uses a spatial hash so
   collision tests only inspect nearby placed rectangles.
   - https://github.com/psykhi/wordclouds
   - https://github.com/psykhi/wordclouds/blob/master/spatialhashmap.go

`image-play` combines those ideas rather than importing either complete
layout engine. The static silhouette uses a summed-area mask for constant-time
rectangle containment checks; dynamic occupied rectangles use a spatial index
for local collision checks. The implementation is adapted to Go's
`image.Rectangle`, our alpha-aware mask semantics, and our existing artistic
placement policy.

This attribution documents the algorithmic references used while designing the
implementation. The source files in `internal/layout` contain the same
references next to the code.

## Placement behavior

Words are processed in frequency order. The minimum font size is configured,
while the default maximum is calibrated against the current image by probing
the layout with the most important words. Their final target sizes are then
logarithmically mapped into that resolved font-size range. If a word cannot
fit, placement retries it at progressively smaller sizes down to the configured
minimum.

The maximum starting size for each new word is capped by the previous
successfully placed word's actual size. This keeps rendered sizes
non-increasing even when an important word had to shrink to fit.

When words have equal frequency, their measured rectangle area breaks the tie:
larger, harder-to-fit words are attempted before smaller gap-filling words.

For each size, placement searches every configured position horizontally before
falling back to 90-degree rotation. Search origins come from deep points in the
distance transform so large words start in roomy parts of the silhouette.

## Text mosaic

The text-mosaic effect rebuilds an image from repeated text, sampling the source
image for the text color at each grid position.

```bash
go run ./cmd/mosaic \
  -effect textmosaic \
  -in testdata/images/gen-img-couple.png \
  -text testdata/text/sample_text_message.txt \
  -font "fonts/NotoSansMono-VariableFont_wdth,wght.ttf" \
  -out output.png
```

The implementation supports resizing, contrast adjustment, transparent source
pixels, and automatic font scaling across image resolutions.

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
- placement centers,
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
