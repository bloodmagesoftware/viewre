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

package db

import (
	"github.com/bloodmagesoftware/speicher"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

var Repos, _ = speicher.LoadMap[*Repo]("data/repos.json")

type Repo struct {
	Name          string `json:"name"`
	Url           string `json:"url"`
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	SshPrivateKey []byte `json:"ssh_private_key,omitempty"`
	SshPassphrase string `json:"ssh_passphrase,omitempty"`
}

func (r *Repo) Auth() transport.AuthMethod {
	if r.Username != "" && r.Password != "" {
		return &http.BasicAuth{
			Username: r.Username,
			Password: r.Password,
		}
	}
	if len(r.SshPrivateKey) > 0 {
		authMethod, err := ssh.NewPublicKeys("git", r.SshPrivateKey, r.SshPassphrase)
		if err != nil {
			return nil
		}
		return authMethod
	}
	return nil
}
