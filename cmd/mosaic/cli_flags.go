package main

import (
	"flag"
	"fmt"
)

type commonOptions struct {
	inputPath  string
	outputPath string
	effect     string
	width      int
	fontPath   string
	text       string
	textFile   string
	overwrite  bool
	createDirs bool
	verbose    bool
	advanced   bool
}

type cliOptions struct {
	common    commonOptions
	text      textMosaicOptions
	wordCloud wordCloudOptions
}

func flagCommandLine() *flag.FlagSet {
	return flag.CommandLine
}

func registerCLIFlags(fs *flag.FlagSet) *cliOptions {
	opts := &cliOptions{}
	registerCommonFlags(fs, &opts.common)
	registerTextMosaicFlags(fs, &opts.text)
	registerWordCloudFlags(fs, &opts.wordCloud)
	return opts
}

func registerCommonFlags(fs *flag.FlagSet, opts *commonOptions) {
	fs.StringVar(&opts.inputPath, "in", "", "Path to input image (PNG, JPEG, WebP, etc.) [required]")
	fs.StringVar(&opts.outputPath, "out", "", "Output path. Can be file or directory. Empty = input_<effect>.png")
	fs.StringVar(&opts.effect, "effect", effectTextMosaic, `Effect to generate: "textmosaic" or "wordcloud"`)
	fs.IntVar(&opts.width, "width", 0, "Target width in pixels. 0 = effect default")
	fs.StringVar(&opts.fontPath, "font", "", "Path to monospace TTF/OTF font. Empty = bundled font when available")
	fs.StringVar(&opts.text, "text", "", "Text to use for the selected effect")
	fs.StringVar(&opts.textFile, "text-file", "", "Path to UTF-8 text file to use as effect text")
	fs.BoolVar(&opts.overwrite, "overwrite", true, "Allow overwriting an existing output file")
	fs.BoolVar(&opts.createDirs, "create-dirs", true, "Create missing output directories")
	fs.BoolVar(&opts.verbose, "v", false, "Enable verbose/debug logging")
	fs.BoolVar(&opts.advanced, "help-advanced", false, "Show all tuning flags")
}

func configureUsage(fs *flag.FlagSet) {
	fs.Usage = func() {
		out := fs.Output()
		fmt.Fprintf(out, "%s - image effects toolkit\n\n", appName)
		fmt.Fprint(out, "Simple usage:\n")
		fmt.Fprintf(out, "  %s -effect textmosaic -in photo.jpg -out text.png\n", appName)
		fmt.Fprintf(out, "  %s -effect wordcloud -in photo.jpg -out cloud.png\n\n", appName)
		fmt.Fprint(out, "Common flags:\n")
		fmt.Fprint(out, "  -effect        textmosaic or wordcloud\n")
		fmt.Fprint(out, "  -in            input image path\n")
		fmt.Fprint(out, "  -out           output image path\n")
		fmt.Fprint(out, "  -width         output width in pixels\n")
		fmt.Fprint(out, "  -text          inline text for the effect\n")
		fmt.Fprint(out, "  -text-file     UTF-8 text file for the effect\n")
		fmt.Fprint(out, "  -font          optional font path; defaults to bundled font\n")
		fmt.Fprint(out, "  -quality       wordcloud quality: fast, balanced, dense, poster\n")
		fmt.Fprint(out, "  -work-width    wordcloud internal packing width for speed/sharpness\n")
		fmt.Fprint(out, "  -bw            textmosaic black-and-white mode\n")
		fmt.Fprint(out, "  -text-weight   textmosaic weight from 1..4\n\n")
		fmt.Fprint(out, "Run with -help-advanced for every tuning flag.\n")
	}
}

func printAdvancedUsage(fs *flag.FlagSet) {
	out := fs.Output()
	fmt.Fprintf(out, "%s advanced flags\n\n", appName)
	fs.PrintDefaults()
}
