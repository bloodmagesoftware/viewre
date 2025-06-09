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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
	"viewre/internal/languagemapping"
)

type LanguageServer struct {
	cmd         *exec.Cmd
	stdin       io.WriteCloser
	stdout      io.ReadCloser
	projectRoot string
	nextID      int
	openFiles   map[string]bool
}

type HoverResult struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
	Range       *Range `json:"range"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type lspMessage struct {
	JSONRPC string `json:"jsonrpc"`
	ID      *int   `json:"id,omitempty"`
	Method  string `json:"method,omitempty"`
	Params  any    `json:"params,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
}

type initializeParams struct {
	ProcessID    int            `json:"processId"`
	RootURI      string         `json:"rootUri"`
	Capabilities map[string]any `json:"capabilities"`
}

type textDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

type didOpenParams struct {
	TextDocument textDocumentItem `json:"textDocument"`
}

type textDocumentIdentifier struct {
	URI string `json:"uri"`
}

type hoverParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

func Start(commandName string, projectRoot string) (LanguageServer, error) {
	absProjectRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return LanguageServer{}, errors.Join(
			fmt.Errorf("failed to get absolute path for project root %q", projectRoot),
			err,
		)
	}

	cmd := exec.Command(commandName)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return LanguageServer{}, errors.Join(
			errors.New("failed to get stdin pipe for language server"),
			err,
		)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return LanguageServer{}, errors.Join(
			errors.New("failed to get stdout pipe for language server"),
			err,
		)
	}

	err = cmd.Start()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return LanguageServer{}, errors.Join(
			fmt.Errorf("failed to start language server %q", commandName),
			err,
		)
	}

	ls := LanguageServer{
		cmd:         cmd,
		stdin:       stdin,
		stdout:      stdout,
		projectRoot: absProjectRoot,
		nextID:      1,
		openFiles:   make(map[string]bool),
	}

	err = ls.initialize()
	if err != nil {
		ls.Stop()
		return LanguageServer{}, errors.Join(
			errors.New("failed to initialize language server"),
			err,
		)
	}

	log.Printf("Started language server %q for project %q", commandName, absProjectRoot)
	return ls, nil
}

func (ls *LanguageServer) Stop() {
	if ls.cmd != nil && ls.cmd.Process != nil {
		_ = ls.cmd.Process.Kill()
		log.Printf("Stopped language server for project %q", ls.projectRoot)
	}
	if ls.stdin != nil {
		_ = ls.stdin.Close()
	}
	if ls.stdout != nil {
		_ = ls.stdout.Close()
	}
}

func (ls *LanguageServer) HoverByteIndex(file string, byteOffset int) (HoverResult, error) {
	absoluteFilePath := filepath.Join(ls.projectRoot, file)
	line, column, err := byteIndexToPosition(absoluteFilePath, byteOffset)
	if err != nil {
		return HoverResult{}, err
	}
	return ls.HoverLineColumn(file, line, column)
}

func (ls *LanguageServer) HoverLineColumn(file string, line int, column int) (HoverResult, error) {
	absoluteFilePath := filepath.Join(ls.projectRoot, file)

	// Check if file exists
	if _, err := os.Stat(absoluteFilePath); err != nil {
		return HoverResult{}, errors.Join(
			fmt.Errorf("file %q does not exist", absoluteFilePath),
			err,
		)
	}

	// Open file if not already open
	if !ls.openFiles[file] {
		err := ls.openDocument(file)
		if err != nil {
			return HoverResult{}, errors.Join(
				fmt.Errorf("failed to open document %q", file),
				err,
			)
		}
		ls.openFiles[file] = true
	}

	// Request hover
	response, err := ls.requestHover(file, line, column)
	if err != nil {
		return HoverResult{}, errors.Join(
			fmt.Errorf("failed to get hover information for %q at line %d, column %d", file, line, column),
			err,
		)
	}

	return ls.parseHoverResponse(response), nil
}

func (ls *LanguageServer) initialize() error {
	id := ls.getNextID()
	initMsg := lspMessage{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  "initialize",
		Params: initializeParams{
			ProcessID: os.Getpid(),
			RootURI:   "file://" + ls.projectRoot,
			Capabilities: map[string]any{
				"textDocument": map[string]any{
					"hover": map[string]any{
						"contentFormat": []string{"markdown", "plaintext"},
					},
				},
			},
		},
	}

	err := ls.sendMessage(initMsg)
	if err != nil {
		return err
	}

	_, err = ls.waitForResponse(id)
	if err != nil {
		return err
	}

	// Send initialized notification
	initNotification := lspMessage{
		JSONRPC: "2.0",
		Method:  "initialized",
		Params:  map[string]any{},
	}

	return ls.sendMessage(initNotification)
}

func (ls *LanguageServer) openDocument(file string) error {
	absolutePath := filepath.Join(ls.projectRoot, file)
	content, err := os.ReadFile(absolutePath)
	if err != nil {
		return err
	}

	uri := "file://" + absolutePath
	didOpenMsg := lspMessage{
		JSONRPC: "2.0",
		Method:  "textDocument/didOpen",
		Params: didOpenParams{
			TextDocument: textDocumentItem{
				URI:        uri,
				LanguageID: languagemapping.GetLanguageID(filepath.Base(file)),
				Version:    1,
				Text:       string(content),
			},
		},
	}

	return ls.sendMessage(didOpenMsg)
}

