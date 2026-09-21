package registry

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/opencontainers/go-digest"
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/types/descriptor"
	"github.com/regclient/regclient/types/ref"

	"github.com/gucchisk/waybill/internal/domain"
)

// RegclientRepository fetches content from a registry using regclient.
type RegclientRepository struct {
	client *regclient.RegClient
}

// NewRegclientRepository creates a client that uses Docker's credential and certificate settings.
func NewRegclientRepository() *RegclientRepository {
	return &RegclientRepository{
		client: regclient.New(regclient.WithDockerCreds(), regclient.WithDockerCerts()),
	}
}

func (repository *RegclientRepository) ResolveImage(ctx context.Context, imageReference string) (string, domain.Descriptor, error) {
	imageRef, err := ref.New(imageReference)
	if err != nil {
		return "", domain.Descriptor{}, fmt.Errorf("parse image reference %q: %w", imageReference, err)
	}
	manifest, err := repository.client.ManifestGet(ctx, imageRef)
	if err != nil {
		return "", domain.Descriptor{}, fmt.Errorf("get manifest of %q: %w", imageReference, err)
	}
	manifestDescriptor := manifest.GetDescriptor()
	return repositoryName(imageRef), domain.Descriptor{
		MediaType: domain.MediaType(manifestDescriptor.MediaType),
		Digest:    manifestDescriptor.Digest.String(),
		Size:      manifestDescriptor.Size,
	}, nil
}

func (repository *RegclientRepository) FetchContent(ctx context.Context, repositoryName string, target domain.Descriptor) (io.ReadCloser, error) {
	repositoryRef, err := ref.New(repositoryName)
	if err != nil {
		return nil, fmt.Errorf("parse repository %q: %w", repositoryName, err)
	}
	contentRef := repositoryRef.SetDigest(target.Digest)

	if target.MediaType.IsManifest() {
		manifest, err := repository.client.ManifestGet(ctx, contentRef)
		if err != nil {
			return nil, fmt.Errorf("get manifest %s: %w", target.Digest, err)
		}
		rawManifest, err := manifest.RawBody()
		if err != nil {
			return nil, fmt.Errorf("read manifest %s: %w", target.Digest, err)
		}
		return io.NopCloser(bytes.NewReader(rawManifest)), nil
	}

	blob, err := repository.client.BlobGet(ctx, contentRef, descriptor.Descriptor{
		MediaType: string(target.MediaType),
		Digest:    digest.Digest(target.Digest),
		Size:      target.Size,
	})
	if err != nil {
		return nil, fmt.Errorf("get blob %s: %w", target.Digest, err)
	}
	return blob, nil
}

// repositoryName returns "registry/path" without the tag or digest.
func repositoryName(imageRef ref.Ref) string {
	return imageRef.Registry + "/" + imageRef.Repository
}
