// ViewRe is a web-based code review tool.
// Copyright (C) 2025  Frank Mayer
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package view

import (
	"fmt"
	"html"
	"path/filepath"
	"sort"
	"strings"

	tree_sitter_markdown "github.com/tree-sitter-grammars/tree-sitter-markdown/bindings/go"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	tree_sitter_rust "github.com/tree-sitter/tree-sitter-rust/bindings/go"
	tree_sitter_typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"

	"github.com/go-git/go-git/v5/plumbing/format/diff"
)

var languages = map[string]*tree_sitter.Language{
	"js":  tree_sitter.NewLanguage(tree_sitter_javascript.Language()),
	"ts":  tree_sitter.NewLanguage(tree_sitter_typescript.LanguageTypescript()),
	"tsx": tree_sitter.NewLanguage(tree_sitter_typescript.LanguageTSX()),
	"go":  tree_sitter.NewLanguage(tree_sitter_go.Language()),
	"rs":  tree_sitter.NewLanguage(tree_sitter_rust.Language()),
	"md":  tree_sitter.NewLanguage(tree_sitter_markdown.Language()),
}

func parse(code string, lang string, oldTree *tree_sitter.Tree) (*tree_sitter.Tree, error) {
	parser := tree_sitter.NewParser()
	defer parser.Close()
	if tsLang, ok := languages[lang]; !ok {
		return nil, fmt.Errorf("unknown language %q", lang)
	} else {
		if err := parser.SetLanguage(tsLang); err != nil {
			return nil, err
		}
	}
	tree := parser.Parse([]byte(code), oldTree)
	return tree, nil
}

