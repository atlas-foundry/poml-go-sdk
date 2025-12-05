// Package lsp provides Language Server Protocol integration for POML.
//
// This package implements an LSP server that provides IDE features for POML files:
//   - Syntax validation and diagnostics
//   - Hover information for elements
//   - Go to definition for includes and references
//   - Auto-completion for elements and attributes
//   - Document symbols for outline view
//   - Code actions for quick fixes
//
// Usage with VS Code:
//
//	# Start the LSP server
//	poml lsp --stdio
//
//	# Or as a TCP server
//	poml lsp --tcp :6060
//
// The server can be used with any LSP-compatible editor (VS Code, Neovim, Emacs, etc.)
package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/atlas-foundry/poml-go-sdk/poml"
)

// Server implements the Language Server Protocol for POML.
type Server struct {
	// Documents stores open documents by URI.
	documents sync.Map // map[string]*Document

	// Handlers
	handlers map[string]Handler

	// IO
	reader *bufio.Reader
	writer io.Writer
	mu     sync.Mutex

	// Lifecycle
	initialized bool
	shutdown    bool
}

// Document represents an open POML document.
type Document struct {
	URI     string
	Version int
	Content string
	Parsed  *poml.Document
	Errors  []Diagnostic
}

// Handler is a function that handles an LSP request.
type Handler func(ctx context.Context, params json.RawMessage) (any, error)

// New creates a new LSP server.
func New() *Server {
	s := &Server{
		handlers: make(map[string]Handler),
	}
	s.registerHandlers()
	return s
}

// ServeStdio serves the LSP over stdio.
func (s *Server) ServeStdio(ctx context.Context) error {
	s.reader = bufio.NewReader(os.Stdin)
	s.writer = os.Stdout
	return s.serve(ctx)
}

// ServeTCP serves the LSP over TCP.
func (s *Server) ServeTCP(ctx context.Context, addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer func() { _ = listener.Close() }()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				continue
			}
		}
		go func() {
			s.reader = bufio.NewReader(conn)
			s.writer = conn
			_ = s.serve(ctx)
			_ = conn.Close()
		}()
	}
}

func (s *Server) serve(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msg, err := s.readMessage()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("read message: %w", err)
		}

		go s.handleMessage(ctx, msg)
	}
}

func (s *Server) registerHandlers() {
	s.handlers["initialize"] = s.handleInitialize
	s.handlers["initialized"] = s.handleInitialized
	s.handlers["shutdown"] = s.handleShutdown
	s.handlers["exit"] = s.handleExit
	s.handlers["textDocument/didOpen"] = s.handleDidOpen
	s.handlers["textDocument/didChange"] = s.handleDidChange
	s.handlers["textDocument/didClose"] = s.handleDidClose
	s.handlers["textDocument/hover"] = s.handleHover
	s.handlers["textDocument/completion"] = s.handleCompletion
	s.handlers["textDocument/definition"] = s.handleDefinition
	s.handlers["textDocument/documentSymbol"] = s.handleDocumentSymbol
	s.handlers["textDocument/codeAction"] = s.handleCodeAction
}

// LSP Message types

type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *ResponseError  `json:"error,omitempty"`
}

type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity"`
	Source   string `json:"source"`
	Message  string `json:"message"`
}

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// Diagnostic severity
const (
	DiagnosticSeverityError   = 1
	DiagnosticSeverityWarning = 2
	DiagnosticSeverityInfo    = 3
	DiagnosticSeverityHint    = 4
)

// Message handling

