package tui

import (
	"context"
	"fmt"
	"slices"

	"github.com/gdamore/tcell/v2"

	"github.com/gucchisk/waybill/internal/adapter/jsonview"
	"github.com/gucchisk/waybill/internal/domain"
	"github.com/gucchisk/waybill/internal/usecase"
)

type Dependencies struct {
	ViewContent     *usecase.ViewContent
	DownloadContent *usecase.DownloadContent
}

type contentView struct {
	title               string
	document            *jsonview.Document
	foldedObjectIndexes map[int]bool
	displayLines        []jsonview.DisplayLine
	cursorLine          int
	topLine             int
}

type App struct {
	screen       tcell.Screen
	ctx          context.Context
	repository   string
	dependencies Dependencies

	cursorLineStyle         tcell.Style
	selectedBackgroundColor tcell.Color

	views         []*contentView
	statusMessage string
	isBusy        bool
}

func NewApp(ctx context.Context, screen tcell.Screen, hasDarkBackground bool, repository string, rootDescriptor domain.Descriptor, rootContent []byte, dependencies Dependencies) (*App, error) {
	app := &App{
		ctx: ctx, screen: screen, repository: repository, dependencies: dependencies,
		cursorLineStyle:         cursorLineStyleFor(hasDarkBackground),
		selectedBackgroundColor: selectedBackgroundColorFor(hasDarkBackground),
	}
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
	foldedObjectIndexes := map[int]bool{}
	return &contentView{
		title:               fmt.Sprintf("%s  %s", descriptor.MediaType, descriptor.Digest),
		document:            document,
		foldedObjectIndexes: foldedObjectIndexes,
		displayLines:        document.DisplayLines(foldedObjectIndexes),
	}, nil
}

func (view *contentView) cursorDocumentLine() int {
	return view.displayLines[view.cursorLine].DocumentLine
}

func (view *contentView) foldTarget() (objectIndex int, isFolded bool) {
	displayLine := view.displayLines[view.cursorLine]
	if displayLine.FoldedObjectIndex >= 0 {
		return displayLine.FoldedObjectIndex, true
	}
	return view.document.Lines[displayLine.DocumentLine].ObjectIndex, false
}

func (view *contentView) toggleFold() {
	objectIndex, isFolded := view.foldTarget()
	if objectIndex < 0 {
		return
	}
	if isFolded {
		delete(view.foldedObjectIndexes, objectIndex)
	} else {
		view.foldedObjectIndexes[objectIndex] = true
	}
	view.displayLines = view.document.DisplayLines(view.foldedObjectIndexes)

	startLine := view.document.Objects[objectIndex].StartLine
	view.cursorLine = slices.IndexFunc(view.displayLines, func(displayLine jsonview.DisplayLine) bool {
		return displayLine.DocumentLine == startLine
	})
}

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

func (app *App) handleKey(event *tcell.EventKey) (shouldQuit bool) {
	keyCommand := commandFor(event)
	if keyCommand == commandQuit {
		return true
	}
	if app.isBusy {
		return false
	}
	return app.handleViewCommand(keyCommand)
}

func (app *App) currentView() *contentView {
	return app.views[len(app.views)-1]
}

func (app *App) handleViewCommand(keyCommand command) (shouldQuit bool) {
	view := app.currentView()
	lastLine := len(view.displayLines) - 1
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
	case commandToggleFold:
		view.toggleFold()
	case commandView:
		app.executeOnSelectedObject(domain.ActionView)
	case commandDownload:
		app.executeOnSelectedObject(domain.ActionDownload)
	case commandCopyValue:
		app.copyCursorLineValue()
	case commandBack:
		if len(app.views) == 1 {
			return true
		}
		app.views = app.views[:len(app.views)-1]
	}
	return false
}

func (app *App) copyCursorLineValue() {
	view := app.currentView()
	value, ok := view.document.CopyableValueAt(view.cursorDocumentLine())
	if !ok {
		return
	}
	app.screen.SetClipboard([]byte(value))
	app.statusMessage = "Copied to clipboard"
}

func (app *App) availableActions() (jsonview.SelectableObject, []domain.Action) {
	view := app.currentView()
	object, ok := view.document.SelectableObjectAt(view.cursorDocumentLine())
	if !ok {
		return jsonview.SelectableObject{}, nil
	}
	return object, domain.ActionsFor(object.Descriptor.MediaType)
}

func (app *App) executeOnSelectedObject(action domain.Action) {
	object, actions := app.availableActions()
	if !slices.Contains(actions, action) {
		return
	}
	app.execute(object.Descriptor, action)
}

func (app *App) execute(descriptor domain.Descriptor, action domain.Action) {
	app.isBusy = true
	app.statusMessage = fmt.Sprintf("%s... (%s)", action.ProgressLabel(), descriptor.Digest)

	go func() {
		applyResult := app.runAction(descriptor, action)
		_ = app.screen.PostEvent(tcell.NewEventInterrupt(applyResult))
	}()
}

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

func Run(ctx context.Context, repository string, rootDescriptor domain.Descriptor, rootContent []byte, dependencies Dependencies, theme Theme) error {
	hasDarkBackground := theme.hasDarkBackground()

	screen, err := tcell.NewScreen()
	if err != nil {
		return fmt.Errorf("create screen: %w", err)
	}
	if err := screen.Init(); err != nil {
		return fmt.Errorf("init screen: %w", err)
	}
	defer screen.Fini()

	app, err := NewApp(ctx, screen, hasDarkBackground, repository, rootDescriptor, rootContent, dependencies)
	if err != nil {
		return err
	}
	app.Run()
	return nil
}
