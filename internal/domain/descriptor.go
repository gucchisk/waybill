package domain

import "strings"

// MediaType はOCI/DockerのmediaTypeを表す。
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

// Descriptor は registry 上のコンテンツを指し示す情報。
type Descriptor struct {
	MediaType MediaType
	Digest    string
	Size      int64
}

// IsManifest は manifest API で取得すべきコンテンツ(index / manifest)かを返す。
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

// FileExtension はダウンロード時のファイル拡張子(先頭のドット付き)を返す。
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

// DownloadFileName は digest 由来のダウンロードファイル名(例: sha256-xxxx.tar.gz)を返す。
func (descriptor Descriptor) DownloadFileName() string {
	return strings.ReplaceAll(descriptor.Digest, ":", "-") + descriptor.MediaType.FileExtension()
}
