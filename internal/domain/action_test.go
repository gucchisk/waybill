package domain

import (
	"reflect"
	"testing"
)

func TestActionsFor(t *testing.T) {
	viewAndDownload := []Action{ActionView, ActionDownload}
	downloadOnly := []Action{ActionDownload}

	testCases := []struct {
		mediaType MediaType
		expected  []Action
	}{
		{MediaTypeOCIImageIndex, viewAndDownload},
		{MediaTypeOCIImageManifest, viewAndDownload},
		{MediaTypeOCIImageConfig, viewAndDownload},
		{MediaTypeInToto, viewAndDownload},
		{MediaTypeDockerManifest, viewAndDownload},
		{MediaTypeOCILayerTarGzip, downloadOnly},
		{MediaTypeOCILayerTarZstd, downloadOnly},
		{MediaTypeDockerLayerGzip, downloadOnly},
		{MediaTypeOCIEmpty, []Action{ActionView}},
		{"application/vnd.unknown+json", viewAndDownload},
		{"application/octet-stream", downloadOnly},
	}
	for _, testCase := range testCases {
		actual := ActionsFor(testCase.mediaType)
		if !reflect.DeepEqual(actual, testCase.expected) {
			t.Errorf("ActionsFor(%q) = %v, want %v", testCase.mediaType, actual, testCase.expected)
		}
	}
}

func TestDownloadFileName(t *testing.T) {
	descriptor := Descriptor{MediaType: MediaTypeOCILayerTarGzip, Digest: "sha256:abc"}
	if actual := descriptor.DownloadFileName(); actual != "sha256-abc.tar.gz" {
		t.Errorf("got %q", actual)
	}
	descriptor = Descriptor{MediaType: MediaTypeInToto, Digest: "sha256:abc"}
	if actual := descriptor.DownloadFileName(); actual != "sha256-abc.json" {
		t.Errorf("got %q", actual)
	}
}