func Patch(patch diff.FilePatch) (header string, body string) {
	from, to := patch.Files()
	headerBuilder := strings.Builder{}
	headerBuilder.WriteString(`<p class="inline-block font-bold text-white">`)
	headerBuilder.WriteString(html.EscapeString(fmt.Sprintf("diff --git i/%s w/%s", from.Path(), to.Path())))
	headerBuilder.WriteString("</p><p>")
	headerBuilder.WriteString(html.EscapeString(fmt.Sprintf("index %.7s..%.7s %s", from.Hash().String(), to.Hash().String(), from.Mode().String())))
	headerBuilder.WriteString("</p><p>")
	headerBuilder.WriteString(html.EscapeString(fmt.Sprintf("--- i/%s", from.Path())))
	headerBuilder.WriteString("</p><p>")
	headerBuilder.WriteString(html.EscapeString(fmt.Sprintf("+++ w/%s", to.Path())))
	headerBuilder.WriteString("</p>")
	header = headerBuilder.String()

	fromFullContentBuilder := strings.Builder{}
	toFullContentBuilder := strings.Builder{}
	for _, chunk := range patch.Chunks() {
		switch chunk.Type() {
		case diff.Equal:
			fromFullContentBuilder.WriteString(chunk.Content())
			toFullContentBuilder.WriteString(chunk.Content())
		case diff.Add:
			toFullContentBuilder.WriteString(chunk.Content())
		case diff.Delete:
			fromFullContentBuilder.WriteString(chunk.Content())
		}
	}
	fromFullContent := fromFullContentBuilder.String()
	toFullContent := toFullContentBuilder.String()

	fromExt := filepath.Ext(from.Path())
	toExt := filepath.Ext(to.Path())

	fromTree, err := parse(fromFullContent, extToLang(fromExt), nil)
	if err != nil {
		fromTree = nil
	}

	fromSegments := renderWithHighlighting(
		[]byte(fromFullContent),
		collectSpans(fromTree),
	)

	toTree, err := parse(toFullContent, extToLang(toExt), nil)
	if err != nil {
		toTree = nil
	}

	toSegments := renderWithHighlighting(
		[]byte(toFullContent),
		collectSpans(toTree),
	)

	fromOffset := uint(0)
	toOffset := uint(0)

	bodyLeftBuilder := strings.Builder{}
	bodyRightBuilder := strings.Builder{}

	leftDiff := 0
	rightDiff := 0

	for _, chunk := range patch.Chunks() {
		chunkLength := uint(len([]byte(chunk.Content())))

		switch chunk.Type() {
		case diff.Equal:
			if spacingLeft := rightDiff - leftDiff; spacingLeft > 0 {
				bodyLeftBuilder.WriteString(fmt.Sprintf(`<div class="chunk chunk--space">%s</div>`, strings.Repeat("<br>", spacingLeft)))
			}
			if spacingRight := leftDiff - rightDiff; spacingRight > 0 {
				bodyRightBuilder.WriteString(fmt.Sprintf(`<div class="chunk chunk--space">%s</div>`, strings.Repeat("<br>", spacingRight)))
			}
			leftDiff = 0
			rightDiff = 0
			rendered := render(toSegments, toOffset, toOffset+chunkLength, []byte(toFullContent))
			bodyLeftBuilder.WriteString(
				fmt.Sprintf(
					`<div data-start="%d" data-end="%d" class="chunk chunk--left chunk--equal">`,
					toOffset,
					toOffset+chunkLength,
				),
			)
			bodyLeftBuilder.WriteString(rendered)
			bodyLeftBuilder.WriteString(`</div>`)
			bodyRightBuilder.WriteString(
				fmt.Sprintf(
					`<div data-start="%d" data-end="%d" class="chunk chunk--equal">`,
					toOffset,
					toOffset+chunkLength,
				),
			)
			bodyRightBuilder.WriteString(rendered)
			bodyRightBuilder.WriteString(`</div>`)
			fromOffset += chunkLength
			toOffset += chunkLength

		case diff.Add:
			bodyRightBuilder.WriteString(
				fmt.Sprintf(
					`<div data-start="%d" data-end="%d" class="chunk chunk--add">`,
					toOffset,
					toOffset+chunkLength,
				),
			)
			bodyRightBuilder.WriteString(render(toSegments, toOffset, toOffset+chunkLength, []byte(toFullContent)))
			bodyRightBuilder.WriteString(`</div>`)
			toOffset += chunkLength
			rightDiff = countLineBreaks(chunk.Content())

		case diff.Delete:
			bodyLeftBuilder.WriteString(
				fmt.Sprintf(
					`<div data-start="%d" data-end="%d" class="chunk chunk--delete">`,
					toOffset,
					toOffset+chunkLength,
				),
			)
			bodyLeftBuilder.WriteString(render(fromSegments, fromOffset, fromOffset+chunkLength, []byte(fromFullContent)))
			bodyLeftBuilder.WriteString(`</div>`)
			fromOffset += chunkLength
			leftDiff = countLineBreaks(chunk.Content())
		}
	}

	body = fmt.Sprintf(`<div class="diff"><div class="diff__left">%s</div><div class="diff__right">%s</div></div>`, bodyLeftBuilder.String(), bodyRightBuilder.String())

	return
}

func extToLang(ext string) string {
	lang := strings.TrimPrefix(ext, ".")
	if lang == "jsx" {
		return "js"
	}
	return lang
}

type syntaxSpan struct {
	start       uint
	end         uint
	class       string
	kind        string
	grammarname string
}

