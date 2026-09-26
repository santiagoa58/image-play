// Package wordcloud generates weighted word clouds constrained to an image
// silhouette.
//
// The package owns the artistic placement policy: frequency-based sizing,
// distance-transform search centers, horizontal-first orientation, spiral
// candidate order, font-size fallback, and final rendering. It delegates
// generic rectangle geometry to internal/layout so mask and collision mechanics
// can evolve independently of the visual behavior.
package wordcloud
