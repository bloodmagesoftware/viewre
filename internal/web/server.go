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

package web

import (
	"net/http"
	"viewre/internal/web/api"
	"viewre/internal/web/context"
	"viewre/internal/web/view"

	"github.com/a-h/templ"
)

func NewServer() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/_static/", view.StaticFileHandler)
	mux.HandleFunc("/", IndexTemplHandler(view.Index()))
	mux.HandleFunc("/compare/{repo}/{a}/{b}", RequireActiveLogin(TemplHandler(view.Compare())))
	mux.HandleFunc("/profile", RequireLogin(TemplHandler(view.Profile())))
	mux.HandleFunc("/admin", RequireActiveLogin(TemplHandler(view.Admin())))
	mux.HandleFunc("/api/login", api.LoginHandler)
	mux.HandleFunc("/api/login_callback", api.LoginCallbackHandler)
	mux.HandleFunc("/api/logout", RequireActiveLogin(api.LogoutHandler))
	mux.HandleFunc("/api/repo", RequireActiveLogin(api.AdminRepoHandler))
	mux.HandleFunc("/api/lsp/hover/{repo}/{commit}/{file}/{index}", api.LspHoverHandler)
	return mux
}

func RequireActiveLogin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.FromRequest(r)
		// check login
		if !ctx.LoggedIn {
			http.Redirect(w, r, "/api/login", http.StatusFound)
			return
		}
		next(w, r.WithContext(ctx))
	}
}

func RequireLogin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.FromRequest(r)
		if !ctx.LoggedIn {
			http.Redirect(w, r, "/api/login", http.StatusFound)
			return
		}
		next(w, r.WithContext(ctx))
	}
}

func TemplHandler(templComponent templ.Component) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		ctx := context.FromRequest(r)
		if err := templComponent.Render(ctx, w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// IndexTemplHandler is a handler for the / route because http.ServeMux can't do basic routing
func IndexTemplHandler(templComponent templ.Component) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		ctx := context.FromRequest(r)
		if err := templComponent.Render(ctx, w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
