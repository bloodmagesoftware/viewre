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
	"embed"
	"fmt"

	"github.com/a-h/templ"
)

//go:embed *.min.css
var StaticFiles embed.FS

func staticUrl(path string) string {
	return "/_static/" + path
}

func fmtUrl(format string, args ...any) templ.SafeURL {
	return templ.URL(fmt.Sprintf(format, args...))
}
