package main

import (
	"strings"
	"testing"
)

func TestParseEffect(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    effect
		wantErr string
	}{
		{name: "wordcloud", input: "wordcloud", want: effectWordCloud},
		{name: "textmosaic", input: "textmosaic", want: effectTextMosaic},
		{name: "case insensitive", input: " WORDCLOUD ", want: effectWordCloud},
		{name: "missing", input: "", wantErr: "missing required flag"},
		{name: "unknown", input: "posterize", wantErr: "unsupported effect"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEffect(tt.input)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("parseEffect(%q) error = %v, want substring %q", tt.input, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseEffect(%q) error = %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("parseEffect(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateRequiredFlags(t *testing.T) {
	valid := options{
		inputPath: "input.png",
		textPath:  "words.txt",
		fontPath:  "font.ttf",
	}
	if err := validateRequiredFlags(valid); err != nil {
		t.Fatalf("validateRequiredFlags(valid) error = %v", err)
	}

	for _, tt := range []struct {
		name string
		edit func(*options)
		flag string
	}{
		{name: "input", edit: func(o *options) { o.inputPath = "" }, flag: "-in"},
		{name: "text", edit: func(o *options) { o.textPath = "" }, flag: "-text"},
		{name: "font", edit: func(o *options) { o.fontPath = "" }, flag: "-font"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			opts := valid
			tt.edit(&opts)
			err := validateRequiredFlags(opts)
			if err == nil || !strings.Contains(err.Error(), tt.flag) {
				t.Fatalf("validateRequiredFlags() error = %v, want mention of %s", err, tt.flag)
			}
		})
	}
}
