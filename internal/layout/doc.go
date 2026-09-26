// Package layout provides the low-level rectangle geometry used by effects.
//
// The package deliberately contains no word-cloud policy. Callers decide which
// rectangle to try, in what order, and at what size; Space only answers whether
// a rectangle is inside the allowed mask and collision-free, then reserves it.
//
// Two mature word-cloud implementations informed this design:
//
//   - Andreas Mueller's Python word_cloud uses a summed-area (integral) image
//     for fast occupancy queries:
//     https://github.com/amueller/word_cloud
//     https://github.com/amueller/word_cloud/blob/master/wordcloud/wordcloud.py
//
//   - psykhi/wordclouds for Go uses a spatial hash so collision checks only
//     inspect nearby placed words:
//     https://github.com/psykhi/wordclouds
//     https://github.com/psykhi/wordclouds/blob/master/spatialhashmap.go
//
// This package combines those ideas for image-play's existing placement policy:
// the static silhouette is represented by an integral mask, while dynamic
// placed rectangles are indexed spatially. The implementation is adapted to
// Go's image.Rectangle and image-play's mask semantics rather than copied as a
// drop-in version of either project.
//
// The referenced Python project is MIT-licensed; psykhi/wordclouds is
// Apache-2.0-licensed. See the repository documentation for attribution details.
package layout
