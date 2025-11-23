package mcp

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/atlas-foundry/poml-go-sdk/poml"
)

// Server exposes a minimal MCP-style HTTP surface to inspect parsed POML docs.
// It is intentionally small: a health endpoint, an AST dump, and a summary.
type Server struct {
	doc     poml.Document
	mux     *http.ServeMux
	once    sync.Once
	summary inspectSummary
}

type inspectSummary struct {
	Meta    poml.Meta          `json:"meta"`
	Counts  map[string]int     `json:"counts"`
	Created time.Time          `json:"created"`
	IDs     []string           `json:"element_ids,omitempty"`
	Types   []poml.ElementType `json:"element_types,omitempty"`
}

// New creates a server for the given document.
func New(doc poml.Document) *Server {
	s := &Server{
		doc: doc,
		mux: http.NewServeMux(),
	}
	s.mux.HandleFunc("/health", s.health)
	s.mux.HandleFunc("/inspect", s.inspect)
	s.mux.HandleFunc("/ast", s.ast)
	return s
}

// Handler returns the server's HTTP handler (useful for tests/embedding).
func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) inspect(w http.ResponseWriter, _ *http.Request) {
	s.once.Do(func() {
		s.summary = buildSummary(s.doc)
	})
	writeJSON(w, s.summary)
}

func (s *Server) ast(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, s.doc)
}

func buildSummary(doc poml.Document) inspectSummary {
	counts := make(map[string]int)
	var ids []string
	var types []poml.ElementType
	for _, el := range doc.Elements {
		counts[string(el.Type)]++
		if el.ID != "" {
			ids = append(ids, el.ID)
		}
		types = append(types, el.Type)
	}
	return inspectSummary{
		Meta:    doc.Meta,
		Counts:  counts,
		Created: time.Now().UTC(),
		IDs:     ids,
		Types:   types,
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
