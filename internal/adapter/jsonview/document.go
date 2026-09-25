// Package jsonview converts JSON into a human-readable, line-based display model.
// It keeps the key order of the original JSON and detects objects that have both mediaType and digest as selectable.
package jsonview

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/gucchisk/waybill/internal/domain"
)

const indentUnit = "  "

// SpanKind is the kind of a text span, used for coloring.
type SpanKind int

const (
	SpanPunctuation SpanKind = iota
	SpanKey
	SpanString
	SpanNumber
	SpanLiteral
)

type Span struct {
	Text string
	Kind SpanKind
}

// Line is one displayed line. SelectableIndex is the index of the innermost selectable object the line belongs to (-1 if none).
// ObjectIndex is the index of the innermost non-empty object the line belongs to (-1 if none).
type Line struct {
	Spans           []Span
	SelectableIndex int
	ObjectIndex     int
}

// ObjectRange is a non-empty object that can be folded. The line range is inclusive on both ends.
// FoldedSpans is the line shown instead of the whole object when it is folded (e.g. `"config": {...},`).
type ObjectRange struct {
	StartLine   int
	EndLine     int
	FoldedSpans []Span
}

// DisplayLine is a line actually shown on screen after folding.
// DocumentLine is the index in Document.Lines (the start line for a folded object).
// FoldedObjectIndex is the index of the folded object this line stands for (-1 if the line is not folded).
type DisplayLine struct {
	Spans             []Span
	DocumentLine      int
	FoldedObjectIndex int
}

// SelectableObject is an object that has mediaType and digest. The line range is inclusive on both ends.
type SelectableObject struct {
	Descriptor domain.Descriptor
	StartLine  int
	EndLine    int
}

type Document struct {
	Lines             []Line
	SelectableObjects []SelectableObject
	Objects           []ObjectRange
}

// DisplayLines returns the lines to show when the objects in foldedObjectIndexes are folded.
// An object folded inside another folded object stays hidden.
func (document *Document) DisplayLines(foldedObjectIndexes map[int]bool) []DisplayLine {
	displayLines := make([]DisplayLine, 0, len(document.Lines))
	for lineNumber := 0; lineNumber < len(document.Lines); lineNumber++ {
		line := document.Lines[lineNumber]
		if objectIndex := line.ObjectIndex; objectIndex >= 0 && foldedObjectIndexes[objectIndex] && document.Objects[objectIndex].StartLine == lineNumber {
			displayLines = append(displayLines, DisplayLine{Spans: document.Objects[objectIndex].FoldedSpans, DocumentLine: lineNumber, FoldedObjectIndex: objectIndex})
			lineNumber = document.Objects[objectIndex].EndLine
			continue
		}
		displayLines = append(displayLines, DisplayLine{Spans: line.Spans, DocumentLine: lineNumber, FoldedObjectIndex: -1})
	}
	return displayLines
}

// SelectableObjectAt returns the innermost selectable object that line belongs to.
func (document *Document) SelectableObjectAt(line int) (SelectableObject, bool) {
	if line < 0 || line >= len(document.Lines) {
		return SelectableObject{}, false
	}
	index := document.Lines[line].SelectableIndex
	if index < 0 {
		return SelectableObject{}, false
	}
	return document.SelectableObjects[index], true
}

// Parse formats raw JSON into a Document.
func Parse(raw []byte) (*Document, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	root, err := parseNode(decoder)
	if err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	builder := &documentBuilder{}
	builder.render(root, nil, 0, false)
	builder.assignSelectableIndexes()
	return &builder.document, nil
}

type nodeKind int

const (
	nodeScalar nodeKind = iota
	nodeObject
	nodeArray
)

type node struct {
	kind        nodeKind
	scalarText  string // JSON representation (strings are quoted)
	scalarKind  SpanKind
	stringValue string // unquoted value when scalarKind is SpanString
	members     []member
	items       []*node
}

type member struct {
	key   string
	value *node
}

func parseNode(decoder *json.Decoder) (*node, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	switch value := token.(type) {
	case json.Delim:
		if value == '{' {
			return parseObjectBody(decoder)
		}
		return parseArrayBody(decoder)
	case string:
		return &node{kind: nodeScalar, scalarText: quoteJSONString(value), scalarKind: SpanString, stringValue: value}, nil
	case json.Number:
		return &node{kind: nodeScalar, scalarText: value.String(), scalarKind: SpanNumber}, nil
	case bool:
		return &node{kind: nodeScalar, scalarText: strconv.FormatBool(value), scalarKind: SpanLiteral}, nil
	case nil:
		return &node{kind: nodeScalar, scalarText: "null", scalarKind: SpanLiteral}, nil
	}
	return nil, fmt.Errorf("unexpected token %v", token)
}

func parseObjectBody(decoder *json.Decoder) (*node, error) {
	object := &node{kind: nodeObject}
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		key, ok := keyToken.(string)
		if !ok {
			return nil, fmt.Errorf("object key must be string, got %v", keyToken)
		}
		value, err := parseNode(decoder)
		if err != nil {
			return nil, err
		}
		object.members = append(object.members, member{key: key, value: value})
	}
	if _, err := decoder.Token(); err != nil && err != io.EOF { // '}'
		return nil, err
	}
	return object, nil
}

func parseArrayBody(decoder *json.Decoder) (*node, error) {
	array := &node{kind: nodeArray}
	for decoder.More() {
		item, err := parseNode(decoder)
		if err != nil {
			return nil, err
		}
		array.items = append(array.items, item)
	}
	if _, err := decoder.Token(); err != nil && err != io.EOF { // ']'
		return nil, err
	}
	return array, nil
}

func quoteJSONString(value string) string {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(value)
	return strings.TrimSuffix(buffer.String(), "\n")
}

