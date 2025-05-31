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
	"context"
	"encoding/json"
	"net/http"
	"viewre/internal/db"
	"viewre/internal/web/auth"
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Has("target") {
		state := q.Get("target")
		authURL := auth.OAuth2Config.AuthCodeURL(state)
		http.Redirect(w, r, authURL, http.StatusFound)
		return
	}
	state := "/profile"
	authURL := auth.OAuth2Config.AuthCodeURL(state)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func CallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	token, err := auth.OAuth2Config.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rawIdToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "No id_token field in oauth2 token", http.StatusInternalServerError)
		return
	}

	// Get the user info
	idToken, err := auth.Verifier.Verify(r.Context(), rawIdToken)
	if err != nil {
		http.Error(w, "Failed to verify ID Token", http.StatusInternalServerError)
		return
	}

	var userInfo auth.UserInfo
	if err := idToken.Claims(&userInfo); err != nil {
		http.Error(w, "Failed to parse ID Token claims", http.StatusInternalServerError)
		return
	}

	// Create a session
	session, _ := auth.Store.Get(r, "auth-session")
	session.Values["id_token"] = rawIdToken
	if userInfoJson, err := json.Marshal(userInfo); err != nil {
		http.Error(w, "Failed to marshal user info", http.StatusInternalServerError)
		return
	} else {
		session.Values["user_info"] = string(userInfoJson)
	}
	if err := session.Save(r, w); err != nil {
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	db.Users.Lock()
	defer db.Users.Unlock()
	if !db.Users.Has(userInfo.Sub) {
		db.Users.Set(
			userInfo.Sub,
			&db.User{Sub: userInfo.Sub, Name: userInfo.GetName(), Email: userInfo.Email, Active: false},
		)
	}

	q := r.URL.Query()
	if q.Has("state") {
		state := q.Get("state")
		http.Redirect(w, r, state, http.StatusFound)
	} else {
		http.Redirect(w, r, "/", http.StatusFound)
	}
}
