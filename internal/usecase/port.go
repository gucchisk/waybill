package usecase

import (
	"context"
	"io"

	"github.com/gucchisk/waybill/internal/domain"
)

type ImageResolver interface {
	ResolveImage(ctx context.Context, imageReference string) (repository string, root domain.Descriptor, err error)
}

type ContentFetcher interface {
	FetchContent(ctx context.Context, repository string, descriptor domain.Descriptor) (io.ReadCloser, error)
}

type FileSaver interface {
	SaveFile(fileName string, content io.Reader) (savedPath string, err error)
}