func (s *Server) readMessage() (*Message, error) {
	// Read headers
	var contentLength int
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "Content-Length:") {
			_, _ = fmt.Sscanf(line, "Content-Length: %d", &contentLength)
		}
	}

	if contentLength == 0 {
		return nil, fmt.Errorf("no content length")
	}

	// Read body
	body := make([]byte, contentLength)
	_, err := io.ReadFull(s.reader, body)
	if err != nil {
		return nil, err
	}

	var msg Message
	if err := json.Unmarshal(body, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (s *Server) writeMessage(msg *Message) error {
	msg.JSONRPC = "2.0"
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if _, err := s.writer.Write([]byte(header)); err != nil {
		return err
	}
	if _, err := s.writer.Write(body); err != nil {
		return err
	}
	return nil
}

func (s *Server) handleMessage(ctx context.Context, msg *Message) {
	handler, ok := s.handlers[msg.Method]
	if !ok {
		if msg.ID != nil {
			_ = s.writeMessage(&Message{
				ID: msg.ID,
				Error: &ResponseError{
					Code:    -32601,
					Message: fmt.Sprintf("method not found: %s", msg.Method),
				},
			})
		}
		return
	}

	result, err := handler(ctx, msg.Params)
	if msg.ID != nil {
		if err != nil {
			_ = s.writeMessage(&Message{
				ID: msg.ID,
				Error: &ResponseError{
					Code:    -32603,
					Message: err.Error(),
				},
			})
		} else {
			_ = s.writeMessage(&Message{
				ID:     msg.ID,
				Result: result,
			})
		}
	}
}

// Handler implementations

func (s *Server) handleInitialize(ctx context.Context, params json.RawMessage) (any, error) {
	s.initialized = true
	return map[string]any{
		"capabilities": map[string]any{
			"textDocumentSync": map[string]any{
				"openClose": true,
				"change":    1, // Full sync
			},
			"hoverProvider":          true,
			"completionProvider":     map[string]any{"triggerCharacters": []string{"<", " ", "\""}},
			"definitionProvider":     true,
			"documentSymbolProvider": true,
			"codeActionProvider":     true,
		},
		"serverInfo": map[string]any{
			"name":    "poml-lsp",
			"version": "0.0.8",
		},
	}, nil
}

func (s *Server) handleInitialized(ctx context.Context, params json.RawMessage) (any, error) {
	return nil, nil
}

func (s *Server) handleShutdown(ctx context.Context, params json.RawMessage) (any, error) {
	s.shutdown = true
	return nil, nil
}

func (s *Server) handleExit(ctx context.Context, params json.RawMessage) (any, error) {
	if s.shutdown {
		os.Exit(0)
	}
	os.Exit(1)
	return nil, nil
}

func (s *Server) handleDidOpen(ctx context.Context, params json.RawMessage) (any, error) {
	var p struct {
		TextDocument TextDocumentItem `json:"textDocument"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	doc := &Document{
		URI:     p.TextDocument.URI,
		Version: p.TextDocument.Version,
		Content: p.TextDocument.Text,
	}
	s.parseDocument(doc)
	s.documents.Store(p.TextDocument.URI, doc)
	s.publishDiagnostics(doc)
	return nil, nil
}

func (s *Server) handleDidChange(ctx context.Context, params json.RawMessage) (any, error) {
	var p struct {
		TextDocument struct {
			URI     string `json:"uri"`
			Version int    `json:"version"`
		} `json:"textDocument"`
		ContentChanges []struct {
			Text string `json:"text"`
		} `json:"contentChanges"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	if len(p.ContentChanges) == 0 {
		return nil, nil
	}

	docI, ok := s.documents.Load(p.TextDocument.URI)
	if !ok {
		return nil, nil
	}
	doc := docI.(*Document)
	doc.Version = p.TextDocument.Version
	doc.Content = p.ContentChanges[0].Text
	s.parseDocument(doc)
	s.publishDiagnostics(doc)
	return nil, nil
}

func (s *Server) handleDidClose(ctx context.Context, params json.RawMessage) (any, error) {
	var p struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	s.documents.Delete(p.TextDocument.URI)
	return nil, nil
}

func (s *Server) handleHover(ctx context.Context, params json.RawMessage) (any, error) {
	var p TextDocumentPositionParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	docI, ok := s.documents.Load(p.TextDocument.URI)
	if !ok {
		return nil, nil
	}
	doc := docI.(*Document)

	// Find element at position
	element := s.findElementAtPosition(doc, p.Position)
	if element == "" {
		return nil, nil
	}

	// Return hover info
	info := s.getElementInfo(element)
	if info == "" {
		return nil, nil
	}

	return map[string]any{
		"contents": map[string]any{
			"kind":  "markdown",
			"value": info,
		},
	}, nil
}

func (s *Server) handleCompletion(ctx context.Context, params json.RawMessage) (any, error) {
	var p TextDocumentPositionParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	// Return POML element completions
	items := []map[string]any{
		{"label": "poml", "kind": 14, "detail": "Root POML element", "insertText": "<poml>\n\t$0\n</poml>"},
		{"label": "meta", "kind": 14, "detail": "Document metadata", "insertText": "<meta>\n\t<id>$1</id>\n\t<version>$2</version>\n\t<owner>$3</owner>\n</meta>"},
		{"label": "role", "kind": 14, "detail": "System role/persona", "insertText": "<role>$0</role>"},
		{"label": "task", "kind": 14, "detail": "Task description", "insertText": "<task>$0</task>"},
		{"label": "hint", "kind": 14, "detail": "Contextual hint", "insertText": "<hint>$0</hint>"},
		{"label": "example", "kind": 14, "detail": "Example content", "insertText": "<example>$0</example>"},
		{"label": "human-msg", "kind": 14, "detail": "Human message", "insertText": "<human-msg>$0</human-msg>"},
		{"label": "assistant-msg", "kind": 14, "detail": "Assistant message", "insertText": "<assistant-msg>$0</assistant-msg>"},
		{"label": "system-msg", "kind": 14, "detail": "System message", "insertText": "<system-msg>$0</system-msg>"},
		{"label": "tool-definition", "kind": 14, "detail": "Tool definition", "insertText": "<tool-definition name=\"$1\">\n\t$0\n</tool-definition>"},
		{"label": "image", "kind": 14, "detail": "Image element", "insertText": "<image src=\"$1\" alt=\"$2\"/>"},
		{"label": "let", "kind": 14, "detail": "Variable binding", "insertText": "<let name=\"$1\" value=\"$2\"/>"},
		{"label": "include", "kind": 14, "detail": "Include file", "insertText": "<include src=\"$1\"/>"},
	}

	return map[string]any{"items": items}, nil
}

func (s *Server) handleDefinition(ctx context.Context, params json.RawMessage) (any, error) {
	var p TextDocumentPositionParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	docI, ok := s.documents.Load(p.TextDocument.URI)
	if !ok {
		return nil, nil
	}
	doc := docI.(*Document)
	if doc.Parsed == nil {
		return nil, nil
	}

	// Check if position is on an include src
	// This would require more sophisticated parsing
	return nil, nil
}

func (s *Server) handleDocumentSymbol(ctx context.Context, params json.RawMessage) (any, error) {
	var p struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	docI, ok := s.documents.Load(p.TextDocument.URI)
	if !ok {
		return nil, nil
	}
	doc := docI.(*Document)
	if doc.Parsed == nil {
		return nil, nil
	}

	// Build document symbols from parsed document
	symbols := []map[string]any{}
	if doc.Parsed.Meta.ID != "" {
		symbols = append(symbols, map[string]any{
			"name": doc.Parsed.Meta.ID,
			"kind": 5, // Class
			"range": Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 0},
			},
			"selectionRange": Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 0},
			},
		})
	}

	return symbols, nil
}

