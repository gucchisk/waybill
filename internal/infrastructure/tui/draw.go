package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/uniseg"

	"github.com/gucchisk/waybill/internal/adapter/jsonview"
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

// selectedBackgroundColor is the subtle background of the selected object (Bright Black).
// With some color schemes (e.g. Solarized) Bright Black blends into the background; use tcell.ColorBlack then.
const selectedBackgroundColor = tcell.ColorGray

// cursorBackgroundColor is the background of the whole cursor line (Blue). It is stronger than selectedBackgroundColor.
const cursorBackgroundColor = tcell.ColorNavy

const helpText = "↑↓/C-p C-n:move  PgUp PgDn/M-v C-v:page  Enter:select action  Esc/q:back  C-c:quit"

var spanStyles = map[jsonview.SpanKind]tcell.Style{
	jsonview.SpanPunctuation: tcell.StyleDefault,
	jsonview.SpanKey:         tcell.StyleDefault.Foreground(tcell.ColorTeal),
	jsonview.SpanString:      tcell.StyleDefault.Foreground(tcell.ColorGreen),
	jsonview.SpanNumber:      tcell.StyleDefault.Foreground(tcell.ColorOlive),
	jsonview.SpanLiteral:     tcell.StyleDefault.Foreground(tcell.ColorPurple),
}

// bodyHeight is the number of rows in the JSON area, excluding the header and footer.
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
	if app.popup != nil {
		app.drawPopup(width, height)
	}
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
	selectedObject, hasSelectedObject := view.document.SelectableObjectAt(view.cursorLine)

	for row := 0; row < app.bodyHeight(); row++ {
		lineNumber := view.topLine + row
		if lineNumber >= len(view.document.Lines) {
			return
		}
		screenRow := row + 1
		isCursorLine := lineNumber == view.cursorLine
		isSelected := hasSelectedObject && lineNumber >= selectedObject.StartLine && lineNumber <= selectedObject.EndLine

		hasBackground := true
		var background tcell.Color
		switch {
		case isCursorLine:
			background = cursorBackgroundColor
		case isSelected:
			background = selectedBackgroundColor
		default:
			hasBackground = false
		}
		if hasBackground {
			app.fillRow(screenRow, width, tcell.StyleDefault.Background(background))
		}

		x := 0
		for _, span := range view.document.Lines[lineNumber].Spans {
			style := spanStyles[span.Kind]
			if hasBackground {
				style = style.Background(background)
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
	app.drawText(0, row, helpText, tcell.StyleDefault.Dim(true), width)
}

func (app *App) drawPopup(screenWidth, screenHeight int) {
	popup := app.popup
	title := string(popup.object.Descriptor.MediaType)

	contentWidth := uniseg.StringWidth(title)
	for _, action := range popup.actions {
		contentWidth = max(contentWidth, uniseg.StringWidth(action.Label())+2)
	}
	boxWidth := min(contentWidth+4, screenWidth)
	boxHeight := min(len(popup.actions)+4, screenHeight)
	left := (screenWidth - boxWidth) / 2
	top := (screenHeight - boxHeight) / 2
	right := left + boxWidth - 1
	bottom := top + boxHeight - 1

	borderStyle := tcell.StyleDefault
	for y := top; y <= bottom; y++ {
		for x := left; x <= right; x++ {
			app.screen.Put(x, y, " ", borderStyle)
		}
	}
	for x := left + 1; x < right; x++ {
		app.screen.Put(x, top, "─", borderStyle)
		app.screen.Put(x, bottom, "─", borderStyle)
	}
	for y := top + 1; y < bottom; y++ {
		app.screen.Put(left, y, "│", borderStyle)
		app.screen.Put(right, y, "│", borderStyle)
	}
	app.screen.Put(left, top, "┌", borderStyle)
	app.screen.Put(right, top, "┐", borderStyle)
	app.screen.Put(left, bottom, "└", borderStyle)
	app.screen.Put(right, bottom, "┘", borderStyle)

	app.drawText(left+2, top+1, title, tcell.StyleDefault.Bold(true), right-1)
	for index, action := range popup.actions {
		style := tcell.StyleDefault
		row := top + 2 + index
		if index == popup.selected {
			style = style.Reverse(true)
			for x := left + 1; x < right; x++ {
				app.screen.Put(x, row, " ", style)
			}
		}
		app.drawText(left+2, row, action.Label(), style, right-1)
	}
}

func (app *App) fillRow(row, width int, style tcell.Style) {
	for x := 0; x < width; x++ {
		app.screen.Put(x, row, " ", style)
	}
}

// drawText draws text up to limitX (exclusive) and returns the next draw position. It accounts for the width of full-width characters.
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
