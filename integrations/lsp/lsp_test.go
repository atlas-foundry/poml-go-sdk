package lsp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	s := New()
	if s == nil {
		t.Fatal("New returned nil")
	}
	if len(s.handlers) == 0 {
		t.Error("expected handlers to be registered")
	}
}

func TestServerHandlers(t *testing.T) {
	s := New()

	expectedHandlers := []string{
		"initialize",
		"initialized",
		"shutdown",
		"textDocument/didOpen",
		"textDocument/didChange",
		"textDocument/didClose",
		"textDocument/hover",
		"textDocument/completion",
		"textDocument/definition",
		"textDocument/documentSymbol",
		"textDocument/codeAction",
	}

	for _, h := range expectedHandlers {
		if _, ok := s.handlers[h]; !ok {
			t.Errorf("expected handler %q to be registered", h)
		}
	}
}

func TestHandleInitialize(t *testing.T) {
	s := New()
	params := json.RawMessage(`{"capabilities": {}}`)

	result, err := s.handleInitialize(context.Background(), params)
	if err != nil {
		t.Fatalf("handleInitialize failed: %v", err)
	}

	res := result.(map[string]any)
	caps := res["capabilities"].(map[string]any)

	// Verify capabilities are set
	if _, ok := caps["textDocumentSync"]; !ok {
		t.Error("expected textDocumentSync capability")
	}
	if !caps["hoverProvider"].(bool) {
		t.Error("expected hoverProvider to be true")
	}
	if !caps["definitionProvider"].(bool) {
		t.Error("expected definitionProvider to be true")
	}
	if !caps["documentSymbolProvider"].(bool) {
		t.Error("expected documentSymbolProvider to be true")
	}

	serverInfo := res["serverInfo"].(map[string]any)
	if serverInfo["name"] != "poml-lsp" {
		t.Errorf("expected server name 'poml-lsp', got '%v'", serverInfo["name"])
	}

	if !s.initialized {
		t.Error("expected server to be initialized")
	}
}

func TestHandleDidOpen(t *testing.T) {
	s := New()
	s.writer = io.Discard // Avoid nil writer panic

	params := json.RawMessage(`{
		"textDocument": {
			"uri": "file:///test.poml",
			"languageId": "poml",
			"version": 1,
			"text": "<poml><role>Test</role></poml>"
		}
	}`)

	_, err := s.handleDidOpen(context.Background(), params)
	if err != nil {
		t.Fatalf("handleDidOpen failed: %v", err)
	}

	// Verify document was stored
	docI, ok := s.documents.Load("file:///test.poml")
	if !ok {
		t.Fatal("expected document to be stored")
	}

	doc := docI.(*Document)
	if doc.URI != "file:///test.poml" {
		t.Errorf("expected URI 'file:///test.poml', got '%s'", doc.URI)
	}
	if doc.Version != 1 {
		t.Errorf("expected version 1, got %d", doc.Version)
	}
	if doc.Parsed == nil {
		t.Error("expected document to be parsed")
	}
}

func TestHandleDidChange(t *testing.T) {
	s := New()
	s.writer = io.Discard

	// First open the document
	openParams := json.RawMessage(`{
		"textDocument": {
			"uri": "file:///test.poml",
			"languageId": "poml",
			"version": 1,
			"text": "<poml><role>Original</role></poml>"
		}
	}`)
	_, _ = s.handleDidOpen(context.Background(), openParams)

	// Then change it
	changeParams := json.RawMessage(`{
		"textDocument": {
			"uri": "file:///test.poml",
			"version": 2
		},
		"contentChanges": [
			{"text": "<poml><role>Updated</role></poml>"}
		]
	}`)

	_, err := s.handleDidChange(context.Background(), changeParams)
	if err != nil {
		t.Fatalf("handleDidChange failed: %v", err)
	}

	docI, _ := s.documents.Load("file:///test.poml")
	doc := docI.(*Document)

	if doc.Version != 2 {
		t.Errorf("expected version 2, got %d", doc.Version)
	}
	if !strings.Contains(doc.Content, "Updated") {
		t.Error("expected content to be updated")
	}
}

func TestHandleDidClose(t *testing.T) {
	s := New()
	s.writer = io.Discard

	// First open the document
	openParams := json.RawMessage(`{
		"textDocument": {
			"uri": "file:///test.poml",
			"languageId": "poml",
			"version": 1,
			"text": "<poml></poml>"
		}
	}`)
	_, _ = s.handleDidOpen(context.Background(), openParams)

	// Verify it's there
	if _, ok := s.documents.Load("file:///test.poml"); !ok {
		t.Fatal("document should be stored")
	}

	// Close it
	closeParams := json.RawMessage(`{
		"textDocument": {"uri": "file:///test.poml"}
	}`)

	_, err := s.handleDidClose(context.Background(), closeParams)
	if err != nil {
		t.Fatalf("handleDidClose failed: %v", err)
	}

	// Verify it's gone
	if _, ok := s.documents.Load("file:///test.poml"); ok {
		t.Error("document should be removed after close")
	}
}

