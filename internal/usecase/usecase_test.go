package usecase

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/gucchisk/waybill/internal/domain"
)

type fakeFetcher struct {
	body string
	err  error
}

func (fetcher fakeFetcher) FetchContent(context.Context, string, domain.Descriptor) (io.ReadCloser, error) {
	if fetcher.err != nil {
		return nil, fetcher.err
	}
	return io.NopCloser(strings.NewReader(fetcher.body)), nil
}

type fakeSaver struct {
	savedName string
	savedBody string
}

func (saver *fakeSaver) SaveFile(fileName string, content io.Reader) (string, error) {
	body, err := io.ReadAll(content)
	saver.savedName = fileName
	saver.savedBody = string(body)
	return "/out/" + fileName, err
}

func TestViewContent(t *testing.T) {
	body, err := NewViewContent(fakeFetcher{body: `{"a":1}`}).Execute(context.Background(), "r", domain.Descriptor{})
	if err != nil || string(body) != `{"a":1}` {
		t.Fatalf("got %q, %v", body, err)
	}

	fetchError := errors.New("boom")
	if _, err := NewViewContent(fakeFetcher{err: fetchError}).Execute(context.Background(), "r", domain.Descriptor{}); !errors.Is(err, fetchError) {
		t.Fatalf("expected fetch error, got %v", err)
	}
}

func TestDownloadContent(t *testing.T) {
	saver := &fakeSaver{}
	descriptor := domain.Descriptor{MediaType: domain.MediaTypeOCILayerTarGzip, Digest: "sha256:abc"}
	savedPath, err := NewDownloadContent(fakeFetcher{body: "layer"}, saver).Execute(context.Background(), "r", descriptor)
	if err != nil {
		t.Fatal(err)
	}
	if savedPath != "/out/sha256-abc.tar.gz" || saver.savedBody != "layer" {
		t.Errorf("path=%q body=%q", savedPath, saver.savedBody)
	}
}
