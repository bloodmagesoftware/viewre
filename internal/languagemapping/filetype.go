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
	"path/filepath"
)

func GetLanguageID(filename string) string {
	ext := filepath.Ext(filename)

	switch ext {
	case ".lua":
		return "lua"
	case ".md", ".mdx":
		return "markdown"
	case ".cs":
		return "cs"
	case ".c", ".h":
		return "c"
	case ".cpp", ".hpp", ".cxx", ".hxx", ".cc", ".hh", ".c++", ".h++":
		return "cpp"
	case ".css":
		return "css"
	case ".erb":
		return "erb"
	case ".ejs":
		return "ejs"
	case ".go":
		return "go"
	case ".hs":
		return "haskell"
	case ".html":
		return "html"
	case ".java":
		return "java"
	case ".js", ".jsx":
		return "javascript"
	case ".json", ".json5", ".jsonc":
		return "json"
	case ".ocaml":
		return "ocaml"
	case ".php":
		return "php"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".rb":
		return "ruby"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "typescriptreact"
	case ".editorconfig":
		return "editorconfig"
	default:
		return "plaintext"
	}
}

func GetImplementation(languageID string) (string, bool) {
	switch languageID {
	case "go":
		return "gopls", true
	case "rust":
		return "rust-analyzer", true
	default:
		return "", false
	}
}
