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

package auth

import (
	"context"
	"fmt"
	"os"
	"viewre/internal/config"

	"github.com/auth0/go-auth0/authentication"
	"github.com/coreos/go-oidc/v3/oidc"
	gsessions "github.com/gorilla/sessions"
	"golang.org/x/oauth2"
)

type UserInfo struct {
	Sub           string `json:"sub"`
	Name          string `json:"name"`
	Nickname      string `json:"nickname"`
	Username      string `json:"username"`
	Email         string `json:"email"`
	Picture       string `json:"picture"`
	EmailVerified bool   `json:"email_verified"`
}

func (u UserInfo) GetName() string {
	if u.Nickname != "" {
		return u.Nickname
	}
	if u.Username != "" {
		return u.Username
	}
	if u.Name != "" {
		return u.Name
	}
	if u.Email != "" {
		return u.Email
	}
	return u.Sub
}

var (
	Store        = gsessions.NewCookieStore(config.SessionSecret)
	OAuth2Config oauth2.Config
	Verifier     *oidc.IDTokenVerifier
	auth0API     *authentication.Authentication
)

func init() {
	provider, err := oidc.NewProvider(context.Background(), "https://"+config.Auth0Domain+"/")
	if err != nil {
		panic(err)
	}

	OAuth2Config = oauth2.Config{
		ClientID:     config.Auth0ClientID,
		ClientSecret: config.Auth0ClientSecret,
		RedirectURL:  config.Url + "/auth/callback",
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	Verifier = provider.Verifier(&oidc.Config{ClientID: config.Auth0ClientID})

	auth0API, err = authentication.New(
		context.Background(),
		config.Auth0Domain,
		authentication.WithClientID(config.Auth0ClientID),
		authentication.WithClientSecret(config.Auth0ClientSecret), // Optional depending on the grants used
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating auth0 client: %v\n", err)
		os.Exit(1)
		return
	}
}
