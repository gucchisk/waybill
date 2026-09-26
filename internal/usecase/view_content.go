package usecase

import (
	"context"
	"fmt"
	"io"

	"github.com/gucchisk/waybill/internal/domain"
)

const maxViewableContentBytes = 32 << 20

type ViewContent struct {
	fetcher ContentFetcher
}

func NewViewContent(fetcher ContentFetcher) *ViewContent {
	return &ViewContent{fetcher: fetcher}
}

func (useCase *ViewContent) Execute(ctx context.Context, repository string, descriptor domain.Descriptor) ([]byte, error) {
	content, err := useCase.fetcher.FetchContent(ctx, repository, descriptor)
	if err != nil {
		return nil, err
	}
	defer content.Close()

	body, err := io.ReadAll(io.LimitReader(content, maxViewableContentBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", descriptor.Digest, err)
	}
	if len(body) > maxViewableContentBytes {
		return nil, fmt.Errorf("content %s is too large to view (limit %d bytes)", descriptor.Digest, maxViewableContentBytes)
	}
	return body, nil
}
