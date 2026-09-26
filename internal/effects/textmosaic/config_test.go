package textmosaic

import (
	"strings"
	"testing"
)

func TestNewConfigDefaults(t *testing.T) {
	cfg := NewConfig()

	if cfg.BaseFontSize != defaultBaseFontSize {
		t.Fatalf("BaseFontSize = %v, want %v", cfg.BaseFontSize, defaultBaseFontSize)
	}
	if cfg.TargetWidth != 0 {
		t.Fatalf("TargetWidth = %d, want 0", cfg.TargetWidth)
	}
	if cfg.ContrastPercent != 0 {
		t.Fatalf("ContrastPercent = %v, want 0", cfg.ContrastPercent)
	}
}

func TestConfigValidate(t *testing.T) {
	valid := NewConfig(
		WithInputPath("input.png"),
		WithTextPath("text.txt"),
		WithFontPath("font.ttf"),
	)

	tests := []struct {
		name    string
		edit    func(*Config)
		wantErr string
	}{
		{name: "valid", edit: func(*Config) {}},
		{name: "missing input", edit: func(c *Config) { c.InputPath = "" }, wantErr: "input path"},
		{name: "missing text", edit: func(c *Config) { c.TextPath = "" }, wantErr: "text path"},
		{name: "missing font", edit: func(c *Config) { c.FontPath = "" }, wantErr: "font path"},
		{name: "negative width", edit: func(c *Config) { c.TargetWidth = -1 }, wantErr: "target width"},
		{name: "invalid font size", edit: func(c *Config) { c.BaseFontSize = 0 }, wantErr: "base font size"},
		{name: "low contrast", edit: func(c *Config) { c.ContrastPercent = -101 }, wantErr: "contrast percent"},
		{name: "high contrast", edit: func(c *Config) { c.ContrastPercent = 101 }, wantErr: "contrast percent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := valid
			tt.edit(&cfg)

			err := cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}
