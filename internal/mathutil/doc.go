// Package mathutil contains the small geometric and numeric helpers used by
// effects that need deterministic placement math.
//
// The package keeps collision predicates strict: contact at a single point or
// edge is not considered overlap. That matches the word-placement use case,
// where touching glyph bounding regions should still leave a small visual gap
// after caller-provided padding is applied.
package mathutil
