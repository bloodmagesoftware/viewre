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
	"fmt"
	"os"
	"strconv"
	"strings"
)

var (
	Address              string
	Origin               string
	Url                  string
	WorkosClientId       string
	WorkosApiKey         string
	WorkosCookiePassword string
	Production           bool
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

	if workosClientId, ok := os.LookupEnv("WORKOS_CLIENT_ID"); ok {
		WorkosClientId = workosClientId
	} else {
		fmt.Fprintf(os.Stderr, "WORKOS_CLIENT_ID is not set")
		os.Exit(1)
	}

	if workosApiKey, ok := os.LookupEnv("WORKOS_API_KEY"); ok {
		WorkosApiKey = workosApiKey
	} else {
		fmt.Fprintf(os.Stderr, "WORKOS_API_KEY is not set")
		os.Exit(1)
	}

	if workosCookiePassword, ok := os.LookupEnv("WORKOS_COOKIE_PASSWORD"); ok {
		WorkosCookiePassword = workosCookiePassword
	} else {
		fmt.Fprintf(os.Stderr, "WORKOS_COOKIE_PASSWORD is not set")
		os.Exit(1)
	}

	if strings.HasPrefix(Origin, "localhost") || strings.HasPrefix(Origin, "127.0.0.1") || strings.HasPrefix(Origin, "host.docker.internal") {
		Url = "http://" + Origin
	} else {
		Url = "https://" + Origin
	}

	Production = strings.HasPrefix(Url, "https://")
}
