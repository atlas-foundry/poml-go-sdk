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

	srv := New(doc, "", nil, nil)
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

	srv := New(doc, "", nil, nil)
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
	srv := New(doc, "", nil, nil)
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
	srv := New(doc, "", nil, nil)
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
	srv := New(doc, "", nil, nil)
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
	srv := New(doc, "", nil, nil)
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
	srv := New(doc, "", nil, nil)
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
	srv := New(doc, "", nil, nil)
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
	srv := New(doc, "", nil, nil)
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
	}, "", nil, nil)
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
	srv := New(poml.Document{}, "", nil, nil)
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
	srv := New(doc, "", nil, nil)
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

func TestTracesEndpoint(t *testing.T) {
	recorder := poml.NewTraceRecorder("mcp-trace")
	srv := New(poml.Document{
		Meta:  poml.Meta{ID: "a", Version: "1", Owner: "o"},
		Role:  poml.Block{Body: "r"},
		Tasks: []poml.Block{{Body: "t"}},
	}, "", recorder.Provider, &recorder)

	// generate a span via convert
	req := httptest.NewRequest(http.MethodGet, "/convert?format=dict", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("convert status = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/traces", nil)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("traces status = %d", rec.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode traces: %v", err)
	}
	spans, ok := resp["spans"].([]any)
	if !ok || len(spans) == 0 {
		t.Fatalf("expected spans array, got %v", resp["spans"])
	}
}

func TestWatchDisabled(t *testing.T) {
	srv := New(poml.Document{}, "", nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/watch", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("watch disabled status = %d", rec.Code)
	}
}

func TestReloadNoSourcePath(t *testing.T) {
	srv := New(poml.Document{}, "", nil, nil)
	if _, err := srv.reload(); err == nil {
		t.Fatalf("expected reload error with no source path")
	}
}

func TestValidateEndpointInvalid(t *testing.T) {
	// Create doc without required fields to trigger validation error
	doc := poml.Document{}
	srv := New(doc, "", nil, nil)
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
	if ok, _ := resp["ok"].(bool); ok {
		t.Fatalf("expected ok false for invalid doc")
	}
}

func TestConvertEndpointBadFormat(t *testing.T) {
	doc, _ := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task></poml>`)
	srv := New(doc, "", nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/convert?format=invalid", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("convert bad format status = %d, expected 400", rec.Code)
	}
}

func TestConvertEndpointMessageDict(t *testing.T) {
	doc, _ := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task><human-msg>Hello</human-msg></poml>`)
	srv := New(doc, "", nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/convert?format=message_dict", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("convert message_dict status = %d", rec.Code)
	}
}

func TestConvertEndpointOpenAI(t *testing.T) {
	doc, _ := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task><human-msg>Hello</human-msg></poml>`)
	srv := New(doc, "", nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/convert?format=openai_chat", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("convert openai_chat status = %d", rec.Code)
	}
}

func TestConvertEndpointLangChain(t *testing.T) {
	doc, _ := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task><human-msg>Hello</human-msg></poml>`)
	srv := New(doc, "", nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/convert?format=langchain", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("convert langchain status = %d", rec.Code)
	}
}

func TestSearchEndpointNoTag(t *testing.T) {
	doc, _ := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task></poml>`)
	srv := New(doc, "", nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("search no tag status = %d", rec.Code)
	}
	var resp struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode search: %v", err)
	}
	// Should return all elements when no tag filter
	if resp.Count == 0 {
		t.Fatalf("expected some matches with no tag filter")
	}
}

func TestSearchEndpointWithAttr(t *testing.T) {
	doc, _ := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task><input name="foo">bar</input></poml>`)
	srv := New(doc, "", nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/search?tag=input&attr=name=foo", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("search with attr status = %d", rec.Code)
	}
	var resp struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode search: %v", err)
	}
	if resp.Count != 1 {
		t.Fatalf("expected 1 match with attr filter, got %d", resp.Count)
	}
}

