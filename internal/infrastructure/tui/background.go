package tui

import (
	"fmt"
	"time"

	"github.com/muesli/termenv"
)

const backgroundDetectionTimeout = 500 * time.Millisecond

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
	return setTimeout(termenv.HasDarkBackground, backgroundDetectionTimeout, true)
}

func setTimeout[T any](fn func() T, timeout time.Duration, fallback T) T {
	result := make(chan T, 1)
	go func() {
		result <- fn()
	}()

	select {
	case value := <-result:
		return value
	case <-time.After(timeout):
		return fallback
	}
}
