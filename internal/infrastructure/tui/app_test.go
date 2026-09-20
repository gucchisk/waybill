package tui

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/gucchisk/waybill/internal/domain"
	"github.com/gucchisk/waybill/internal/usecase"
)

const indexJSON = `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.index.v1+json","manifests":[` +
	`{"mediaType":"application/vnd.oci.image.manifest.v1+json","digest":"sha256:m1","size":5}]}`

type staticFetcher map[string]string

func (fetcher staticFetcher) FetchContent(_ context.Context, _ string, descriptor domain.Descriptor) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(fetcher[descriptor.Digest])), nil
}

type discardSaver struct{ fileName string }

func (saver *discardSaver) SaveFile(fileName string, content io.Reader) (string, error) {
	saver.fileName = fileName
	_, err := io.Copy(io.Discard, content)
	return "/out/" + fileName, err
}

func newTestApp(t *testing.T) (*App, tcell.SimulationScreen, *discardSaver) {
	t.Helper()
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(100, 20)

	fetcher := staticFetcher{"sha256:m1": `{"schemaVersion":2,"layers":[]}`}
	saver := &discardSaver{}
	app, err := NewApp(context.Background(), screen, "example.com/repo",
		domain.Descriptor{MediaType: domain.MediaTypeOCIImageIndex, Digest: "sha256:root"}, []byte(indexJSON),
		Dependencies{ViewContent: usecase.NewViewContent(fetcher), DownloadContent: usecase.NewDownloadContent(fetcher, saver)})
	if err != nil {
		t.Fatal(err)
	}
	return app, screen, saver
}

func keyPress(key tcell.Key) *tcell.EventKey { return tcell.NewEventKey(key, 0, tcell.ModNone) }

func TestCursorMovesWithArrowAndEmacsKeys(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.handleKey(keyPress(tcell.KeyDown))
	app.handleKey(keyPress(tcell.KeyCtrlN))
	if app.currentView().cursorLine != 2 {
		t.Fatalf("cursorLine = %d", app.currentView().cursorLine)
	}
	app.handleKey(keyPress(tcell.KeyCtrlP))
	app.handleKey(keyPress(tcell.KeyUp))
	app.handleKey(keyPress(tcell.KeyUp))
	if app.currentView().cursorLine != 0 {
		t.Fatalf("cursor must stay at top, got %d", app.currentView().cursorLine)
	}
}

func TestEnterOnlyOpensPopupOnSelectableObject(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.handleKey(keyPress(tcell.KeyEnter))
	if app.popup != nil {
		t.Fatal("root object has no digest, popup must not open")
	}

	for range 4 { // 行4: manifests[0] 内
		app.handleKey(keyPress(tcell.KeyDown))
	}
	app.handleKey(keyPress(tcell.KeyEnter))
	if app.popup == nil || len(app.popup.actions) != 2 {
		t.Fatalf("popup = %+v", app.popup)
	}
	app.handleKey(keyPress(tcell.KeyCtrlN))
	if app.popup.selected != 1 {
		t.Errorf("selected = %d", app.popup.selected)
	}
	app.handleKey(keyPress(tcell.KeyEscape))
	if app.popup != nil {
		t.Error("popup must close on Esc")
	}
}

func TestSelectedObjectIsReversed(t *testing.T) {
	app, screen, _ := newTestApp(t)
	for range 4 {
		app.handleKey(keyPress(tcell.KeyDown))
	}
	app.draw()

	isReversed := func(screenRow int) bool {
		_, style, _ := screen.Get(10, screenRow)
		_, _, attributes := style.Decompose()
		return attributes&tcell.AttrReverse != 0
	}
	// カーソルは行4(manifests[0]の "{")。オブジェクトは行4〜8、画面上はヘッダ1行ぶん下がって5〜9行目。
	for screenRow := 5; screenRow <= 9; screenRow++ {
		if !isReversed(screenRow) {
			t.Errorf("screen row %d must be reversed", screenRow)
		}
	}
	if isReversed(3) || isReversed(10) {
		t.Error("rows outside the selected object must not be reversed")
	}
}

func TestRunActionViewAndDownload(t *testing.T) {
	app, _, saver := newTestApp(t)
	descriptor := domain.Descriptor{MediaType: domain.MediaTypeOCIImageManifest, Digest: "sha256:m1"}

	app.runAction(descriptor, domain.ActionView)()
	if len(app.views) != 2 {
		t.Fatalf("view not pushed: %d", len(app.views))
	}
	app.handleKey(keyPress(tcell.KeyEscape))
	if len(app.views) != 1 {
		t.Fatal("Esc must pop view")
	}

	app.runAction(descriptor, domain.ActionDownload)()
	if saver.fileName != "sha256-m1.json" || !strings.Contains(app.statusMessage, "/out/sha256-m1.json") {
		t.Errorf("fileName=%q status=%q", saver.fileName, app.statusMessage)
	}
}

func TestEscapeOnRootViewQuits(t *testing.T) {
	app, _, _ := newTestApp(t)
	if !app.handleKey(keyPress(tcell.KeyEscape)) {
		t.Error("Esc on root view must quit")
	}
}
