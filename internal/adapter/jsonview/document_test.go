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
	// The root (no digest) and {"note"} are not selectable; only config and layers[0] are.
	if len(document.SelectableObjects) != 2 {
		t.Fatalf("got %d selectable objects: %+v", len(document.SelectableObjects), document.SelectableObjects)
	}
	if _, ok := document.SelectableObjectAt(0); ok {
		t.Error("root object must not be selectable")
	}

	configObject, ok := document.SelectableObjectAt(4) // the "mediaType" line inside config
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

func displayLineTexts(displayLines []DisplayLine) []string {
	texts := make([]string, len(displayLines))
	for index, displayLine := range displayLines {
		texts[index] = lineText(Line{Spans: displayLine.Spans})
	}
	return texts
}

func TestDisplayLinesFoldObject(t *testing.T) {
	document, err := Parse([]byte(sampleManifest))
	if err != nil {
		t.Fatal(err)
	}
	configIndex := document.Lines[4].ObjectIndex // the "mediaType" line inside config
	if configIndex < 0 || document.Objects[configIndex].StartLine != 3 || document.Objects[configIndex].EndLine != 7 {
		t.Fatalf("config object = %d %+v", configIndex, document.Objects)
	}

	displayLines := document.DisplayLines(map[int]bool{configIndex: true})
	texts := displayLineTexts(displayLines)
	if texts[3] != `  "config": {...},` || texts[4] != `  "layers": [` {
		t.Errorf("folded config must be one line: %q", texts[:5])
	}
	if displayLines[3].FoldedObjectIndex != configIndex || displayLines[3].DocumentLine != 3 || displayLines[4].DocumentLine != 8 {
		t.Errorf("unexpected display lines: %+v", displayLines[3:5])
	}
	if len(displayLines) != len(document.Lines)-4 {
		t.Errorf("got %d display lines, want %d", len(displayLines), len(document.Lines)-4)
	}
}

func TestDisplayLinesFoldRootAndNested(t *testing.T) {
	document, err := Parse([]byte(sampleManifest))
	if err != nil {
		t.Fatal(err)
	}
	rootIndex := document.Lines[0].ObjectIndex
	configIndex := document.Lines[4].ObjectIndex
	texts := displayLineTexts(document.DisplayLines(map[int]bool{rootIndex: true, configIndex: true}))
	if len(texts) != 1 || texts[0] != "{...}" {
		t.Errorf("folded root must hide everything including folded children: %q", texts)
	}
	if document.Lines[1].ObjectIndex != rootIndex {
		t.Error("a scalar member must belong to its enclosing object")
	}
}

func TestCopyableValueAt(t *testing.T) {
	document, err := Parse([]byte(`{"name":"a\"b","size":10,"ok":true,"none":null,"labels":{"k":[1,"x"]},"empty":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		line          int
		expectedValue string
		expectedOK    bool
	}{
		{0, "", false}, // root `{`
		{1, `a"b`, true},
		{2, "10", true},
		{3, "", false}, // true
		{4, "", false}, // null
		{5, "", false}, // object `{`
		{6, "", false}, // array `[`
		{7, "1", true},
		{8, "x", true},
		{9, "", false},  // closing `]`
		{11, "", false}, // empty array `[]`
	}
	for _, test := range tests {
		value, ok := document.CopyableValueAt(test.line)
		if value != test.expectedValue || ok != test.expectedOK {
			t.Errorf("CopyableValueAt(%d) = %q, %v; want %q, %v", test.line, value, ok, test.expectedValue, test.expectedOK)
		}
	}
}
