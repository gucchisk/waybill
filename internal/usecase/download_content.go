package usecase

import (
	"context"

	"github.com/gucchisk/waybill/internal/domain"
)

type DownloadContent struct {
	fetcher ContentFetcher
	saver   FileSaver
}

func NewDownloadContent(fetcher ContentFetcher, saver FileSaver) *DownloadContent {
	return &DownloadContent{fetcher: fetcher, saver: saver}
}

func (useCase *DownloadContent) Execute(ctx context.Context, repository string, descriptor domain.Descriptor) (string, error) {
	content, err := useCase.fetcher.FetchContent(ctx, repository, descriptor)
	if err != nil {
		return "", err
	}
	defer content.Close()

	return useCase.saver.SaveFile(descriptor.DownloadFileName(), content)
}
