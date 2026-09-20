package jsonview

import (
	"strings"
	"testing"

	"github.com/gucchisk/waybill/internal/domain"
)

func lineText(line Line) string {
	var builder strings.Builder
	for _, span := range line.Spans {
		builder.WriteString(span.Text)
	}
	return builder.String()
}

const sampleManifest = `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json",` +
	`"config":{"mediaType":"application/vnd.oci.image.config.v1+json","digest":"sha256:c0","size":10},` +
	`"layers":[{"mediaType":"application/vnd.oci.image.layer.v1.tar+gzip","digest":"sha256:l1","size":20},{"note":"plain"}],` +
	`"annotations":{"a":"<b>"},"empty":{}}`

func TestParsePreservesOrderAndFormats(t *testing.T) {
	document, err := Parse([]byte(sampleManifest))
	if err != nil {
		t.Fatal(err)
	}
	expectedHead := []string{
		`{`,
		`  "schemaVersion": 2,`,
		`  "mediaType": "application/vnd.oci.image.manifest.v1+json",`,
		`  "config": {`,
	}
	for index, expected := range expectedHead {
		if actual := lineText(document.Lines[index]); actual != expected {
			t.Errorf("line %d = %q, want %q", index, actual, expected)
		}
	}
	last := len(document.Lines) - 1
	if lineText(document.Lines[last]) != "}" || lineText(document.Lines[last-1]) != `  "empty": {}` {
		t.Errorf("unexpected tail: %q / %q", lineText(document.Lines[last-1]), lineText(document.Lines[last]))
	}
	if !strings.Contains(lineText(document.Lines[last-3]), `"<b>"`) {
		t.Errorf("html chars must not be escaped: %q", lineText(document.Lines[last-3]))
	}
}

func TestSelectableObjects(t *testing.T) {
	document, err := Parse([]byte(sampleManifest))
	if err != nil {
		t.Fatal(err)
	}
	// ルート(digestなし)と {"note"} は選択対象外。config と layers[0] の2つ。
	if len(document.SelectableObjects) != 2 {
		t.Fatalf("got %d selectable objects: %+v", len(document.SelectableObjects), document.SelectableObjects)
	}
	if _, ok := document.SelectableObjectAt(0); ok {
		t.Error("root object must not be selectable")
	}

	configObject, ok := document.SelectableObjectAt(4) // config 内の "mediaType" 行
	if !ok || configObject.Descriptor != (domain.Descriptor{MediaType: domain.MediaTypeOCIImageConfig, Digest: "sha256:c0", Size: 10}) {
		t.Errorf("config object = %+v, %v", configObject, ok)
	}
	if lineText(document.Lines[configObject.StartLine]) != `  "config": {` || lineText(document.Lines[configObject.EndLine]) != `  },` {
		t.Errorf("config range = %d-%d", configObject.StartLine, configObject.EndLine)
	}
}

func TestNestedSelectableObjectPrefersInnermost(t *testing.T) {
	nested := `{"mediaType":"application/vnd.oci.image.index.v1+json","digest":"sha256:outer",` +
		`"inner":{"mediaType":"application/vnd.oci.image.manifest.v1+json","digest":"sha256:inner"}}`
	document, err := Parse([]byte(nested))
	if err != nil {
		t.Fatal(err)
	}
	if outer, _ := document.SelectableObjectAt(1); outer.Descriptor.Digest != "sha256:outer" {
		t.Errorf("line 1 = %+v", outer)
	}
	if inner, _ := document.SelectableObjectAt(4); inner.Descriptor.Digest != "sha256:inner" {
		t.Errorf("line 4 = %+v", inner)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	if _, err := Parse([]byte(`{"a":`)); err == nil {
		t.Error("expected error")
	}
}
