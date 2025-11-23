package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/atlas-foundry/poml-go-sdk/poml"
)

func TestInspect(t *testing.T) {
	doc, err := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task></poml>`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := doc.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	srv := New(doc)
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

	srv := New(doc)
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

func TestHealth(t *testing.T) {
	doc := poml.Document{}
	srv := New(doc)
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