func TestSearchEndpointWithBody(t *testing.T) {
	doc, _ := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>special task</task></poml>`)
	srv := New(doc, "", nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/search?tag=task&body=special", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("search with body status = %d", rec.Code)
	}
	var resp struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode search: %v", err)
	}
	if resp.Count != 1 {
		t.Fatalf("expected 1 match with body filter, got %d", resp.Count)
	}
}

func TestDiffEndpointBadJSON(t *testing.T) {
	srv := New(poml.Document{}, "", nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/diff", strings.NewReader("not json"))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("diff bad json status = %d, expected 400", rec.Code)
	}
}

func TestDiffEndpointBadPOML(t *testing.T) {
	srv := New(poml.Document{}, "", nil, nil)
	body := `{"a":"<poml><invalid","b":"<poml></poml>"}`
	req := httptest.NewRequest(http.MethodPost, "/diff", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("diff bad poml status = %d, expected 400", rec.Code)
	}
}

func TestPatchEndpointBadJSON(t *testing.T) {
	srv := New(poml.Document{}, "", nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/patch", strings.NewReader("not json"))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("patch bad json status = %d, expected 400", rec.Code)
	}
}

func TestPatchEndpointUnsupportedTag(t *testing.T) {
	doc, _ := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task></poml>`)
	srv := New(doc, "", nil, nil)
	body := `{"tag":"unsupported","index":0,"body":"x"}`
	req := httptest.NewRequest(http.MethodPost, "/patch", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("patch unsupported tag status = %d, expected 400", rec.Code)
	}
}

func TestPatchEndpointTask(t *testing.T) {
	doc, _ := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>old</task></poml>`)
	srv := New(doc, "", nil, nil)
	body := `{"tag":"task","index":0,"body":"new task"}`
	req := httptest.NewRequest(http.MethodPost, "/patch", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch task status = %d", rec.Code)
	}
}

func TestPatchEndpointRole(t *testing.T) {
	doc, _ := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>old</role><task>t</task></poml>`)
	srv := New(doc, "", nil, nil)
	body := `{"tag":"role","body":"new role"}`
	req := httptest.NewRequest(http.MethodPost, "/patch", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch role status = %d", rec.Code)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	doc, _ := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task></poml>`)
	srv := New(doc, "", nil, nil)

	// Hit some endpoints to generate counts
	srv.Handler().ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/health", nil))
	srv.Handler().ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/inspect", nil))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("metrics status = %d", rec.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode metrics: %v", err)
	}
	requests, ok := resp["requests"].([]any)
	if !ok {
		t.Fatalf("expected requests array, got %T", resp["requests"])
	}
	if len(requests) == 0 {
		t.Fatalf("expected some requests in metrics")
	}
}

func TestTracesEndpointNoRecorder(t *testing.T) {
	srv := New(poml.Document{}, "", nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/traces", nil)
	srv.Handler().ServeHTTP(rec, req)
	// When no recorder configured, returns 404
	if rec.Code != http.StatusNotFound {
		t.Fatalf("traces no recorder status = %d, expected 404", rec.Code)
	}
}

func TestCountEndpoint(t *testing.T) {
	doc, _ := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task><task>t2</task><human-msg>hi</human-msg></poml>`)
	srv := New(doc, "", nil, nil)

	tests := []struct {
		tag   string
		count int
	}{
		{"task", 2},
		{"human_msg", 1},
		{"meta", 1},
		{"role", 1},
	}

	for _, tc := range tests {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/search?tag="+tc.tag, nil)
		srv.Handler().ServeHTTP(rec, req)
		var resp struct {
			Count int `json:"count"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode search: %v", err)
		}
		if resp.Count != tc.count {
			t.Errorf("tag %s: expected %d, got %d", tc.tag, tc.count, resp.Count)
		}
	}
}

