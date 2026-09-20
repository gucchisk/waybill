// Package tui は tcell を使った端末UIを提供する。
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

// contentView は表示中の1つのJSON画面。
type contentView struct {
	title      string
	document   *jsonview.Document
	cursorLine int
	topLine    int
}

// actionPopup は mediaType を持つオブジェクトに対する操作の選択肢。
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

// Run は終了操作が行われるまでイベントループを回す。
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

// handleKey はキー入力を処理し、アプリを終了すべきなら true を返す。
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
		app.popup = nil
	case commandConfirm:
		app.popup = nil
		app.execute(popup.object.Descriptor, popup.actions[popup.selected])
	}
}

// execute は通信を伴う操作をバックグラウンドで実行し、結果はイベントループ上で反映する。
func (app *App) execute(descriptor domain.Descriptor, action domain.Action) {
	app.isBusy = true
	app.statusMessage = fmt.Sprintf("%s中... (%s)", action.Label(), descriptor.Digest)

	go func() {
		applyResult := app.runAction(descriptor, action)
		_ = app.screen.PostEvent(tcell.NewEventInterrupt(applyResult))
	}()
}

// runAction は状態を触らずに処理だけを行い、結果を反映する関数を返す。
func (app *App) runAction(descriptor domain.Descriptor, action domain.Action) func() {
	fail := func(err error) func() {
		return func() {
			app.isBusy = false
			app.statusMessage = "エラー: " + err.Error()
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
			app.statusMessage = "保存しました: " + savedPath
		}
	}
	return fail(fmt.Errorf("unsupported action %d", action))
}

// Run は実端末で画面を開き、終了操作が行われるまでブロックする。
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
