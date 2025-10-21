package core

import (
	"encoding/json"
	"fmt"

	"github.com/casapps/casrad/internal/database"
)

type ThemeColors struct {
	Background    string `json:"background"`
	CurrentLine   string `json:"current_line"`
	Foreground    string `json:"foreground"`
	Comment       string `json:"comment"`
	Cyan          string `json:"cyan"`
	Green         string `json:"green"`
	Orange        string `json:"orange"`
	Pink          string `json:"pink"`
	Purple        string `json:"purple"`
	Red           string `json:"red"`
	Yellow        string `json:"yellow"`
	Selection     string `json:"selection"`
	Border        string `json:"border"`
	Shadow        string `json:"shadow"`
	Hover         string `json:"hover"`
	Active        string `json:"active"`
	Success       string `json:"success"`
	Warning       string `json:"warning"`
	Error         string `json:"error"`
	Info          string `json:"info"`
}

type Theme struct {
	Name         string      `json:"name"`
	DisplayName  string      `json:"display_name"`
	Description  string      `json:"description"`
	IsDefault    bool        `json:"is_default"`
	Colors       ThemeColors `json:"colors"`
	FontFamily   string      `json:"font_family"`
	FontMono     string      `json:"font_mono"`
	FontSize     string      `json:"font_size"`
	BorderRadius string      `json:"border_radius"`
	Transition   string      `json:"transition"`
}

type ThemeManager struct {
	db           *database.Engine
	themes       map[string]*Theme
	defaultTheme string
}

func NewThemeManager(db *database.Engine) *ThemeManager {
	tm := &ThemeManager{
		db:           db,
		themes:       make(map[string]*Theme),
		defaultTheme: "dark",
	}

	// Initialize default themes
	tm.initDefaultThemes()

	return tm
}

func (tm *ThemeManager) initDefaultThemes() {
	// Dracula Dark Theme (Default)
	tm.themes["dark"] = &Theme{
		Name:        "dark",
		DisplayName: "Dracula Dark",
		Description: "Beautiful dark theme based on Dracula",
		IsDefault:   true,
		Colors: ThemeColors{
			Background:  "#282a36",
			CurrentLine: "#44475a",
			Foreground:  "#f8f8f2",
			Comment:     "#6272a4",
			Cyan:        "#8be9fd",
			Green:       "#50fa7b",
			Orange:      "#ffb86c",
			Pink:        "#ff79c6",
			Purple:      "#bd93f9",
			Red:         "#ff5555",
			Yellow:      "#f1fa8c",
			Selection:   "#44475a",
			Border:      "#6272a4",
			Shadow:      "rgba(0,0,0,0.3)",
			Hover:       "#50fa7b",
			Active:      "#bd93f9",
			Success:     "#50fa7b",
			Warning:     "#f1fa8c",
			Error:       "#ff5555",
			Info:        "#8be9fd",
		},
		FontFamily:   "Inter, system-ui, sans-serif",
		FontMono:     "JetBrains Mono, monospace",
		FontSize:     "16px",
		BorderRadius: "6px",
		Transition:   "200ms",
	}

	// Clean Light Theme
	tm.themes["light"] = &Theme{
		Name:        "light",
		DisplayName: "Clean Light",
		Description: "Clean and modern light theme",
		IsDefault:   false,
		Colors: ThemeColors{
			Background:  "#ffffff",
			CurrentLine: "#f5f5f5",
			Foreground:  "#2e3440",
			Comment:     "#6c757d",
			Cyan:        "#0969da",
			Green:       "#1a7f37",
			Orange:      "#fb8500",
			Pink:        "#bf3989",
			Purple:      "#8250df",
			Red:         "#cf222e",
			Yellow:      "#d4a72c",
			Selection:   "#e1e4e8",
			Border:      "#d0d7de",
			Shadow:      "rgba(0,0,0,0.1)",
			Hover:       "#0969da",
			Active:      "#8250df",
			Success:     "#1a7f37",
			Warning:     "#d4a72c",
			Error:       "#cf222e",
			Info:        "#0969da",
		},
		FontFamily:   "Inter, system-ui, sans-serif",
		FontMono:     "JetBrains Mono, monospace",
		FontSize:     "16px",
		BorderRadius: "6px",
		Transition:   "200ms",
	}
}

func (tm *ThemeManager) GetTheme(name string) (*Theme, error) {
	if theme, ok := tm.themes[name]; ok {
		return theme, nil
	}
	return nil, fmt.Errorf("theme not found: %s", name)
}

func (tm *ThemeManager) GetDefaultTheme() *Theme {
	return tm.themes[tm.defaultTheme]
}

func (tm *ThemeManager) GetCSS(themeName string) string {
	theme, err := tm.GetTheme(themeName)
	if err != nil {
		theme = tm.GetDefaultTheme()
	}

	return fmt.Sprintf(`
:root {
	--color-background: %s;
	--color-current-line: %s;
	--color-foreground: %s;
	--color-comment: %s;
	--color-cyan: %s;
	--color-green: %s;
	--color-orange: %s;
	--color-pink: %s;
	--color-purple: %s;
	--color-red: %s;
	--color-yellow: %s;
	--color-selection: %s;
	--color-border: %s;
	--color-shadow: %s;
	--color-hover: %s;
	--color-active: %s;
	--color-success: %s;
	--color-warning: %s;
	--color-error: %s;
	--color-info: %s;
	--font-primary: %s;
	--font-mono: %s;
	--font-size-base: %s;
	--border-radius: %s;
	--transition-speed: %s;
}`,
		theme.Colors.Background,
		theme.Colors.CurrentLine,
		theme.Colors.Foreground,
		theme.Colors.Comment,
		theme.Colors.Cyan,
		theme.Colors.Green,
		theme.Colors.Orange,
		theme.Colors.Pink,
		theme.Colors.Purple,
		theme.Colors.Red,
		theme.Colors.Yellow,
		theme.Colors.Selection,
		theme.Colors.Border,
		theme.Colors.Shadow,
		theme.Colors.Hover,
		theme.Colors.Active,
		theme.Colors.Success,
		theme.Colors.Warning,
		theme.Colors.Error,
		theme.Colors.Info,
		theme.FontFamily,
		theme.FontMono,
		theme.FontSize,
		theme.BorderRadius,
		theme.Transition,
	)
}

func (tm *ThemeManager) ToJSON(themeName string) ([]byte, error) {
	theme, err := tm.GetTheme(themeName)
	if err != nil {
		return nil, err
	}
	return json.Marshal(theme)
}