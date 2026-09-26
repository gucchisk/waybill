package tui

import (
	"slices"
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/gucchisk/waybill/internal/adapter/jsonview"
	"github.com/gucchisk/waybill/internal/domain"
)

// Colors: use only the ANSI palette colors 0-15 so that the TUI follows the terminal's color scheme
// (e.g. the iTerm2 profile). Colors 16-255 and RGB colors are fixed and do not follow it.
//
// tcell's named colors come from HTML/CSS, so they differ from the ANSI names.
// For example, tcell.ColorRed is Bright Red (9), not Red (1). There are no constants such as tcell.Color8.
//
//	 0 Black          tcell.ColorBlack     8 Bright Black    tcell.ColorGray
//	 1 Red            tcell.ColorMaroon    9 Bright Red      tcell.ColorRed
//	 2 Green          tcell.ColorGreen    10 Bright Green    tcell.ColorLime
//	 3 Yellow         tcell.ColorOlive    11 Bright Yellow   tcell.ColorYellow
//	 4 Blue           tcell.ColorNavy     12 Bright Blue     tcell.ColorBlue
//	 5 Magenta        tcell.ColorPurple   13 Bright Magenta  tcell.ColorFuchsia
//	 6 Cyan           tcell.ColorTeal     14 Bright Cyan     tcell.ColorAqua
//	 7 White          tcell.ColorSilver   15 Bright White    tcell.ColorWhite

// cursorLineStyleFor returns the style of the whole cursor line for the terminal's background brightness.
// On a dark background it is reverse video, which swaps the terminal's default foreground and background.
// On a light background reverse video turns the line pure black and hard to read, so it is White (15) text on Bright Black (8) instead.
func cursorLineStyleFor(hasDarkBackground bool) tcell.Style {
	if hasDarkBackground {
		return tcell.StyleDefault.Reverse(true)
	}
	return tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorGray)
}

func selectedBackgroundColorFor(hasDarkBackground bool) tcell.Color {
	if hasDarkBackground {
		return tcell.ColorGray
	}
	return tcell.ColorSilver
}

var spanStyles = map[jsonview.SpanKind]tcell.Style{
	jsonview.SpanPunctuation: tcell.StyleDefault,
	jsonview.SpanKey:         tcell.StyleDefault.Foreground(tcell.ColorTeal),
	jsonview.SpanString:      tcell.StyleDefault.Foreground(tcell.ColorGreen),
	jsonview.SpanNumber:      tcell.StyleDefault.Foreground(tcell.ColorOlive),
	jsonview.SpanLiteral:     tcell.StyleDefault.Foreground(tcell.ColorPurple),
}

func (app *App) bodyHeight() int {
	_, height := app.screen.Size()
	return max(height-2, 1)
}

func (app *App) draw() {
	width, height := app.screen.Size()
	app.screen.Clear()

	view := app.currentView()
	app.keepCursorVisible(view)

	headerStyle := tcell.StyleDefault.Bold(true)
	app.drawText(0, 0, view.title, headerStyle, width)
	app.drawBody(view, width)
	app.drawFooter(height-1, width)
	app.screen.Show()
}

func (app *App) keepCursorVisible(view *contentView) {
	bodyHeight := app.bodyHeight()
	if view.cursorLine < view.topLine {
		view.topLine = view.cursorLine
	}
	if view.cursorLine >= view.topLine+bodyHeight {
		view.topLine = view.cursorLine - bodyHeight + 1
	}
}

func (app *App) drawBody(view *contentView, width int) {
	selectedObject, hasSelectedObject := view.document.SelectableObjectAt(view.cursorDocumentLine())

	for row := 0; row < app.bodyHeight(); row++ {
		displayLineNumber := view.topLine + row
		if displayLineNumber >= len(view.displayLines) {
			return
		}
		displayLine := view.displayLines[displayLineNumber]
		screenRow := row + 1
		isCursorLine := displayLineNumber == view.cursorLine
		isSelected := hasSelectedObject && displayLine.DocumentLine >= selectedObject.StartLine && displayLine.DocumentLine <= selectedObject.EndLine

		rowStyle := tcell.StyleDefault
		switch {
		case isCursorLine:
			rowStyle = app.cursorLineStyle
			app.fillRow(screenRow, width, rowStyle)
		case isSelected:
			rowStyle = rowStyle.Background(app.selectedBackgroundColor)
			app.fillRow(screenRow, width, rowStyle)
		}

		x := 0
		for _, span := range displayLine.Spans {
			style := rowStyle
			if !isCursorLine {
				style = spanStyles[span.Kind]
				if isSelected {
					style = style.Background(app.selectedBackgroundColor)
				}
			}
			x = app.drawText(x, screenRow, span.Text, style, width)
		}
	}
}

func (app *App) drawFooter(row, width int) {
	if app.statusMessage != "" {
		app.drawText(0, row, app.statusMessage, tcell.StyleDefault.Bold(true), width)
		return
	}
	app.drawText(0, row, app.helpText(), tcell.StyleDefault.Dim(true), width)
}

// helpText lists only the keys that are usable right now.
// The action keys are shown only when the cursor is inside an object whose mediaType allows the action.
func (app *App) helpText() string {
	_, actions := app.availableActions()
	keyHelps := []string{"↑↓/C-p C-n:move", "PgUp PgDn/M-v C-v:page"}
	if objectIndex, isFolded := app.currentView().foldTarget(); objectIndex >= 0 {
		if isFolded {
			keyHelps = append(keyHelps, "Space:unfold")
		} else {
			keyHelps = append(keyHelps, "Space:fold")
		}
	}
	if slices.Contains(actions, domain.ActionView) {
		keyHelps = append(keyHelps, "Enter:view JSON")
	}
	if slices.Contains(actions, domain.ActionDownload) {
		keyHelps = append(keyHelps, "d:download")
	}
	if len(app.views) == 1 {
		keyHelps = append(keyHelps, "Esc/q:quit")
	} else {
		keyHelps = append(keyHelps, "Esc/q:back")
	}
	keyHelps = append(keyHelps, "C-c:quit")
	return strings.Join(keyHelps, "  ")
}

func (app *App) fillRow(row, width int, style tcell.Style) {
	for x := 0; x < width; x++ {
		app.screen.Put(x, row, " ", style)
	}
}

func (app *App) drawText(x, y int, text string, style tcell.Style, limitX int) int {
	for text != "" && x < limitX {
		remaining, cellWidth := app.screen.Put(x, y, text, style)
		if remaining == text {
			break
		}
		text = remaining
		x += cellWidth
	}
	return x
}