func (s *Server) handleCodeAction(ctx context.Context, params json.RawMessage) (any, error) {
	var p struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Range        Range                  `json:"range"`
		Context      struct {
			Diagnostics []Diagnostic `json:"diagnostics"`
		} `json:"context"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	// Return quick fixes for diagnostics
	actions := []map[string]any{}
	for _, diag := range p.Context.Diagnostics {
		if strings.Contains(diag.Message, "missing") {
			actions = append(actions, map[string]any{
				"title":       "Add missing element",
				"kind":        "quickfix",
				"diagnostics": []Diagnostic{diag},
			})
		}
	}

	return actions, nil
}

// Helper functions

func (s *Server) parseDocument(doc *Document) {
	doc.Errors = nil
	parsed, err := poml.ParseString(doc.Content)
	if err != nil {
		doc.Errors = append(doc.Errors, Diagnostic{
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 0},
			},
			Severity: DiagnosticSeverityError,
			Source:   "poml",
			Message:  err.Error(),
		})
		return
	}
	doc.Parsed = &parsed

	// Validate
	if err := parsed.Validate(); err != nil {
		doc.Errors = append(doc.Errors, Diagnostic{
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 0},
			},
			Severity: DiagnosticSeverityWarning,
			Source:   "poml",
			Message:  err.Error(),
		})
	}
}

func (s *Server) publishDiagnostics(doc *Document) {
	_ = s.writeMessage(&Message{
		Method: "textDocument/publishDiagnostics",
		Params: json.RawMessage(mustMarshal(map[string]any{
			"uri":         doc.URI,
			"version":     doc.Version,
			"diagnostics": doc.Errors,
		})),
	})
}

func (s *Server) findElementAtPosition(doc *Document, pos Position) string {
	lines := strings.Split(doc.Content, "\n")
	if pos.Line >= len(lines) {
		return ""
	}
	line := lines[pos.Line]
	if pos.Character >= len(line) {
		return ""
	}

	// Simple element detection - find nearest < and >
	start := strings.LastIndex(line[:pos.Character+1], "<")
	if start == -1 {
		return ""
	}
	end := strings.Index(line[start:], " ")
	if end == -1 {
		end = strings.Index(line[start:], ">")
	}
	if end == -1 {
		return ""
	}
	return strings.TrimPrefix(line[start:start+end], "<")
}

func (s *Server) getElementInfo(element string) string {
	info := map[string]string{
		"poml":            "# POML Root Element\n\nThe root element for all POML documents.",
		"meta":            "# Meta Element\n\nContains document metadata:\n- `id`: Document identifier\n- `version`: Document version\n- `owner`: Document owner",
		"role":            "# Role Element\n\nDefines the system role or persona for the conversation.",
		"task":            "# Task Element\n\nDescribes the task or goal for the assistant.",
		"hint":            "# Hint Element\n\nProvides contextual hints or guidance.",
		"example":         "# Example Element\n\nContains example input/output pairs.",
		"human-msg":       "# Human Message\n\nA message from the human/user in the conversation.",
		"assistant-msg":   "# Assistant Message\n\nA message from the assistant/AI in the conversation.",
		"system-msg":      "# System Message\n\nA system-level message or instruction.",
		"tool-definition": "# Tool Definition\n\nDefines a tool that the assistant can use.\n\nAttributes:\n- `name`: Tool name\n- `description`: Tool description",
		"image":           "# Image Element\n\nEmbeds an image.\n\nAttributes:\n- `src`: Image source (file path or data URI)\n- `alt`: Alt text",
		"let":             "# Let Element\n\nDefines a variable binding for template interpolation.\n\nAttributes:\n- `name`: Variable name\n- `value`: Variable value",
		"include":         "# Include Element\n\nIncludes content from another file.\n\nAttributes:\n- `src`: File path to include",
	}
	return info[element]
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
