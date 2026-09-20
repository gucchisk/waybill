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
	commandConfirm
	commandBack
	commandQuit
)

// commandFor はキー入力を操作へ変換する。矢印キーとEmacsバインドの両方に対応する。
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
		return commandConfirm
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
	if event.Rune() == 'q' {
		return commandBack
	}
	return commandNone
}
