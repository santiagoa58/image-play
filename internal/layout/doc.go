// Package layout provides exact rectangular placement geometry. FreeSpace
// tracks the remaining binary mask; rectangular erosion identifies every legal
// integer center for a footprint, and Reserve removes that footprint. The
// package does not know about words, orientation policy, or shape regions.
package layout
