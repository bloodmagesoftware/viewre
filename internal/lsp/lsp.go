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

package lsp

import (
	"fmt"
	"log"
	"sync"
	"time"
	"viewre/internal/languagemapping"
)

type ServerRegister struct {
	LastUsed time.Time
	server   LanguageServer
	rootDir  string
	language string
}

var runningServers = make(map[string]ServerRegister)

var mutex = &sync.Mutex{}

func init() {
	go func() {
		for {
			time.Sleep(10 * time.Second)
			stopIdleServers()
		}
	}()
}

func GetServer(language string, rootDir string) (LanguageServer, error) {
	mutex.Lock()
	defer mutex.Unlock()
	key := language + "@" + rootDir
	log.Println("looking for running server", key)
	if reg, ok := runningServers[key]; ok {
		log.Println("found running server", key)
		reg.LastUsed = time.Now()
		runningServers[key] = reg
		return reg.server, nil
	}
	log.Println("starting new server", key)
	lspImpl, ok := languagemapping.GetImplementation(language)
	if !ok {
		return LanguageServer{}, fmt.Errorf("no LSP implementation for language %q", language)
	}
	client, err := Start(lspImpl, rootDir)
	if err != nil {
		return LanguageServer{}, err
	}
	runningServers[key] = ServerRegister{
		LastUsed: time.Now(),
		server:   client,
		rootDir:  rootDir,
		language: language,
	}
	return client, nil
}

var idleTimeout = 10 * time.Minute

func stopIdleServers() {
	mutex.Lock()
	defer mutex.Unlock()
	var stoppedKeys []string
	for key, reg := range runningServers {
		if time.Since(reg.LastUsed) > idleTimeout {
			log.Println("stopping idle server", key)
			stoppedKeys = append(stoppedKeys, key)
			reg.server.Stop()
		}
	}
	for _, key := range stoppedKeys {
		delete(runningServers, key)
	}
}

func StopAll() {
	mutex.Lock()
	defer mutex.Unlock()
	wg := &sync.WaitGroup{}
	wg.Add(len(runningServers))
	for _, reg := range runningServers {
		go func(reg ServerRegister) {
			defer wg.Done()
			reg.server.Stop()
		}(reg)
	}
	wg.Wait()
	runningServers = make(map[string]ServerRegister)
}
