package usecase

import (
	"context"
	"io"

	"github.com/gucchisk/waybill/internal/domain"
)

// ImageResolver はイメージ参照(例: ghcr.io/foo/bar:latest)をルートのdescriptorへ解決する。
// 返す repository は tag/digest を含まない "registry/path" 形式。
type ImageResolver interface {
	ResolveImage(ctx context.Context, imageReference string) (repository string, root domain.Descriptor, err error)
}

// ContentFetcher は repository 上の descriptor が指すコンテンツを取得する。
type ContentFetcher interface {
	FetchContent(ctx context.Context, repository string, descriptor domain.Descriptor) (io.ReadCloser, error)
}

// FileSaver は内容をファイルとして保存し、保存先パスを返す。
type FileSaver interface {
	SaveFile(fileName string, content io.Reader) (savedPath string, err error)
}
