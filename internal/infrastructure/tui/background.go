package tui

import (
	"fmt"

	"github.com/muesli/termenv"
)

type Theme int

const (
	ThemeAuto Theme = iota
	ThemeDark
	ThemeLight
)

func (theme Theme) String() string {
	switch theme {
	case ThemeDark:
		return "dark"
	case ThemeLight:
		return "light"
	default:
		return "auto"
	}
}

func (theme *Theme) Set(value string) error {
	switch value {
	case "auto":
		*theme = ThemeAuto
	case "dark":
		*theme = ThemeDark
	case "light":
		*theme = ThemeLight
	default:
		return fmt.Errorf("must be one of auto, dark, light")
	}
	return nil
}

func (theme Theme) Type() string {
	return "theme"
}

func (theme Theme) hasDarkBackground() bool {
	if theme == ThemeLight {
		return false
	}
	if theme == ThemeDark {
		return true
	}
	return detectDarkBackground()
}

func detectDarkBackground() bool {
	return termenv.HasDarkBackground()
}
