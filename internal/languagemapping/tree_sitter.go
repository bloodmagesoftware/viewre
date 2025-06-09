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

package languagemapping

import (
	tree_sitter_markdown "github.com/tree-sitter-grammars/tree-sitter-markdown/bindings/go"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_cs "github.com/tree-sitter/tree-sitter-c-sharp/bindings/go"
	tree_sitter_c "github.com/tree-sitter/tree-sitter-c/bindings/go"
	tree_sitter_cpp "github.com/tree-sitter/tree-sitter-cpp/bindings/go"
	tree_sitter_css "github.com/tree-sitter/tree-sitter-css/bindings/go"
	tree_sitter_embedded_template "github.com/tree-sitter/tree-sitter-embedded-template/bindings/go"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tree_sitter_haskell "github.com/tree-sitter/tree-sitter-haskell/bindings/go"
	tree_sitter_html "github.com/tree-sitter/tree-sitter-html/bindings/go"
	tree_sitter_java "github.com/tree-sitter/tree-sitter-java/bindings/go"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	tree_sitter_json "github.com/tree-sitter/tree-sitter-json/bindings/go"
	tree_sitter_ocaml "github.com/tree-sitter/tree-sitter-ocaml/bindings/go"
	tree_sitter_php "github.com/tree-sitter/tree-sitter-php/bindings/go"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"
	tree_sitter_ruby "github.com/tree-sitter/tree-sitter-ruby/bindings/go"
	tree_sitter_rust "github.com/tree-sitter/tree-sitter-rust/bindings/go"
	tree_sitter_typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
	tree_sitter_editorconfig "github.com/valdezfomar/tree-sitter-editorconfig/bindings/go"
)

var (
	md           = tree_sitter.NewLanguage(tree_sitter_markdown.Language())
	cs           = tree_sitter.NewLanguage(tree_sitter_cs.Language())
	c            = tree_sitter.NewLanguage(tree_sitter_c.Language())
	cpp          = tree_sitter.NewLanguage(tree_sitter_cpp.Language())
	css          = tree_sitter.NewLanguage(tree_sitter_css.Language())
	erb          = tree_sitter.NewLanguage(tree_sitter_embedded_template.Language())
	golang       = tree_sitter.NewLanguage(tree_sitter_go.Language())
	hs           = tree_sitter.NewLanguage(tree_sitter_haskell.Language())
	html         = tree_sitter.NewLanguage(tree_sitter_html.Language())
	java         = tree_sitter.NewLanguage(tree_sitter_java.Language())
	js           = tree_sitter.NewLanguage(tree_sitter_javascript.Language())
	json         = tree_sitter.NewLanguage(tree_sitter_json.Language())
	ocaml        = tree_sitter.NewLanguage(tree_sitter_ocaml.LanguageOCaml())
	php          = tree_sitter.NewLanguage(tree_sitter_php.LanguagePHP())
	py           = tree_sitter.NewLanguage(tree_sitter_python.Language())
	rs           = tree_sitter.NewLanguage(tree_sitter_rust.Language())
	rb           = tree_sitter.NewLanguage(tree_sitter_ruby.Language())
	ts           = tree_sitter.NewLanguage(tree_sitter_typescript.LanguageTypescript())
	tsx          = tree_sitter.NewLanguage(tree_sitter_typescript.LanguageTSX())
	editorconfig = tree_sitter.NewLanguage(tree_sitter_editorconfig.Language())
)

func GetParser(languageID string) (*tree_sitter.Language, bool) {
	switch languageID {
	case "markdown":
		return md, true
	case "cs":
		return cs, true
	case "c":
		return c, true
	case "cpp":
		return cpp, true
	case "css":
		return css, true
	case "erb", "ejs":
		return erb, true
	case "go":
		return golang, true
	case "haskell":
		return hs, true
	case "html":
		return html, true
	case "java":
		return java, true
	case "javascript":
		return js, true
	case "json":
		return json, true
	case "ocaml":
		return ocaml, true
	case "php":
		return php, true
	case "python":
		return py, true
	case "rust":
		return rs, true
	case "ruby":
		return rb, true
	case "typescript":
		return ts, true
	case "typescriptreact":
		return tsx, true
	case "editorconfig":
		return editorconfig, true
	default:
		return nil, false
	}
}
