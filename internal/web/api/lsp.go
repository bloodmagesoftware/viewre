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

package api

import (
	"encoding/base64"
	"html"
	"net/http"
	"strconv"
	"sync"
	"viewre/internal/db"
	"viewre/internal/lsp"
	"viewre/internal/repository"

	"github.com/gomarkdown/markdown"
	gomdhtml "github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

var lspApiMutex = &sync.Mutex{}

func LspHoverHandler(w http.ResponseWriter, r *http.Request) {
	lspApiMutex.Lock()
	defer lspApiMutex.Unlock()

	db.Repos.RLock()
	defer db.Repos.RUnlock()

	repo := r.PathValue("repo")
	commit := r.PathValue("commit")
	fileB64 := r.PathValue("file")
	fileB, err := base64.URLEncoding.DecodeString(fileB64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	file := string(fileB)
	indexStr := r.PathValue("index")
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbRepo, ok := db.Repos.Get(repo)
	if !ok {
		http.Error(w, "repo not found", http.StatusNotFound)
		return
	}

	projectDir, err := repository.CheckoutCommit(r.Context(), dbRepo, commit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client, err := lsp.GetServer("gopls", projectDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	hover, err := client.HoverByteIndex(file, index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var hoverHtml []byte
	switch hover.ContentType {
	case "markdown":
		hoverHtml = mdToHTML([]byte(hover.Content))
	case "plaintext":
		hoverHtml = []byte(html.EscapeString(hover.Content))
	}

	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(hoverHtml)
}

func mdToHTML(md []byte) []byte {
	// create markdown parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	// create HTML renderer with extensions
	htmlFlags := gomdhtml.CommonFlags | gomdhtml.HrefTargetBlank
	opts := gomdhtml.RendererOptions{Flags: htmlFlags}
	renderer := gomdhtml.NewRenderer(opts)

	return markdown.Render(doc, renderer)
}
