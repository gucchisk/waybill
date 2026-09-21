package domain

import "strings"

// MediaType represents an OCI/Docker mediaType.
type MediaType string

const (
	MediaTypeOCIImageIndex    MediaType = "application/vnd.oci.image.index.v1+json"
	MediaTypeOCIImageManifest MediaType = "application/vnd.oci.image.manifest.v1+json"
	MediaTypeOCIImageConfig   MediaType = "application/vnd.oci.image.config.v1+json"
	MediaTypeOCIEmpty         MediaType = "application/vnd.oci.empty.v1+json"

	MediaTypeOCILayerTar     MediaType = "application/vnd.oci.image.layer.v1.tar"
	MediaTypeOCILayerTarGzip MediaType = "application/vnd.oci.image.layer.v1.tar+gzip"
	MediaTypeOCILayerTarZstd MediaType = "application/vnd.oci.image.layer.v1.tar+zstd"

	MediaTypeDockerManifest     MediaType = "application/vnd.docker.distribution.manifest.v2+json"
	MediaTypeDockerManifestList MediaType = "application/vnd.docker.distribution.manifest.list.v2+json"
	MediaTypeDockerImageConfig  MediaType = "application/vnd.docker.container.image.v1+json"
	MediaTypeDockerLayerGzip    MediaType = "application/vnd.docker.image.rootfs.diff.tar.gzip"

	MediaTypeInToto           MediaType = "application/vnd.in-toto+json"
	MediaTypeDSSEEnvelope     MediaType = "application/vnd.dsse.envelope.v1+json"
	MediaTypeSigstoreBundle   MediaType = "application/vnd.dev.sigstore.bundle.v0.3+json"
	MediaTypeCosignSimpleSign MediaType = "application/vnd.dev.cosign.simplesigning.v1+json"
)

// Descriptor is the information that points to content in a registry.
type Descriptor struct {
	MediaType MediaType
	Digest    string
	Size      int64
}

// IsManifest reports whether the content (index / manifest) should be fetched through the manifest API.
func (mediaType MediaType) IsManifest() bool {
	switch mediaType {
	case MediaTypeOCIImageIndex, MediaTypeOCIImageManifest,
		MediaTypeDockerManifest, MediaTypeDockerManifestList:
		return true
	}
	return false
}

func (mediaType MediaType) isJSON() bool {
	return strings.HasSuffix(string(mediaType), "+json")
}

// FileExtension returns the file extension used for downloads, including the leading dot.
func (mediaType MediaType) FileExtension() string {
	switch mediaType {
	case MediaTypeOCILayerTarGzip, MediaTypeDockerLayerGzip:
		return ".tar.gz"
	case MediaTypeOCILayerTar:
		return ".tar"
	case MediaTypeOCILayerTarZstd:
		return ".tar.zst"
	}
	if mediaType.isJSON() {
		return ".json"
	}
	return ".bin"
}

// DownloadFileName returns the download file name derived from the digest (e.g. sha256-xxxx.tar.gz).
func (descriptor Descriptor) DownloadFileName() string {
	return strings.ReplaceAll(descriptor.Digest, ":", "-") + descriptor.MediaType.FileExtension()
}
