package tui

import "github.com/gdamore/tcell/v2"

type command int

const (
	commandNone command = iota
	commandUp
	commandDown
	commandPageUp
	commandPageDown
	commandTop
	commandBottom
	commandView
	commandDownload
	commandBack
	commandQuit
)

// commandFor converts a key input into a command. Both arrow keys and Emacs bindings are supported.
func commandFor(event *tcell.EventKey) command {
	switch event.Key() {
	case tcell.KeyUp, tcell.KeyCtrlP:
		return commandUp
	case tcell.KeyDown, tcell.KeyCtrlN:
		return commandDown
	case tcell.KeyPgUp:
		return commandPageUp
	case tcell.KeyPgDn, tcell.KeyCtrlV:
		return commandPageDown
	case tcell.KeyHome:
		return commandTop
	case tcell.KeyEnd:
		return commandBottom
	case tcell.KeyEnter:
		return commandView
	case tcell.KeyEscape, tcell.KeyCtrlG:
		return commandBack
	case tcell.KeyCtrlC:
		return commandQuit
	case tcell.KeyRune:
		return commandForRune(event)
	}
	return commandNone
}

func commandForRune(event *tcell.EventKey) command {
	if event.Modifiers()&tcell.ModAlt != 0 {
		switch event.Rune() {
		case 'v':
			return commandPageUp
		case '<':
			return commandTop
		case '>':
			return commandBottom
		}
		return commandNone
	}
	switch event.Rune() {
	case 'q':
		return commandBack
	case 'd':
		return commandDownload
	}
	return commandNone
}
