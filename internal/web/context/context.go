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

package context

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
	"viewre/internal/db"
	"viewre/internal/web/auth"
)

func FromRequest(r *http.Request) MyContext {
	currentCtx := r.Context()
	if currentCtx == nil {
		currentCtx = context.Background()
	} else if myCtx, ok := currentCtx.(MyContext); ok {
		return myCtx
	}
	ctx := MyContext{
		ctx:     currentCtx,
		request: r,
	}
	if s, err := auth.Store.Get(r, "auth-session"); err == nil {
		userInfoJson, ok := s.Values["user_info"].(string)
		if ok {
			_ = json.NewDecoder(strings.NewReader(userInfoJson)).Decode(&ctx.UserInfo)
			ctx.LoggedIn = true
		}
	}
	return ctx
}

type MyContext struct {
	UserInfo auth.UserInfo
	ctx      context.Context
	request  *http.Request
	LoggedIn bool
}

func (ctx MyContext) Deadline() (deadline time.Time, ok bool) {
	return ctx.ctx.Deadline()
}

func (ctx MyContext) Done() <-chan struct{} {
	return ctx.ctx.Done()
}

func (ctx MyContext) Err() error {
	return ctx.ctx.Err()
}

func (ctx MyContext) Value(key any) any {
	if keyStr, ok := key.(string); ok {
		switch keyStr {
		case "sub":
			return ctx.UserInfo.Sub
		case "name":
			return ctx.UserInfo.GetName()
		case "email":
			return ctx.UserInfo.Email
		case "picture":
			return ctx.UserInfo.Picture
		case "email_verified":
			return ctx.UserInfo.EmailVerified
		case "active":
			db.Users.RLock()
			defer db.Users.RUnlock()
			if user, ok := db.Users.Get(ctx.UserInfo.Sub); ok {
				return user.Active
			}
			return false
		case "logged_in":
			return ctx.LoggedIn
		default:
			val := ctx.request.PathValue(keyStr)
			if val != "" {
				return val
			}
		}
	}

	return ctx.ctx.Value(key)
}
