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

package tree_sitter

import (
	"fmt"
	"html"
	"path/filepath"
	"sort"
	"strings"
	"viewre/internal/languagemapping"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/format/diff"
)

func parse(code []byte, lang string, oldTree *tree_sitter.Tree) (*tree_sitter.Tree, error) {
	parser := tree_sitter.NewParser()
	defer parser.Close()
	if tsLang, ok := languagemapping.GetParser(lang); !ok {
		return nil, fmt.Errorf("unknown language %q", lang)
	} else {
		if err := parser.SetLanguage(tsLang); err != nil {
			return nil, err
		}
	}
	tree := parser.Parse(code, oldTree)
	return tree, nil
}

type nullFile struct{}

// Hash returns the File Hash.
func (nullFile) Hash() plumbing.Hash {
	return plumbing.ZeroHash
}

// Mode returns the FileMode.
func (nullFile) Mode() filemode.FileMode {
	return filemode.FileMode(0)
}

// Path returns the complete Path to the file, including the filename.
func (nullFile) Path() string {
	return ""
}

func Patch(a, b string, filePatch diff.FilePatch) (header string, body string) {
	if filePatch == nil {
		return
	}
	from, to := filePatch.Files()
	if from == nil {
		from = nullFile{}
	}
	if to == nil {
		to = nullFile{}
	}
	headerBuilder := strings.Builder{}
	headerBuilder.WriteString(`<p class="inline font-bold text-white">`)
	headerBuilder.WriteString(html.EscapeString(fmt.Sprintf("diff --git i/%s w/%s", from.Path(), to.Path())))
	headerBuilder.WriteString("</p><p>")
	headerBuilder.WriteString(html.EscapeString(fmt.Sprintf("index %.7s..%.7s %s", from.Hash().String(), to.Hash().String(), from.Mode().String())))
	headerBuilder.WriteString("</p><p>")
	if from.Path() == "" {
		headerBuilder.WriteString(html.EscapeString("--- /dev/null"))
	} else {
		headerBuilder.WriteString(html.EscapeString(fmt.Sprintf("--- i/%s", from.Path())))
	}
	headerBuilder.WriteString("</p><p>")
	if to.Path() == "" {
		headerBuilder.WriteString(html.EscapeString("+++ /dev/null"))
	} else {
		headerBuilder.WriteString(html.EscapeString(fmt.Sprintf("+++ w/%s", to.Path())))
	}
	headerBuilder.WriteString("</p>")
	header = headerBuilder.String()

	fromFullContentBuilder := strings.Builder{}
	toFullContentBuilder := strings.Builder{}
	for _, chunk := range filePatch.Chunks() {
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

	fromCode := []byte(fromFullContentBuilder.String())
	toCode := []byte(toFullContentBuilder.String())

	fromLang := languagemapping.GetLanguageID(filepath.Base(from.Path()))
	toLang := languagemapping.GetLanguageID(filepath.Base(to.Path()))

	fromTree, err := parse(fromCode, fromLang, nil)
	if err != nil {
		fromTree = nil
	}

	fromSegments := renderWithHighlighting(
		fromCode,
		collectSpans(fromTree),
	)

	toTree, err := parse(toCode, toLang, nil)
	if err != nil {
		toTree = nil
	}

	toSegments := renderWithHighlighting(
		toCode,
		collectSpans(toTree),
	)

	fromOffset := uint(0)
	toOffset := uint(0)

	bodyLeftBuilder := strings.Builder{}
	bodyRightBuilder := strings.Builder{}

	leftDiff := 0
	rightDiff := 0

	for _, chunk := range filePatch.Chunks() {
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
			bodyLeftBuilder.WriteString(`<div class="chunk chunk--left chunk--equal">`)
			bodyLeftBuilder.WriteString(render(fromSegments, fromOffset, fromOffset+chunkLength, fromCode))
			bodyLeftBuilder.WriteString(`</div>`)
			bodyRightBuilder.WriteString(`<div class="chunk chunk--equal">`)
			bodyRightBuilder.WriteString(render(toSegments, toOffset, toOffset+chunkLength, toCode))
			bodyRightBuilder.WriteString(`</div>`)
			fromOffset += chunkLength
			toOffset += chunkLength

		case diff.Add:
			bodyRightBuilder.WriteString(`<div class="chunk chunk--add">`)
			bodyRightBuilder.WriteString(render(toSegments, toOffset, toOffset+chunkLength, toCode))
			bodyRightBuilder.WriteString(`</div>`)
			toOffset += chunkLength
			rightDiff = countLineBreaks(chunk.Content())

		case diff.Delete:
			bodyLeftBuilder.WriteString(`<div class="chunk chunk--delete">`)
			bodyLeftBuilder.WriteString(render(fromSegments, fromOffset, fromOffset+chunkLength, fromCode))
			bodyLeftBuilder.WriteString(`</div>`)
			fromOffset += chunkLength
			leftDiff = countLineBreaks(chunk.Content())
		}
	}

	body = fmt.Sprintf(
		`<div class="diff"><div class="diff__left" data-file="%s" data-commit="%s">%s</div><div class="diff__right" data-file="%s" data-commit="%s">%s</div></div>`,
		from.Path(),
		a,
		bodyLeftBuilder.String(),
		to.Path(),
		b,
		bodyRightBuilder.String(),
	)

	return
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
	case "comment", "comment_content", "line_comment", "block_comment", "//", "shebang", "--":
		return "text-neutral-400", true

	case "string", "string_start", "string_end", "string_content", "string_fragment", "string_literal_content", "string_literal", "raw_string_literal", "interpreted_string_literal", "interpreted_string_literal_content", "\"", "'", "`", "fenced_code_block_delimiter", "indented_code_block", "fenced_code_block", "link_title", "attribute_value":
		return "text-green-400", true

	case "link_destination", "link_label":
		return "text-blue-400 underline", true

	case "escape_sequence", "backslash_escape":
		return "text-lime-400", true

	case "block_continuation", "block_quote_marker":
		return "text-emerald-400", true

	case "number", "int", "float", "int_literal", "integer_literal", "float_literal", "rune_literal", "chan", "decimal_integer_literal", "hex_integer_literal", "octal_integer_literal", "binary_integer_literal", "true", "false":
		return "text-amber-400", true

	case "field_identifier", "attribute_name":
		return "text-blue-400", true

	case "_line":
		return "text-pink-400", true

	case "identifier", "tag_name":
		return "text-white", true

	case "type_identifier", "language", "void_type":
		return "text-yellow-400", true
	case "predefined_type":
		return "text-amber-400", true

	case "export":
		return "text-cyan-400", true

	case "import", "from", "as", "require", "package", "class", "def", "interface", "enum", "type", "function", "fn", "fun", "func", "go", "var", "let", "const", "async", "await", "break", "case", "catch", "continue", "debugger", "default", "delete", "do", "else", "finally", "for", "if", "in", "instanceof", "new", "return", "switch", "this", "throw", "try", "typeof", "void", "while", "with", "yield", "private", "public", "protected", "internal", "pub", "use", "mod", "mut", "satisfies", "override", "readonly", "namespace", "keyof", "implements", "abstract", "declare", "using", "static", "except", "local", "then", "end", "elseif":
		return "text-indigo-400", true

	case "assembly", "get", "set":
		return "text-purple-400", true

	case "operator", ":=", "=", "+", "-", "~", "*", "/", "%", "==", "!=", "===", "!==", "=>", "==>", "<-", "->", "<<", ">>", "<", ">", "/>", "</", "<=", ">=", "&&", "||", "!", "|", "&", "$", "@":
		return "text-cyan-400", true

	case "#if", "#else", "#elif", "#endif", "#ifdef", "#ifndef", "#include":
		return "text-rose-400", true

	case "list_marker_plus", "list_marker_minus", "list_marker_star", "list_marker_dot", "list_marker_parenthesis", "thematic_break":
		return "text-red-300", true

	case "punctuation", "(", ")", "[", "]", "{", "}", ";", "?", ":", ",", ".", "...", "..", "::", "#", "atx_h1_marker", "atx_h2_marker", "atx_h3_marker", "atx_h4_marker", "atx_h5_marker", "atx_h6_marker", "setext_h1_underline", "setext_h2_underline":
		return "text-gray-400", true

	default:
		return "text-white", false
	}
}

