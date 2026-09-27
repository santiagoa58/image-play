// Package wordcloud generates weighted word clouds constrained to an image
// silhouette.
//
// The package owns frequency-based sizing, shape-region priorities,
// orientation preference, font-size search, and rendering. It delegates exact
// rectangle geometry to internal/layout so visual policy and fit mechanics can
// evolve independently.
package wordcloud
