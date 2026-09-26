// Package imageutil contains image representations and OpenCV helpers shared by
// visual effects.
//
// The word-cloud pipeline uses it to derive a binary placement silhouette,
// preserve alpha semantics, and compute a distance field that describes how far
// each interior pixel is from the silhouette boundary.
package imageutil
