// Package tui provides a terminal UI built on tcell.
package tui

import (
	"context"
	"fmt"

	"github.com/gdamore/tcell/v2"

	"github.com/gucchisk/waybill/internal/adapter/jsonview"
	"github.com/gucchisk/waybill/internal/domain"
	"github.com/gucchisk/waybill/internal/usecase"
)

type Dependencies struct {
	ViewContent     *usecase.ViewContent
	DownloadContent *usecase.DownloadContent
}

// contentView is one JSON screen being displayed.
type contentView struct {
	title      string
	document   *jsonview.Document
	cursorLine int
	topLine    int
}

// actionPopup holds the actions available for an object that has a mediaType.
type actionPopup struct {
	object   jsonview.SelectableObject
	actions  []domain.Action
	selected int
}

type App struct {
	screen       tcell.Screen
	ctx          context.Context
	repository   string
	dependencies Dependencies

	views         []*contentView
	popup         *actionPopup
	statusMessage string
	isBusy        bool
}

func NewApp(ctx context.Context, screen tcell.Screen, repository string, rootDescriptor domain.Descriptor, rootContent []byte, dependencies Dependencies) (*App, error) {
	app := &App{ctx: ctx, screen: screen, repository: repository, dependencies: dependencies}
	rootView, err := newContentView(rootDescriptor, rootContent)
	if err != nil {
		return nil, err
	}
	app.views = []*contentView{rootView}
	return app, nil
}

func newContentView(descriptor domain.Descriptor, content []byte) (*contentView, error) {
	document, err := jsonview.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("%s is not viewable as JSON: %w", descriptor.Digest, err)
	}
	return &contentView{
		title:    fmt.Sprintf("%s  %s", descriptor.MediaType, descriptor.Digest),
		document: document,
	}, nil
}

// Run runs the event loop until a quit operation is performed.
func (app *App) Run() {
	for {
		app.draw()
		switch event := app.screen.PollEvent().(type) {
		case nil:
			return
		case *tcell.EventResize:
			app.screen.Sync()
		case *tcell.EventKey:
			if app.handleKey(event) {
				return
			}
		case *tcell.EventInterrupt:
			if applyResult, ok := event.Data().(func()); ok {
				applyResult()
			}
		}
	}
}

// handleKey handles a key input and returns true if the app should quit.
func (app *App) handleKey(event *tcell.EventKey) (shouldQuit bool) {
	keyCommand := commandFor(event)
	if keyCommand == commandQuit {
		return true
	}
	if app.isBusy {
		return false
	}
	if app.popup != nil {
		app.handlePopupCommand(keyCommand)
		return false
	}
	return app.handleViewCommand(keyCommand)
}

func (app *App) currentView() *contentView {
	return app.views[len(app.views)-1]
}

func (app *App) handleViewCommand(keyCommand command) (shouldQuit bool) {
	view := app.currentView()
	lastLine := len(view.document.Lines) - 1
	pageSize := max(app.bodyHeight()-1, 1)
	app.statusMessage = ""

	switch keyCommand {
	case commandUp:
		view.cursorLine = max(view.cursorLine-1, 0)
	case commandDown:
		view.cursorLine = min(view.cursorLine+1, lastLine)
	case commandPageUp:
		view.cursorLine = max(view.cursorLine-pageSize, 0)
	case commandPageDown:
		view.cursorLine = min(view.cursorLine+pageSize, lastLine)
	case commandTop:
		view.cursorLine = 0
	case commandBottom:
		view.cursorLine = lastLine
	case commandConfirm:
		app.openPopup()
	case commandBack:
		if len(app.views) == 1 {
			return true
		}
		app.views = app.views[:len(app.views)-1]
	}
	return false
}

func (app *App) openPopup() {
	object, ok := app.currentView().document.SelectableObjectAt(app.currentView().cursorLine)
	if !ok {
		return
	}
	actions := domain.ActionsFor(object.Descriptor.MediaType)
	if len(actions) == 0 {
		return
	}
	app.popup = &actionPopup{object: object, actions: actions}
}

func (app *App) closePopup() {
	app.popup = nil
}

func (app *App) handlePopupCommand(keyCommand command) {
	popup := app.popup
	switch keyCommand {
	case commandUp:
		popup.selected = max(popup.selected-1, 0)
	case commandDown:
		popup.selected = min(popup.selected+1, len(popup.actions)-1)
	case commandTop, commandPageUp:
		popup.selected = 0
	case commandBottom, commandPageDown:
		popup.selected = len(popup.actions) - 1
	case commandBack:
		app.closePopup()
	case commandConfirm:
		app.closePopup()
		app.execute(popup.object.Descriptor, popup.actions[popup.selected])
	}
}

// execute runs an operation that involves network access in the background and applies the result on the event loop.
func (app *App) execute(descriptor domain.Descriptor, action domain.Action) {
	app.isBusy = true
	app.statusMessage = fmt.Sprintf("%s... (%s)", action.Label(), descriptor.Digest)

	go func() {
		applyResult := app.runAction(descriptor, action)
		_ = app.screen.PostEvent(tcell.NewEventInterrupt(applyResult))
	}()
}

// runAction only performs the work without touching state, and returns a function that applies the result.
func (app *App) runAction(descriptor domain.Descriptor, action domain.Action) func() {
	fail := func(err error) func() {
		return func() {
			app.isBusy = false
			app.statusMessage = "Error: " + err.Error()
		}
	}

	switch action {
	case domain.ActionView:
		content, err := app.dependencies.ViewContent.Execute(app.ctx, app.repository, descriptor)
		if err != nil {
			return fail(err)
		}
		view, err := newContentView(descriptor, content)
		if err != nil {
			return fail(err)
		}
		return func() {
			app.isBusy = false
			app.statusMessage = ""
			app.views = append(app.views, view)
		}
	case domain.ActionDownload:
		savedPath, err := app.dependencies.DownloadContent.Execute(app.ctx, app.repository, descriptor)
		if err != nil {
			return fail(err)
		}
		return func() {
			app.isBusy = false
			app.statusMessage = "Saved: " + savedPath
		}
	}
	return fail(fmt.Errorf("unsupported action %d", action))
}

// Run opens the screen on the real terminal and blocks until a quit operation is performed.
func Run(ctx context.Context, repository string, rootDescriptor domain.Descriptor, rootContent []byte, dependencies Dependencies) error {
	screen, err := tcell.NewScreen()
	if err != nil {
		return fmt.Errorf("create screen: %w", err)
	}
	if err := screen.Init(); err != nil {
		return fmt.Errorf("init screen: %w", err)
	}
	defer screen.Fini()

	app, err := NewApp(ctx, screen, repository, rootDescriptor, rootContent, dependencies)
	if err != nil {
		return err
	}
	app.Run()
	return nil
}