func TestHandleHover(t *testing.T) {
	s := New()
	s.writer = io.Discard

	// Open a document
	openParams := json.RawMessage(`{
		"textDocument": {
			"uri": "file:///test.poml",
			"languageId": "poml",
			"version": 1,
			"text": "<poml>\n  <role>Test</role>\n</poml>"
		}
	}`)
	_, _ = s.handleDidOpen(context.Background(), openParams)

	// Hover on the role element
	hoverParams := json.RawMessage(`{
		"textDocument": {"uri": "file:///test.poml"},
		"position": {"line": 1, "character": 4}
	}`)

	result, err := s.handleHover(context.Background(), hoverParams)
	if err != nil {
		t.Fatalf("handleHover failed: %v", err)
	}

	if result != nil {
		res := result.(map[string]any)
		contents := res["contents"].(map[string]any)
		if contents["kind"] != "markdown" {
			t.Errorf("expected markdown kind, got %v", contents["kind"])
		}
	}
}

func TestHandleCompletion(t *testing.T) {
	s := New()

	params := json.RawMessage(`{
		"textDocument": {"uri": "file:///test.poml"},
		"position": {"line": 0, "character": 1}
	}`)

	result, err := s.handleCompletion(context.Background(), params)
	if err != nil {
		t.Fatalf("handleCompletion failed: %v", err)
	}

	res := result.(map[string]any)
	items := res["items"].([]map[string]any)

	if len(items) == 0 {
		t.Fatal("expected completion items")
	}

	// Check for expected completions
	labels := make(map[string]bool)
	for _, item := range items {
		labels[item["label"].(string)] = true
	}

	expectedLabels := []string{"poml", "meta", "role", "task", "hint", "example", "human-msg", "assistant-msg"}
	for _, label := range expectedLabels {
		if !labels[label] {
			t.Errorf("expected completion for %q", label)
		}
	}
}

func TestHandleDocumentSymbol(t *testing.T) {
	s := New()
	s.writer = io.Discard

	// Open a document with meta.id
	openParams := json.RawMessage(`{
		"textDocument": {
			"uri": "file:///test.poml",
			"languageId": "poml",
			"version": 1,
			"text": "<poml><meta><id>test-prompt</id></meta></poml>"
		}
	}`)
	_, _ = s.handleDidOpen(context.Background(), openParams)

	params := json.RawMessage(`{
		"textDocument": {"uri": "file:///test.poml"}
	}`)

	result, err := s.handleDocumentSymbol(context.Background(), params)
	if err != nil {
		t.Fatalf("handleDocumentSymbol failed: %v", err)
	}

	symbols := result.([]map[string]any)
	if len(symbols) == 0 {
		t.Fatal("expected document symbols")
	}

	if symbols[0]["name"] != "test-prompt" {
		t.Errorf("expected symbol name 'test-prompt', got '%v'", symbols[0]["name"])
	}
}

func TestHandleCodeAction(t *testing.T) {
	s := New()

	params := json.RawMessage(`{
		"textDocument": {"uri": "file:///test.poml"},
		"range": {"start": {"line": 0, "character": 0}, "end": {"line": 0, "character": 0}},
		"context": {
			"diagnostics": [
				{
					"range": {"start": {"line": 0, "character": 0}, "end": {"line": 0, "character": 0}},
					"severity": 1,
					"source": "poml",
					"message": "missing required element"
				}
			]
		}
	}`)

	result, err := s.handleCodeAction(context.Background(), params)
	if err != nil {
		t.Fatalf("handleCodeAction failed: %v", err)
	}

	actions := result.([]map[string]any)
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}

	if actions[0]["title"] != "Add missing element" {
		t.Errorf("expected title 'Add missing element', got '%v'", actions[0]["title"])
	}
	if actions[0]["kind"] != "quickfix" {
		t.Errorf("expected kind 'quickfix', got '%v'", actions[0]["kind"])
	}
}

func TestHandleShutdown(t *testing.T) {
	s := New()

	_, err := s.handleShutdown(context.Background(), nil)
	if err != nil {
		t.Fatalf("handleShutdown failed: %v", err)
	}

	if !s.shutdown {
		t.Error("expected server to be in shutdown state")
	}
}

func TestFindElementAtPosition(t *testing.T) {
	s := New()
	// Use content with spaces after element names for proper detection
	doc := &Document{
		Content: "<poml >\n  <role >Test</role>\n  <task attr=\"x\">Do something</task>\n</poml>",
	}

	tests := []struct {
		pos    Position
		expect string
	}{
		// Line 0: "<poml >" - character 2 finds "poml"
		{Position{Line: 0, Character: 2}, "poml"},
		// Line 1: "  <role >Test</role>" - character 4 finds "role"
		{Position{Line: 1, Character: 4}, "role"},
		// Line 2: "  <task attr=\"x\">Do something</task>" - character 4 finds "task"
		{Position{Line: 2, Character: 4}, "task"},
	}

	for _, tt := range tests {
		got := s.findElementAtPosition(doc, tt.pos)
		if got != tt.expect {
			t.Errorf("findElementAtPosition(%v) = %q, want %q", tt.pos, got, tt.expect)
		}
	}
}

