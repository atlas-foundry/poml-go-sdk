package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/atlas-foundry/poml-go-sdk/poml"
	"github.com/gorilla/websocket"
)

func TestInspect(t *testing.T) {
	doc, err := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task></poml>`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := doc.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	srv := New(doc, "", nil)
	req := httptest.NewRequest(http.MethodGet, "/inspect", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("inspect status = %d", rec.Code)
	}
	var summary inspectSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if summary.Meta.ID != "a" {
		t.Fatalf("meta.id = %s", summary.Meta.ID)
	}
	if got := summary.Counts["meta"]; got != 1 {
		t.Fatalf("meta count = %d", got)
	}
	if got := summary.Counts["task"]; got != 1 {
		t.Fatalf("task count = %d", got)
	}
}

func TestAST(t *testing.T) {
	doc, err := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task></poml>`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	srv := New(doc, "", nil)
	req := httptest.NewRequest(http.MethodGet, "/ast", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ast status = %d", rec.Code)
	}
	var out poml.Document
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode ast: %v", err)
	}
	if out.Meta.ID != "a" {
		t.Fatalf("expected meta.id a, got %s", out.Meta.ID)
	}
}

func TestValidateEndpoint(t *testing.T) {
	doc, err := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task></poml>`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	srv := New(doc, "", nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/validate", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("validate status = %d", rec.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode validate: %v", err)
	}
	if ok, _ := resp["ok"].(bool); !ok {
		t.Fatalf("expected ok true, got %v", resp["ok"])
	}
}

func TestConvertEndpoint(t *testing.T) {
	doc, err := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task><human-msg>Hello</human-msg></poml>`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	srv := New(doc, "", nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/convert?format=dict", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("convert status = %d", rec.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode convert: %v", err)
	}
	if _, ok := resp["messages"]; !ok {
		t.Fatalf("expected messages in convert output")
	}
}

func TestSearchEndpoint(t *testing.T) {
	doc, err := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task><human-msg>Hello world</human-msg></poml>`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	srv := New(doc, "", nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/search?tag=task", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("search status = %d", rec.Code)
	}
	var resp struct {
		Count   int              `json:"count"`
		Matches []map[string]any `json:"matches"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode search: %v", err)
	}
	if resp.Count != 1 {
		t.Fatalf("expected 1 match, got %d", resp.Count)
	}
	if resp.Matches[0]["type"] != "task" {
		t.Fatalf("expected type task, got %v", resp.Matches[0]["type"])
	}
}

func TestHealth(t *testing.T) {
	doc := poml.Document{}
	srv := New(doc, "", nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("health status = %d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("health body = %s", rec.Body.String())
	}
}

func TestToolsEndpoint(t *testing.T) {
	doc, err := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task><tool-definition name="foo" description="bar"/><tool-request id="1" name="foo"/></poml>`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	srv := New(doc, "", nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tools", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("tools status = %d", rec.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode tools: %v", err)
	}
	if defs, ok := resp["tool_definitions"].([]any); !ok || len(defs) != 1 {
		t.Fatalf("expected 1 tool definition, got %v", resp["tool_definitions"])
	}
}

func TestDiagramEndpoint(t *testing.T) {
	doc, err := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task><diagram id="d"><graph><node id="n"/></graph></diagram></poml>`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	srv := New(doc, "", nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/diagram", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("diagram status = %d", rec.Code)
	}
	var resp []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode diagram: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("expected 1 diagram, got %d", len(resp))
	}
}

func TestRoundtripEndpoint(t *testing.T) {
	doc, err := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task></poml>`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	srv := New(doc, "", nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/roundtrip", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("roundtrip status = %d", rec.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode roundtrip: %v", err)
	}
	if ok, _ := resp["ok"].(bool); !ok {
		t.Fatalf("expected roundtrip ok true, got %v", resp["ok"])
	}
}

func TestWebSocketInitialAndUpdate(t *testing.T) {
	srv := New(poml.Document{
		Meta:  poml.Meta{ID: "a", Version: "1", Owner: "o"},
		Role:  poml.Block{Body: "r"},
		Tasks: []poml.Block{{Body: "t"}},
	}, "", nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	u := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// initial message
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ws read initial: %v", err)
	}
	if !strings.Contains(string(msg), `"event":"initial"`) {
		t.Fatalf("expected initial event, got %s", msg)
	}

	// simulate update and expect broadcast
	srv.updateDoc(poml.Document{
		Meta:  poml.Meta{ID: "b", Version: "1", Owner: "o"},
		Role:  poml.Block{Body: "r"},
		Tasks: []poml.Block{{Body: "t"}},
	})
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for {
		_, msg, err = conn.ReadMessage()
		if err != nil {
			t.Fatalf("ws read update: %v", err)
		}
		if strings.Contains(string(msg), `"event":"update"`) {
			break
		}
	}
}

func TestDiffEndpoint(t *testing.T) {
	srv := New(poml.Document{}, "", nil)
	body := `{"a":"<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task></poml>","b":"<poml><meta><id>b</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task></poml>"}`
	req := httptest.NewRequest(http.MethodPost, "/diff", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("diff status = %d", rec.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode diff: %v", err)
	}
	if equal, _ := resp["equal_meta"].(bool); equal {
		t.Fatalf("expected meta differ")
	}
}

func TestPatchEndpoint(t *testing.T) {
	doc, err := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task><human-msg>hi</human-msg></poml>`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	srv := New(doc, "", nil)
	body := `{"tag":"human_msg","index":0,"body":"updated"}`
	req := httptest.NewRequest(http.MethodPost, "/patch", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch status = %d", rec.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode patch: %v", err)
	}
	if ok, _ := resp["ok"].(bool); !ok {
		t.Fatalf("expected ok true, got %v", resp["ok"])
	}
}

func TestWatchDisabled(t *testing.T) {
	srv := New(poml.Document{}, "", nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/watch", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("watch disabled status = %d", rec.Code)
	}
}

func TestReloadNoSourcePath(t *testing.T) {
	srv := New(poml.Document{}, "", nil)
	if _, err := srv.reload(); err == nil {
		t.Fatalf("expected reload error with no source path")
	}
}
