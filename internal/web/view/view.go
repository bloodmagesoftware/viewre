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
	"crypto/sha1"
	"embed"
	"encoding/base64"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	unixpath "path"
	"path/filepath"
	"strings"

	"github.com/a-h/templ"
)

var (
	//go:embed *.min.css *.ttf *.js
	staticFiles         embed.FS
	staticFileNameToUrl map[string]string
	staticFileUrlToName map[string]string
)

func init() {
	staticFileNameToUrl = make(map[string]string)
	staticFileUrlToName = make(map[string]string)

	if err := fs.WalkDir(staticFiles, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		f, _ := staticFiles.Open(name)
		defer f.Close()
		b, _ := io.ReadAll(f)
		hash := sha1.Sum(b)
		hashStr := base64.RawURLEncoding.EncodeToString(hash[:])
		staticFileName := filenameWithHash(name, hashStr)
		staticFileNameToUrl[name] = staticFileName
		staticFileUrlToName[staticFileName] = name
		return nil
	}); err != nil {
		panic(err)
	}
}

func StaticFileHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/_static/")
	log.Println("StaticFileHandler", path)
	if name, ok := staticFileUrlToName[path]; ok {
		f, err := staticFiles.Open(name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer f.Close()
		w.Header().Set("Content-Type", contentTypeFromFilename(name))
		w.Header().Set("Cache-Control", "public, max-age=2592000") // 30 days
		_, _ = io.Copy(w, f)
	} else {
		f, err := staticFiles.Open(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		defer f.Close()
		w.Header().Set("Content-Type", contentTypeFromFilename(path))
		w.Header().Set("Cache-Control", "public, max-age=300") // 5 minutes
		_, _ = io.Copy(w, f)
	}
}

func contentTypeFromFilename(name string) string {
	switch ext := unixpath.Ext(name); ext {
	case ".js":
		return "application/javascript"
	case ".css":
		return "text/css"
	case ".woff2":
		return "font/woff2"
	case ".map":
		return "application/json"
	default:
		return "text/plain"
	}
}

func filenameWithHash(name string, hash string) string {
	dir := unixpath.Dir(name)
	base := unixpath.Base(name)
	ext := unixpath.Ext(name)
	withoutExt := base[:len(base)-len(ext)]
	return filepath.Join(dir, withoutExt+"."+hash+ext)
}

func staticUrl(path string) string {
	if url, ok := staticFileNameToUrl[path]; ok {
		return unixpath.Join("/_static/", url)
	}
	panic(fmt.Sprintf("static file not found: %s", path))
}

func fmtUrl(format string, args ...any) templ.SafeURL {
	return templ.URL(fmt.Sprintf(format, args...))
}