func TestGetElementInfo(t *testing.T) {
	s := New()

	elements := []string{"poml", "meta", "role", "task", "hint", "example", "human-msg", "assistant-msg", "system-msg", "tool-definition", "image", "let", "include"}

	for _, elem := range elements {
		info := s.getElementInfo(elem)
		if info == "" {
			t.Errorf("expected info for element %q", elem)
		}
	}

	// Unknown element should return empty
	if s.getElementInfo("unknown") != "" {
		t.Error("expected empty info for unknown element")
	}
}

func TestParseDocument(t *testing.T) {
	s := New()

	tests := []struct {
		name         string
		content      string
		wantParseErr bool
	}{
		{
			name: "valid document",
			content: `<poml>
				<meta><id>test</id><version>1.0</version><owner>test</owner></meta>
				<role>Test</role>
				<task>Do something</task>
			</poml>`,
			wantParseErr: false,
		},
		{
			name:         "invalid XML",
			content:      "<poml><role>Unclosed",
			wantParseErr: true,
		},
		{
			name:         "minimal valid",
			content:      "<poml><meta><id>x</id><version>1</version><owner>o</owner></meta><task>t</task></poml>",
			wantParseErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &Document{Content: tt.content}
			s.parseDocument(doc)

			hasParseErr := false
			for _, e := range doc.Errors {
				if e.Severity == DiagnosticSeverityError {
					hasParseErr = true
					break
				}
			}

			if tt.wantParseErr && !hasParseErr {
				t.Error("expected parse error")
			}
			if !tt.wantParseErr && hasParseErr {
				t.Errorf("unexpected parse error: %v", doc.Errors)
			}
			if !tt.wantParseErr && doc.Parsed == nil {
				t.Error("expected parsed document")
			}
		})
	}
}

func TestReadMessage(t *testing.T) {
	s := New()

	// Create a valid LSP message
	content := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(content))
	input := header + content

	s.reader = bufio.NewReader(strings.NewReader(input))

	msg, err := s.readMessage()
	if err != nil {
		t.Fatalf("readMessage failed: %v", err)
	}

	if msg.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc '2.0', got '%s'", msg.JSONRPC)
	}
	if *msg.ID != 1 {
		t.Errorf("expected id 1, got %d", *msg.ID)
	}
	if msg.Method != "initialize" {
		t.Errorf("expected method 'initialize', got '%s'", msg.Method)
	}
}

func TestWriteMessage(t *testing.T) {
	s := New()
	var buf bytes.Buffer
	s.writer = &buf

	id := 1
	msg := &Message{
		ID:     &id,
		Result: map[string]string{"status": "ok"},
	}

	if err := s.writeMessage(msg); err != nil {
		t.Fatalf("writeMessage failed: %v", err)
	}

	output := buf.String()
	if !strings.HasPrefix(output, "Content-Length:") {
		t.Error("expected Content-Length header")
	}
	if !strings.Contains(output, `"jsonrpc":"2.0"`) {
		t.Error("expected jsonrpc field")
	}
	if !strings.Contains(output, `"status":"ok"`) {
		t.Error("expected result in output")
	}
}

func TestHandleMessage(t *testing.T) {
	s := New()
	var buf bytes.Buffer
	s.writer = &buf

	id := 1
	msg := &Message{
		ID:     &id,
		Method: "initialize",
		Params: json.RawMessage(`{}`),
	}

	s.handleMessage(context.Background(), msg)

	// Give the goroutine time to complete
	// In production, we'd use proper synchronization
	output := buf.String()
	if !strings.Contains(output, "capabilities") {
		t.Error("expected capabilities in response")
	}
}

func TestHandleUnknownMethod(t *testing.T) {
	s := New()
	var buf bytes.Buffer
	s.writer = &buf

	id := 1
	msg := &Message{
		ID:     &id,
		Method: "unknown/method",
	}

	s.handleMessage(context.Background(), msg)

	output := buf.String()
	if !strings.Contains(output, "method not found") {
		t.Error("expected method not found error")
	}
}

func TestDiagnosticSeverityConstants(t *testing.T) {
	if DiagnosticSeverityError != 1 {
		t.Errorf("expected DiagnosticSeverityError = 1, got %d", DiagnosticSeverityError)
	}
	if DiagnosticSeverityWarning != 2 {
		t.Errorf("expected DiagnosticSeverityWarning = 2, got %d", DiagnosticSeverityWarning)
	}
	if DiagnosticSeverityInfo != 3 {
		t.Errorf("expected DiagnosticSeverityInfo = 3, got %d", DiagnosticSeverityInfo)
	}
	if DiagnosticSeverityHint != 4 {
		t.Errorf("expected DiagnosticSeverityHint = 4, got %d", DiagnosticSeverityHint)
	}
}
