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
	"fmt"
	"net/http"
	"viewre/internal/db"
)

func encodeSshKey(key string) []byte {
	if len(key) == 0 {
		return nil
	}
	return []byte(key)
}

func AdminRepoHandler(w http.ResponseWriter, r *http.Request) {
	noCache(w)
	switch r.Method {
	case "DELETE":
		repoName := r.URL.Query().Get("name")
		if repoName == "" {
			http.Error(w, "No repo name provided", http.StatusBadRequest)
			return
		}
		db.Repos.Lock()
		defer db.Repos.Unlock()
		if _, ok := db.Repos.Get(repoName); !ok {
			http.Error(w, "Repo not found", http.StatusNotFound)
			return
		}
		db.Repos.Delete(repoName)
		http.Redirect(w, r, "/admin", http.StatusFound)
	case "POST":
		repo := db.Repo{
			r.FormValue("name"),
			r.FormValue("url"),
			r.FormValue("username"),
			r.FormValue("password"),
			encodeSshKey(r.FormValue("ssh_private_key")),
			r.FormValue("ssh_passphrase"),
		}
		if repo.Name == "" {
			http.Error(w, "No name provided", http.StatusBadRequest)
			return
		}
		if repo.Url == "" {
			http.Error(w, "No url provided", http.StatusBadRequest)
			return
		}
		if repo.Username != "" {
			if repo.Password == "" {
				http.Error(w, "No password provided", http.StatusBadRequest)
				return
			}
			if len(repo.SshPrivateKey) > 0 {
				http.Error(w, "Only one authentication method can be used", http.StatusBadRequest)
				return
			}
		} else if repo.Password != "" {
			http.Error(w, "Password can only be used with username", http.StatusBadRequest)
			return
		}
		db.Repos.Lock()
		defer db.Repos.Unlock()
		db.Repos.Set(repo.Name, &repo)
		http.Redirect(w, r, "/admin", http.StatusFound)
	default:
		http.Error(w, fmt.Sprintf("Method %s not allowed", r.Method), http.StatusMethodNotAllowed)
	}
}