func getRainbowBracketClass(index int) string {
	switch index % 11 {
	case 0:
		return "text-yellow-300"
	case 1:
		return "text-green-300"
	case 2:
		return "text-cyan-300"
	case 3:
		return "text-violet-300"
	case 4:
		return "text-orange-300"
	case 5:
		return "text-lime-300"
	case 6:
		return "text-blue-300"
	case 7:
		return "text-red-300"
	case 8:
		return "text-teal-300"
	case 9:
		return "text-fuchsia-300"
	case 10:
		return "text-lime-300"
	}
	return "text-gray-300"
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
				class, _ = getNodeClass(n.GrammarName())
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

	rainbowBrackets := 0

	for i, span := range spans {
		switch span.kind {
		case "(", "[", "{":
			span.class = getRainbowBracketClass(rainbowBrackets)
			spans[i] = span
			rainbowBrackets++
		case ")", "]", "}":
			rainbowBrackets--
			span.class = getRainbowBracketClass(rainbowBrackets)
			spans[i] = span
		}
	}

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
		if segment.end < windowStart || segment.start > windowEnd {
			continue
		}
		s := max(segment.start, windowStart)
		e := min(segment.end, windowEnd)
		slice := code[s:e]

		chunkBuilder.WriteString(fmt.Sprintf(
			`<span class="%s" data-start="%d" data-end="%d" data-kind="%s" data-grammarname="%s">%s</span>`,
			segment.class,
			s,
			e,
			segment.kind,
			segment.grammarname,
			html.EscapeString(string(slice)),
		))
	}

	return chunkBuilder.String()
}
