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

package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
	"viewre/internal/config"
	"viewre/internal/web"

	"github.com/go-webauthn/webauthn/webauthn"
)

var (
	listener *net.TCPListener
	server   *http.Server
	WebAuthn *webauthn.WebAuthn
)

func Start() error {
	ln, err := net.Listen("tcp", config.Address)
	if err != nil {
		return errors.Join(fmt.Errorf("listening to %s", config.Address), err)
	}
	listener = ln.(*net.TCPListener)

	server = &http.Server{
		Addr:    ln.Addr().String(),
		Handler: web.NewServer(),
	}

	if err := server.Serve(ln); err != nil {
		return errors.Join(fmt.Errorf("serving %s", server.Addr), err)
	}

	if webAuthn, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "ViewRe",
		RPID:          config.Origin,
		RPOrigins:     []string{config.Url},
	}); err != nil {
		return errors.Join(fmt.Errorf("creating webauthn"), err)
	} else {
		WebAuthn = webAuthn
	}

	return nil
}

func Stop() {
	if server == nil {
		if listener == nil {
			return
		} else {
			_ = listener.Close()
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}