func getNodeClass(nodeType string) (string, bool) {
	switch nodeType {
	case "comment", "line_comment", "block_comment":
		return "text-neutral-400", true

	case "string", "string_fragment", "string_literal", "raw_string_literal", "interpreted_string_literal", "interpreted_string_literal_content", "\"", "'", "`", "fenced_code_block_delimiter":
		return "text-green-400", true

	case "number", "int_literal", "float_literal", "rune_literal", "chan":
		return "text-amber-400", true

	case "field_identifier":
		return "text-blue-400", true

	case "_line":
		return "text-pink-400", true

	case "identifier":
		return "text-white", true

	case "type_identifier", "language":
		return "text-yellow-400", true

	case "export":
		return "text-cyan-400", true

	case "import", "from", "as", "require", "package", "class", "interface", "enum", "type", "function", "fn", "fun", "func", "go", "var", "let", "const", "async", "await", "break", "case", "catch", "continue", "debugger", "default", "delete", "do", "else", "finally", "for", "if", "in", "instanceof", "new", "return", "switch", "this", "throw", "try", "typeof", "void", "while", "with", "yield":
		return "text-indigo-400", true

	case "operator", ":=", "=", "+", "-", "*", "/", "%", "==", "!=", "===", "!==", "=>", "==>", "<-", "->", "<<", ">>", "<", ">", "<=", ">=", "&&", "||", "!", "|", "&", "$":
		return "text-cyan-400", true

	case "punctuation", "{", "}", "(", ")", "[", "]", ";", "?", ":", ",", ".", "atx_h1_marker", "atx_h2_marker", "atx_h3_marker", "atx_h4_marker", "atx_h5_marker", "atx_h6_marker":
		return "text-gray-400", true

	default:
		return "text-white", false
	}
}

func collectSpans(tree *tree_sitter.Tree) []syntaxSpan {
	if tree == nil {
		return nil
	}

	var spans []syntaxSpan

	var traverse func(*tree_sitter.Node)
	traverse = func(n *tree_sitter.Node) {
		if n.ChildCount() == 0 {
			class, ok := getNodeClass(n.Kind())
			if !ok {
				class, ok = getNodeClass(n.GrammarName())
			}
			class += " ts-node"
			spans = append(spans, syntaxSpan{
				start:       n.StartByte(),
				end:         n.EndByte(),
				class:       class,
				grammarname: n.GrammarName(),
				kind:        n.Kind(),
			})
		} else {
			for i := uint(0); i < n.ChildCount(); i++ {
				child := n.Child(uint(i))
				traverse(child)
			}
		}
	}

	traverse(tree.RootNode())

	sort.Slice(spans, func(i, j int) bool {
		return spans[i].start < spans[j].start
	})

	return spans
}

func renderWithHighlighting(code []byte, spans []syntaxSpan) []highlightedSegment {
	if len(spans) == 0 {
		return []highlightedSegment{
			{
				text:  string(code),
				class: "text-white",
				start: 0,
				end:   uint(len(code)),
			},
		}
	}

	var segments []highlightedSegment
	pos := uint(0)

	for _, span := range spans {
		if pos < span.start {
			segments = append(segments, highlightedSegment{
				text:  string(code[pos:span.start]),
				class: "text-white",
				start: pos,
				end:   span.start,
			})
		}
		segments = append(segments, highlightedSegment{
			text:        string(code[span.start:span.end]),
			class:       span.class,
			start:       span.start,
			end:         span.end,
			kind:        span.kind,
			grammarname: span.grammarname,
		})
		pos = span.end
	}

	if pos < uint(len(code)) {
		end := uint(len(code))
		segments = append(segments, highlightedSegment{
			text:  string(code[pos:]),
			class: "text-white",
			start: pos,
			end:   end,
		})
	}

	return segments
}

type highlightedSegment struct {
	text        string
	class       string
	start       uint
	end         uint
	kind        string
	grammarname string
}

func countLineBreaks(s string) int {
	return strings.Count(s, "\n")
}

func render(segments []highlightedSegment, windowStart uint, windowEnd uint, code []byte) string {
	chunkBuilder := strings.Builder{}

	for _, segment := range segments {
		// skip completely out of this chunk
		if segment.end < windowStart || segment.start > windowEnd {
			continue
		}
		// clip to [start,end]
		s := max(segment.start, windowStart)
		e := min(segment.end, windowEnd)
		slice := code[s:e]

		chunkBuilder.WriteString(fmt.Sprintf(`<span class="%s" data-start="%d" data-end="%d" data-kind="%s" data-grammarname="%s">%s</span>`, segment.class, s, e, html.EscapeString(segment.kind), html.EscapeString(segment.grammarname), html.EscapeString(string(slice))))
	}

	return chunkBuilder.String()
}