func TestAuthorizationWithToken(t *testing.T) {
	doc, _ := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task></poml>`)
	srv := New(doc, "", nil, nil)
	srv.authToken = "secret"

	// Without token - should fail (using /metrics which passes r to authorize)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}

	// With wrong token (via query param)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/metrics?token=wrong", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong token, got %d", rec.Code)
	}

	// With correct token (via query param)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/metrics?token=secret", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with correct token, got %d", rec.Code)
	}
}

func TestSearchVariousElementTypes(t *testing.T) {
	// Test search with many element types to cover extractAttrsBody branches
	doc, err := poml.ParseString(`<poml>
		<meta><id>a</id><version>1</version><owner>o</owner></meta>
		<role>r</role>
		<task>t</task>
		<input name="foo" required="true">bar</input>
		<hint>hint text</hint>
		<example>example text</example>
		<human-msg>hello</human-msg>
		<assistant-msg>hi</assistant-msg>
		<system-msg>system</system-msg>
		<tool-definition name="tool1" description="desc"/>
		<tool-request id="tr1" name="tool1"/>
		<tool-response id="tr1" name="tool1">response</tool-response>
		<tool-result id="tr1" name="tool1">result</tool-result>
		<tool-error id="tr1" name="tool1">error</tool-error>
		<image src="img.png" alt="image"/>
		<audio src="audio.mp3" alt="audio"/>
		<video src="video.mp4" alt="video"/>
		<document src="doc.pdf"/>
		<output-schema>{"type":"object"}</output-schema>
		<style><output format="json"/></style>
		<output-format>markdown</output-format>
		<runtime model="gpt-4"/>
		<diagram id="d1"><graph><node id="n1"/></graph></diagram>
	</poml>`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	srv := New(doc, "", nil, nil)

	// Test each element type
	tags := []string{
		"meta", "role", "task", "input", "hint", "example",
		"human_msg", "assistant_msg", "system_msg",
		"tool_definition", "tool_request", "tool_response", "tool_result", "tool_error",
		"image", "audio", "video", "document",
		"output_schema", "style", "output_format", "runtime", "diagram",
	}

	for _, tag := range tags {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/search?tag="+tag, nil)
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("search tag=%s status=%d", tag, rec.Code)
		}
	}
}

func TestSearchWithAttrSubstringMatch(t *testing.T) {
	doc, _ := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task><input name="foobar">x</input></poml>`)
	srv := New(doc, "", nil, nil)

	// Test substring match (without =)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/search?tag=input&attr=foo", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("search status = %d", rec.Code)
	}
	var resp struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Count != 1 {
		t.Fatalf("expected 1 match with substring attr filter, got %d", resp.Count)
	}
}

func TestSearchWithAttrNoMatch(t *testing.T) {
	doc, _ := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task><input name="bar">x</input></poml>`)
	srv := New(doc, "", nil, nil)

	// Test key=value that doesn't match
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/search?tag=input&attr=name=xyz", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("search status = %d", rec.Code)
	}
	var resp struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Count != 0 {
		t.Fatalf("expected 0 matches, got %d", resp.Count)
	}
}

func TestConvertSceneFormat(t *testing.T) {
	doc, _ := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task></poml>`)
	srv := New(doc, "", nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/convert?format=scene", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("convert scene status = %d", rec.Code)
	}
}

func TestConvertPydanticFormat(t *testing.T) {
	doc, _ := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task></poml>`)
	srv := New(doc, "", nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/convert?format=pydantic", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("convert pydantic status = %d", rec.Code)
	}
}

func TestRoundtripWithError(t *testing.T) {
	// Create a document and test roundtrip
	doc, _ := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task></poml>`)
	srv := New(doc, "", nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/roundtrip", nil)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("roundtrip status = %d", rec.Code)
	}
}

func TestDiffEqualDocs(t *testing.T) {
	srv := New(poml.Document{}, "", nil, nil)
	body := `{"a":"<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task></poml>","b":"<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task></poml>"}`
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
	if equal, _ := resp["equal_meta"].(bool); !equal {
		t.Fatalf("expected equal_meta true for identical docs")
	}
}

func TestPatchAssistantMsg(t *testing.T) {
	doc, _ := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task><assistant-msg>old</assistant-msg></poml>`)
	srv := New(doc, "", nil, nil)
	body := `{"tag":"assistant_msg","index":0,"body":"new msg"}`
	req := httptest.NewRequest(http.MethodPost, "/patch", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch assistant_msg status = %d", rec.Code)
	}
}

func TestPatchHumanMsg(t *testing.T) {
	doc, _ := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task><human-msg>old</human-msg></poml>`)
	srv := New(doc, "", nil, nil)
	body := `{"tag":"human_msg","index":0,"body":"new msg"}`
	req := httptest.NewRequest(http.MethodPost, "/patch", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch human_msg status = %d", rec.Code)
	}
}
