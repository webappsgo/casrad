// Package theme - Tests for palette selection and CSS generation.
// Covers: Get(dark), Get(light), Get(unknown falls back to dark), CSS() output.
package theme

import (
	"strings"
	"testing"
)

func TestGetDarkTheme(t *testing.T) {
	t.Parallel()

	p := Get("dark")
	if p.Background != Dark.Background {
		t.Errorf("Get(dark) background = %q, want %q", p.Background, Dark.Background)
	}
}

func TestGetLightTheme(t *testing.T) {
	t.Parallel()

	p := Get("light")
	if p.Background != Light.Background {
		t.Errorf("Get(light) background = %q, want %q", p.Background, Light.Background)
	}
}

func TestGetUnknownDefaultsDark(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"", "auto", "custom", "dracula"} {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			p := Get(name)
			if p.Background != Dark.Background {
				t.Errorf("Get(%q) background = %q, want dark default %q", name, p.Background, Dark.Background)
			}
		})
	}
}

func TestPaletteCSSContainsVariables(t *testing.T) {
	t.Parallel()

	p := Dark
	css := p.CSS()

	required := []string{
		"--color-background",
		"--color-foreground",
		"--color-primary",
		"--color-secondary",
		"--color-accent",
		"--color-success",
		"--color-warning",
		"--color-error",
		"--color-info",
		"--color-surface",
		"--color-surface-alt",
		"--color-border",
		"--color-muted",
		":root",
	}
	for _, v := range required {
		if !strings.Contains(css, v) {
			t.Errorf("CSS() missing variable %q", v)
		}
	}
}

func TestPaletteCSSContainsActualColors(t *testing.T) {
	t.Parallel()

	p := Dark
	css := p.CSS()

	if !strings.Contains(css, p.Background) {
		t.Errorf("CSS() does not contain background color %q", p.Background)
	}
	if !strings.Contains(css, p.Primary) {
		t.Errorf("CSS() does not contain primary color %q", p.Primary)
	}
}

func TestDarkAndLightDiffer(t *testing.T) {
	t.Parallel()

	if Dark.Background == Light.Background {
		t.Error("Dark and Light themes have the same background — they must differ")
	}
	if Dark.Primary == Light.Primary {
		t.Error("Dark and Light themes have the same primary color — they must differ")
	}
}
