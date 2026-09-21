package usecase

import (
	"context"
	"io"

	"github.com/gucchisk/waybill/internal/domain"
)

// ImageResolver resolves an image reference (e.g. ghcr.io/foo/bar:latest) to the root descriptor.
// The returned repository is in "registry/path" form, without the tag or digest.
type ImageResolver interface {
	ResolveImage(ctx context.Context, imageReference string) (repository string, root domain.Descriptor, err error)
}

// ContentFetcher fetches the content that a descriptor in a repository points to.
type ContentFetcher interface {
	FetchContent(ctx context.Context, repository string, descriptor domain.Descriptor) (io.ReadCloser, error)
}

// FileSaver saves content as a file and returns the path it was saved to.
type FileSaver interface {
	SaveFile(fileName string, content io.Reader) (savedPath string, err error)
}
