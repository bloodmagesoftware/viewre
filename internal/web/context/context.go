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
	"encoding/base64"
	"fmt"
	"html"
	"net/http"
	"time"
	"viewre/internal/web/api"

	"github.com/workos/workos-go/v4/pkg/usermanagement"
)

func FromRequest(r *http.Request) MyContext {
	currentCtx := r.Context()
	if currentCtx == nil {
		currentCtx = context.Background()
	} else if myCtx, ok := currentCtx.(MyContext); ok {
		return myCtx
	}
	user, ok := api.GetAuthenticatedUser(r)
	ctx := MyContext{
		ctx:      currentCtx,
		request:  r,
		user:     &user,
		LoggedIn: ok,
	}
	return ctx
}

type MyContext struct {
	ctx      context.Context
	request  *http.Request
	user     *usermanagement.User
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
		case "sub", "id":
			return ctx.user.ID
		case "name":
			return ctx.user.FirstName + " " + ctx.user.LastName
		case "email":
			return ctx.user.Email
		case "picture":
			if len(ctx.user.ProfilePictureURL) > 0 {
				return ctx.user.ProfilePictureURL
			}
			return "data:image/svg+xml;base64," +
				base64.StdEncoding.EncodeToString([]byte(
					fmt.Sprintf(
						`<svg width="64" height="64" viewBox="0 0 64 64" fill="none" xmlns="http://www.w3.org/2000/svg"><g clip-path="url(#clip0_605_2)"><rect width="64" height="64" fill="#2A233E"/><text fill="white" xml:space="preserve" style="white-space: pre" font-family="sans-serif" font-size="32" letter-spacing="0em"><tspan x="8.375" y="43.6364">%s</tspan></text></g><defs><clipPath id="clip0_605_2"><rect width="64" height="64" fill="white"/></clipPath></defs></svg>`,
						html.EscapeString(initials(ctx.user)),
					),
				))
		case "email_verified":
			return ctx.user.EmailVerified
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

func initials(user *usermanagement.User) string {
	if len(user.FirstName) > 0 {
		if len(user.LastName) > 0 {
			return user.FirstName[0:1] + user.LastName[0:1]
		} else {
			return user.FirstName[0:2]
		}
	} else {
		if len(user.LastName) > 0 {
			return user.LastName[0:2]
		} else {
			return "??"
		}
	}
}
