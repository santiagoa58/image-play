package wordcloud

import "testing"

func TestNewConfigUsesDefaultsAndAppliesOptions(t *testing.T) {
	got := NewConfig(
		WithInputPath("custom.png"),
		WithWordLimit(500),
		WithDebug(true),
	)

	if got.InputPath != "custom.png" {
		t.Errorf("InputPath = %q, want custom.png", got.InputPath)
	}
	if got.WordLimit != 500 {
		t.Errorf("WordLimit = %d, want 500", got.WordLimit)
	}
	if !got.Debug {
		t.Error("Debug = false, want true")
	}
	if got.MinFontSize != 0 {
		t.Errorf("MinFontSize = %v, want automatic default 0", got.MinFontSize)
	}
	if got.MaxFontSize != 0 {
		t.Errorf("MaxFontSize = %v, want default 0 (automatic)", got.MaxFontSize)
	}
}

func TestConfigValidate(t *testing.T) {
	cfg := NewConfig(
		WithInputPath("input.png"),
		WithTextPath("words.txt"),
		WithFontPath("font.ttf"),
	)

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestNewConfigAllowsZeroValueOverrides(t *testing.T) {
	got := NewConfig(WithWordLimit(0), WithDebug(false))

	if got.WordLimit != 0 {
		t.Errorf("WordLimit = %d, want 0", got.WordLimit)
	}
	if got.Debug {
		t.Error("Debug = true, want false")
	}
}

func TestConfigValidateRejectsNegativeWordPadding(t *testing.T) {
	cfg := NewConfig(
		WithInputPath("input.png"),
		WithTextPath("words.txt"),
		WithFontPath("font.ttf"),
	)
	cfg.WordPadding = -1

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want negative padding error")
	}
}
