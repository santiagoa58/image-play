// Package textmosaic generates text mosaics from source images.
//
// The package owns the complete effect pipeline: loading input and text,
// preparing the source image, measuring the font, clipping it through repeated text,
// and writing the final PNG. The CLI only selects the effect and supplies paths.
package textmosaic
