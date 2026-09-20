// Package jsonview はJSONを人間に見やすい行単位の表示モデルへ変換する。
// キー順は元のJSONのまま保持し、mediaType と digest を持つオブジェクトを選択対象として検出する。
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

// SpanKind は色分けのための文字列の種別。
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

// Line は表示1行分。SelectableIndex は行が属する最内の選択可能オブジェクトの番号(なければ -1)。
type Line struct {
	Spans           []Span
	SelectableIndex int
}

// SelectableObject は mediaType と digest を持つオブジェクト。行範囲は両端を含む。
type SelectableObject struct {
	Descriptor domain.Descriptor
	StartLine  int
	EndLine    int
}

type Document struct {
	Lines             []Line
	SelectableObjects []SelectableObject
}

// SelectableObjectAt は line 行目が属する最内の選択可能オブジェクトを返す。
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

// Parse は raw JSON を整形して Document にする。
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
	scalarText  string // JSON表現(文字列は引用符付き)
	scalarKind  SpanKind
	stringValue string // scalarKind が SpanString のときの引用符なしの値
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
}

// render は node を lines へ追加する。linePrefix は最初の行の先頭に付くインデントとキー。
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
	spans := make([]Span, 0, len(linePrefix)+len(rest))
	spans = append(spans, linePrefix...)
	for _, span := range rest {
		if span.Text != "" {
			spans = append(spans, span)
		}
	}
	builder.document.Lines = append(builder.document.Lines, Line{Spans: spans, SelectableIndex: -1})
}

// registerIfSelectable は mediaType と digest が文字列で存在するオブジェクトを選択対象として登録する。
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

// assignSelectableIndexes は各行に最内の選択可能オブジェクトを割り当てる。
// 範囲の広い順に塗ることで、入れ子の内側が外側を上書きする。
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