type documentBuilder struct {
	document Document
	// openObjectIndexes is the stack of objects being rendered; the last one is the innermost.
	openObjectIndexes []int
}

// render appends node to lines. linePrefix is the indentation and key placed at the start of the first line.
func (builder *documentBuilder) render(target *node, linePrefix []Span, depth int, hasNextSibling bool) {
	comma := ""
	if hasNextSibling {
		comma = ","
	}
	closingIndent := Span{Text: strings.Repeat(indentUnit, depth), Kind: SpanPunctuation}

	switch target.kind {
	case nodeScalar:
		builder.appendLine(linePrefix, Span{Text: target.scalarText, Kind: target.scalarKind}, Span{Text: comma, Kind: SpanPunctuation})

	case nodeObject:
		if len(target.members) == 0 {
			builder.appendLine(linePrefix, Span{Text: "{}" + comma, Kind: SpanPunctuation})
			return
		}
		startLine := len(builder.document.Lines)
		objectIndex := len(builder.document.Objects)
		builder.document.Objects = append(builder.document.Objects, ObjectRange{
			StartLine:   startLine,
			FoldedSpans: joinSpans(linePrefix, Span{Text: "{...}" + comma, Kind: SpanPunctuation}),
		})
		builder.openObjectIndexes = append(builder.openObjectIndexes, objectIndex)
		builder.appendLine(linePrefix, Span{Text: "{", Kind: SpanPunctuation})
		for memberIndex, child := range target.members {
			childPrefix := []Span{
				{Text: strings.Repeat(indentUnit, depth+1), Kind: SpanPunctuation},
				{Text: quoteJSONString(child.key), Kind: SpanKey},
				{Text: ": ", Kind: SpanPunctuation},
			}
			builder.render(child.value, childPrefix, depth+1, memberIndex < len(target.members)-1)
		}
		builder.appendLine([]Span{closingIndent}, Span{Text: "}" + comma, Kind: SpanPunctuation})
		builder.openObjectIndexes = builder.openObjectIndexes[:len(builder.openObjectIndexes)-1]
		builder.document.Objects[objectIndex].EndLine = len(builder.document.Lines) - 1
		builder.registerIfSelectable(target, startLine, len(builder.document.Lines)-1)

	case nodeArray:
		if len(target.items) == 0 {
			builder.appendLine(linePrefix, Span{Text: "[]" + comma, Kind: SpanPunctuation})
			return
		}
		builder.appendLine(linePrefix, Span{Text: "[", Kind: SpanPunctuation})
		for itemIndex, item := range target.items {
			itemPrefix := []Span{{Text: strings.Repeat(indentUnit, depth+1), Kind: SpanPunctuation}}
			builder.render(item, itemPrefix, depth+1, itemIndex < len(target.items)-1)
		}
		builder.appendLine([]Span{closingIndent}, Span{Text: "]" + comma, Kind: SpanPunctuation})
	}
}

func (builder *documentBuilder) appendLine(linePrefix []Span, rest ...Span) {
	objectIndex := -1
	if len(builder.openObjectIndexes) > 0 {
		objectIndex = builder.openObjectIndexes[len(builder.openObjectIndexes)-1]
	}
	builder.document.Lines = append(builder.document.Lines, Line{Spans: joinSpans(linePrefix, rest...), SelectableIndex: -1, ObjectIndex: objectIndex})
}

// joinSpans returns linePrefix followed by the non-empty spans of rest, as a new slice.
func joinSpans(linePrefix []Span, rest ...Span) []Span {
	spans := make([]Span, 0, len(linePrefix)+len(rest))
	spans = append(spans, linePrefix...)
	for _, span := range rest {
		if span.Text != "" {
			spans = append(spans, span)
		}
	}
	return spans
}

// registerIfSelectable registers an object as selectable if it has string mediaType and digest.
func (builder *documentBuilder) registerIfSelectable(object *node, startLine, endLine int) {
	var mediaType, digestValue string
	var size int64
	hasMediaType, hasDigest := false, false
	for _, child := range object.members {
		if child.value.kind != nodeScalar {
			continue
		}
		switch child.key {
		case "mediaType":
			mediaType, hasMediaType = child.value.stringValue, child.value.scalarKind == SpanString
		case "digest":
			digestValue, hasDigest = child.value.stringValue, child.value.scalarKind == SpanString
		case "size":
			size, _ = strconv.ParseInt(child.value.scalarText, 10, 64)
		}
	}
	if !hasMediaType || !hasDigest {
		return
	}
	builder.document.SelectableObjects = append(builder.document.SelectableObjects, SelectableObject{
		Descriptor: domain.Descriptor{MediaType: domain.MediaType(mediaType), Digest: digestValue, Size: size},
		StartLine:  startLine,
		EndLine:    endLine,
	})
}

// assignSelectableIndexes assigns the innermost selectable object to each line.
// Painting from the widest range first lets inner objects overwrite outer ones.
func (builder *documentBuilder) assignSelectableIndexes() {
	objects := builder.document.SelectableObjects
	paintOrder := make([]int, len(objects))
	for index := range paintOrder {
		paintOrder[index] = index
	}
	sort.SliceStable(paintOrder, func(left, right int) bool {
		leftSpan := objects[paintOrder[left]].EndLine - objects[paintOrder[left]].StartLine
		rightSpan := objects[paintOrder[right]].EndLine - objects[paintOrder[right]].StartLine
		return leftSpan > rightSpan
	})
	for _, objectIndex := range paintOrder {
		for line := objects[objectIndex].StartLine; line <= objects[objectIndex].EndLine; line++ {
			builder.document.Lines[line].SelectableIndex = objectIndex
		}
	}
}
