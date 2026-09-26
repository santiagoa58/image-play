// Package textutil provides text ingestion, frequency counting, output-path
// helpers, and font measurement for text-based image effects.
//
// Word-cloud code intentionally keeps tokenization and font measurement here so
// placement can work with already measured Word values instead of parsing text
// or loading files itself.
package textutil
