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
	app, err := NewApp(context.Background(), screen, true, "example.com/repo",
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

	for range 4 { // line 4: inside manifests[0]
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

func TestSelectedObjectHasBackground(t *testing.T) {
	app, screen, _ := newTestApp(t)
	for range 4 {
		app.handleKey(keyPress(tcell.KeyDown))
	}
	app.draw()

	styleAt := func(screenRow int) (background tcell.Color, isReversed bool) {
		_, style, _ := screen.Get(10, screenRow)
		_, background, attributes := style.Decompose()
		return background, attributes&tcell.AttrReverse != 0
	}
	// The cursor is on line 4 (the "{" of manifests[0]). The object spans lines 4-8, shown at rows 5-9 on screen because of the one-line header.
	// The cursor line (row 5) is reversed; the other lines of the object have the selected background.
	if _, isReversed := styleAt(5); !isReversed {
		t.Error("the cursor line must be drawn in reverse video")
	}
	for screenRow := 6; screenRow <= 9; screenRow++ {
		background, isReversed := styleAt(screenRow)
		if background != app.selectedBackgroundColor || isReversed {
			t.Errorf("screen row %d must have the selected background and no reverse", screenRow)
		}
	}
	for _, screenRow := range []int{3, 10} {
		background, isReversed := styleAt(screenRow)
		if background == app.selectedBackgroundColor || isReversed {
			t.Errorf("screen row %d is outside the selected object and must have no highlight", screenRow)
		}
	}
}

func TestSelectedBackgroundColorFollowsTerminalBrightness(t *testing.T) {
	if got := selectedBackgroundColorFor(true); got != tcell.ColorGray {
		t.Errorf("dark background: %v, want Bright Black", got)
	}
	if got := selectedBackgroundColorFor(false); got != tcell.ColorSilver {
		t.Errorf("light background: %v, want White", got)
	}
}

func TestCursorLineStyleIsNotPureBlackOnLightBackground(t *testing.T) {
	if _, _, attributes := cursorLineStyleFor(true).Decompose(); attributes&tcell.AttrReverse == 0 {
		t.Error("dark background: the cursor line must be reverse video")
	}
	foreground, background, attributes := cursorLineStyleFor(false).Decompose()
	if attributes&tcell.AttrReverse != 0 || foreground != tcell.ColorWhite || background != tcell.ColorGray {
		t.Errorf("light background: got fg=%v bg=%v attributes=%v, want White on Bright Black without reverse", foreground, background, attributes)
	}
}

func TestCursorLineIsHighlightedEvenWithoutSelectableObject(t *testing.T) {
	app, screen, _ := newTestApp(t)
	app.draw()

	// The cursor is on line 0 (the root "{"), which is not inside a selectable object.
	_, style, _ := screen.Get(10, 1)
	_, _, attributes := style.Decompose()
	if attributes&tcell.AttrReverse == 0 {
		t.Error("the cursor line must be drawn in reverse video")
	}
	if text, _, _ := screen.Get(0, 1); text == ">" {
		t.Error("the \">\" cursor marker must not be drawn")
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