func (ls *LanguageServer) requestHover(file string, line int, column int) (map[string]any, error) {
	id := ls.getNextID()
	absolutePath := filepath.Join(ls.projectRoot, file)
	uri := "file://" + absolutePath

	hoverMsg := lspMessage{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  "textDocument/hover",
		Params: hoverParams{
			TextDocument: textDocumentIdentifier{
				URI: uri,
			},
			Position: Position{
				Line:      line,
				Character: column,
			},
		},
	}

	err := ls.sendMessage(hoverMsg)
	if err != nil {
		return nil, err
	}

	return ls.waitForResponse(id)
}

func (ls *LanguageServer) sendMessage(msg lspMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	content := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(data), data)
	_, err = ls.stdin.Write([]byte(content))
	return err
}

func (ls *LanguageServer) readMessage() (map[string]any, error) {
	var headerBytes []byte
	var contentLength int

	// Read headers
	for {
		b := make([]byte, 1)
		_, err := ls.stdout.Read(b)
		if err != nil {
			return nil, err
		}
		headerBytes = append(headerBytes, b[0])

		if len(headerBytes) >= 4 &&
			string(headerBytes[len(headerBytes)-4:]) == "\r\n\r\n" {
			break
		}
	}

	// Parse headers
	headerStr := string(headerBytes[:len(headerBytes)-4])
	for _, line := range strings.Split(headerStr, "\r\n") {
		if strings.HasPrefix(line, "Content-Length:") {
			lengthStr := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			var err error
			contentLength, err = strconv.Atoi(lengthStr)
			if err != nil {
				return nil, err
			}
			break
		}
	}

	if contentLength == 0 {
		return nil, errors.New("no Content-Length header found")
	}

	// Read content
	content := make([]byte, contentLength)
	totalRead := 0
	for totalRead < contentLength {
		n, err := ls.stdout.Read(content[totalRead:])
		if err != nil {
			return nil, err
		}
		totalRead += n
	}

	var message map[string]any
	err := json.Unmarshal(content, &message)
	if err != nil {
		return nil, err
	}

	return message, nil
}

func (ls *LanguageServer) waitForResponse(expectedID int) (map[string]any, error) {
	for {
		msg, err := ls.readMessage()
		if err != nil {
			return nil, err
		}

		// Skip notifications (no ID)
		if _, hasID := msg["id"]; !hasID {
			continue
		}

		// Check if this is the response we're waiting for
		if id, ok := msg["id"].(float64); ok && int(id) == expectedID {
			return msg, nil
		}
	}
}

func (ls *LanguageServer) parseHoverResponse(response map[string]any) HoverResult {
	result := HoverResult{}

	// Check if there's a result
	resultData, ok := response["result"]
	if !ok || resultData == nil {
		return result
	}

	resultMap, ok := resultData.(map[string]any)
	if !ok {
		return result
	}

	// Parse contents
	if contents, ok := resultMap["contents"]; ok {
		if contentsMap, ok := contents.(map[string]any); ok {
			// Modern LSP format with kind and value
			if kind, ok := contentsMap["kind"].(string); ok {
				result.ContentType = kind
			} else {
				result.ContentType = "plaintext"
			}
			if value, ok := contentsMap["value"].(string); ok {
				result.Content = value
			}
		} else if contentsStr, ok := contents.(string); ok {
			// Simple string format
			result.ContentType = "plaintext"
			result.Content = contentsStr
		}
	}

	// Parse range if present
	if rangeData, ok := resultMap["range"]; ok {
		if rangeMap, ok := rangeData.(map[string]any); ok {
			result.Range = ls.parseRange(rangeMap)
		}
	}

	return result
}

func (ls *LanguageServer) parseRange(rangeMap map[string]any) *Range {
	start := ls.parsePosition(rangeMap["start"])
	end := ls.parsePosition(rangeMap["end"])
	if start == nil || end == nil {
		return nil
	}

	return &Range{
		Start: *start,
		End:   *end,
	}
}

func (ls *LanguageServer) parsePosition(posData any) *Position {
	posMap, ok := posData.(map[string]any)
	if !ok {
		return nil
	}

	line, lineOk := posMap["line"].(float64)
	character, charOk := posMap["character"].(float64)
	if !lineOk || !charOk {
		return nil
	}

	return &Position{
		Line:      int(line),
		Character: int(character),
	}
}

func (ls *LanguageServer) getNextID() int {
	id := ls.nextID
	ls.nextID++
	return id
}

func byteIndexToPosition(filename string, byteIndex int) (line, col int, err error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return 0, 0, err
	}

	if byteIndex > len(content) {
		byteIndex = len(content)
	}

	line = 0
	col = 0

	for i := 0; i < byteIndex; {
		if content[i] == '\n' {
			line++
			col = 0
			i++
		} else if content[i] == '\r' {
			line++
			col = 0
			i++
			if i < byteIndex && content[i] == '\n' {
				i++
			}
		} else {
			r, size := utf8.DecodeRune(content[i:])
			if r == utf8.RuneError {
				col++
				i++
			} else {
				utf16Encoded := utf16.Encode([]rune{r})
				col += len(utf16Encoded)
				i += size
			}
		}
	}

	return line, col, nil
}
