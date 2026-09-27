package usecase

import (
	"context"

	"github.com/gucchisk/waybill/internal/domain"
)

type OpenImage struct {
	resolver ImageResolver
}

func NewOpenImage(resolver ImageResolver) *OpenImage {
	return &OpenImage{resolver: resolver}
}

func (useCase *OpenImage) Execute(ctx context.Context, imageReference string) (repository string, root domain.Descriptor, err error) {
	return useCase.resolver.ResolveImage(ctx, imageReference)
}
