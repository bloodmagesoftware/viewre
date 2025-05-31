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

package config

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var (
	Address           string
	Origin            string
	Url               string
	SessionSecret     []byte
	Auth0Domain       string
	Auth0ClientID     string
	Auth0ClientSecret string
)

func loadEnv() {
	if _, err := os.Stat(".env"); err == nil {
		if f, err := os.Open(".env"); err == nil {
			defer f.Close()
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) != 2 {
					fmt.Fprintf(os.Stderr, "Error parsing .env line: %v\n", line)
					os.Exit(1)
				}
				_ = os.Setenv(parts[0], parts[1])
			}
		}
	}
}

func init() {
	loadEnv()

	if portStr, ok := os.LookupEnv("PORT"); ok {
		if portInt, err := strconv.Atoi(portStr); err == nil {
			Address = ":" + strconv.Itoa(portInt)
		} else {
			fmt.Fprintf(os.Stderr, "Error parsing PORT: %v\n", err)
			os.Exit(1)
		}
	}

	if originStr, ok := os.LookupEnv("ORIGIN"); ok {
		Origin = originStr
	} else {
		fmt.Fprintf(os.Stderr, "Error: ORIGIN not set\n")
		os.Exit(1)
	}

	if sessionSecretStr, ok := os.LookupEnv("SESSION_SECRET"); ok {
		if b64Std, err := base64.StdEncoding.DecodeString(sessionSecretStr); err == nil {
			SessionSecret = b64Std
		} else if b64Url, err := base64.URLEncoding.DecodeString(sessionSecretStr); err == nil {
			SessionSecret = b64Url
		} else {
			SessionSecret = []byte(sessionSecretStr)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Error: SESSION_SECRET not set\n")
		os.Exit(1)
	}

	if auth0DomainStr, ok := os.LookupEnv("AUTH0_DOMAIN"); ok {
		Auth0Domain = auth0DomainStr
	} else {
		fmt.Fprintf(os.Stderr, "Error: AUTH0_DOMAIN not set\n")
		os.Exit(1)
	}

	if auth0ClientIDStr, ok := os.LookupEnv("AUTH0_CLIENT_ID"); ok {
		Auth0ClientID = auth0ClientIDStr
	} else {
		fmt.Fprintf(os.Stderr, "Error: AUTH0_CLIENT_ID not set\n")
		os.Exit(1)
	}

	if auth0ClientSecretStr, ok := os.LookupEnv("AUTH0_CLIENT_SECRET"); ok {
		Auth0ClientSecret = auth0ClientSecretStr
	} else {
		fmt.Fprintf(os.Stderr, "Error: AUTH0_CLIENT_SECRET not set\n")
		os.Exit(1)
	}

	if strings.HasPrefix(Origin, "localhost") {
		Url = "http://" + Origin
	} else {
		Url = "https://" + Origin
	}
}
