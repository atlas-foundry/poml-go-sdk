package mcp

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/atlas-foundry/poml-go-sdk/poml"
	"go.opentelemetry.io/otel/trace"
)

// Server exposes a minimal MCP-style HTTP surface to inspect parsed POML docs.
// It is intentionally small: a health endpoint, an AST dump, and a summary.
type Server struct {
	doc     poml.Document
	mux     *http.ServeMux
	once    sync.Once
	summary inspectSummary
	tracer  trace.Tracer
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
		doc:    doc,
		mux:    http.NewServeMux(),
		tracer: trace.NewNoopTracerProvider().Tracer("github.com/atlas-foundry/poml-go-sdk/mcp"),
	}
	s.mux.HandleFunc("/health", s.health)
	s.mux.HandleFunc("/inspect", s.inspect)
	s.mux.HandleFunc("/ast", s.ast)
	s.mux.HandleFunc("/validate", s.validate)
	s.mux.HandleFunc("/convert", s.convert)
	s.mux.HandleFunc("/search", s.search)
	s.mux.HandleFunc("/tools", s.tools)
	s.mux.HandleFunc("/diagram", s.diagram)
	s.mux.HandleFunc("/roundtrip", s.roundtrip)
	s.mux.HandleFunc("/diff", s.diff)
	s.mux.HandleFunc("/patch", s.patch)
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

func (s *Server) validate(w http.ResponseWriter, _ *http.Request) {
	err := s.doc.Validate()
	if err == nil {
		writeJSON(w, map[string]any{"ok": true})
		return
	}
	writeJSON(w, map[string]any{
		"ok":     false,
		"error":  err.Error(),
		"detail": err,
	})
}

func (s *Server) convert(w http.ResponseWriter, r *http.Request) {
	formatStr := r.URL.Query().Get("format")
	if formatStr == "" {
		formatStr = "dict"
	}
	format := poml.Format(formatStr)
	_, span := s.tracer.Start(r.Context(), "mcp.convert")
	defer span.End()
	out, err := poml.Convert(s.doc, format, poml.ConvertOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("convert error: %v", err), http.StatusBadRequest)
		return
	}
	writeJSON(w, out)
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	tag := strings.TrimSpace(r.URL.Query().Get("tag"))
	attr := strings.TrimSpace(r.URL.Query().Get("attr"))
	text := strings.TrimSpace(r.URL.Query().Get("text"))

	var matches []map[string]any
	for _, el := range s.doc.Elements {
		attrs, body := extractAttrsBody(s.doc, el)
		if tag != "" && !strings.EqualFold(tag, string(el.Type)) {
			continue
		}
		if attr != "" && !attrMatch(attrs, attr) {
			continue
		}
		if text != "" && !strings.Contains(strings.ToLower(body), strings.ToLower(text)) {
			continue
		}
		matches = append(matches, map[string]any{
			"type":   el.Type,
			"id":     el.ID,
			"index":  el.Index,
			"attrs":  attrs,
			"body":   body,
			"parent": el.Parent,
		})
	}
	writeJSON(w, map[string]any{
		"count":   len(matches),
		"matches": matches,
	})
}

func attrMatch(attrs map[string]string, filter string) bool {
	// Support key=value or substring match
	if filter == "" {
		return true
	}
	if strings.Contains(filter, "=") {
		parts := strings.SplitN(filter, "=", 2)
		key := parts[0]
		val := parts[1]
		if v, ok := attrs[key]; ok && strings.Contains(strings.ToLower(v), strings.ToLower(val)) {
			return true
		}
		return false
	}
	for k, v := range attrs {
		if strings.Contains(strings.ToLower(k), strings.ToLower(filter)) || strings.Contains(strings.ToLower(v), strings.ToLower(filter)) {
			return true
		}
	}
	return false
}

func extractAttrsBody(doc poml.Document, el poml.Element) (map[string]string, string) {
	switch el.Type {
	case poml.ElementMeta:
		return map[string]string{"id": doc.Meta.ID, "version": doc.Meta.Version, "owner": doc.Meta.Owner}, ""
	case poml.ElementRole:
		return attrsToMap(doc.Role.Attrs), doc.Role.Body
	case poml.ElementTask:
		if el.Index >= 0 && el.Index < len(doc.Tasks) {
			t := doc.Tasks[el.Index]
			return attrsToMap(t.Attrs), t.Body
		}
	case poml.ElementHumanMsg, poml.ElementAssistantMsg, poml.ElementSystemMsg:
		if el.Index >= 0 && el.Index < len(doc.Messages) {
			m := doc.Messages[el.Index]
			return attrsToMap(m.Attrs), m.Body
		}
	case poml.ElementInput:
		if el.Index >= 0 && el.Index < len(doc.Inputs) {
			in := doc.Inputs[el.Index]
			attrs := attrsToMap(in.Attrs)
			attrs["name"] = in.Name
			attrs["required"] = strconv.FormatBool(in.Required)
			return attrs, in.Body
		}
	case poml.ElementHint:
		if el.Index >= 0 && el.Index < len(doc.Hints) {
			h := doc.Hints[el.Index]
			return attrsToMap(h.Attrs), h.Body
		}
	case poml.ElementExample:
		if el.Index >= 0 && el.Index < len(doc.Examples) {
			e := doc.Examples[el.Index]
			return attrsToMap(e.Attrs), e.Body
		}
	case poml.ElementContentPart:
		if el.Index >= 0 && el.Index < len(doc.ContentParts) {
			c := doc.ContentParts[el.Index]
			return attrsToMap(c.Attrs), c.Body
		}
	case poml.ElementObject:
		if el.Index >= 0 && el.Index < len(doc.Objects) {
			o := doc.Objects[el.Index]
			attrs := attrsToMap(o.Attrs)
			attrs["data"] = o.Data
			attrs["syntax"] = o.Syntax
			return attrs, o.Body
		}
	case poml.ElementDocument:
		if el.Index >= 0 && el.Index < len(doc.Documents) {
			d := doc.Documents[el.Index]
			attrs := attrsToMap(d.Attrs)
			attrs["src"] = d.Src
			return attrs, ""
		}
	case poml.ElementImage:
		if el.Index >= 0 && el.Index < len(doc.Images) {
			im := doc.Images[el.Index]
			attrs := attrsToMap(im.Attrs)
			attrs["src"] = im.Src
			attrs["alt"] = im.Alt
			attrs["syntax"] = im.Syntax
			return attrs, im.Body
		}
	case poml.ElementAudio, poml.ElementVideo:
		var media poml.Media
		switch el.Type {
		case poml.ElementAudio:
			if el.Index >= 0 && el.Index < len(doc.Audios) {
				media = doc.Audios[el.Index]
			}
		case poml.ElementVideo:
			if el.Index >= 0 && el.Index < len(doc.Videos) {
				media = doc.Videos[el.Index]
			}
		}
		if media.Src != "" || media.Body != "" {
			attrs := attrsToMap(media.Attrs)
			attrs["src"] = media.Src
			attrs["alt"] = media.Alt
			attrs["syntax"] = media.Syntax
			return attrs, media.Body
		}
	case poml.ElementToolDefinition:
		if el.Index >= 0 && el.Index < len(doc.ToolDefs) {
			td := doc.ToolDefs[el.Index]
			attrs := attrsToMap(td.Attrs)
			attrs["name"] = td.Name
			attrs["description"] = td.Description
			return attrs, td.Body
		}
	case poml.ElementToolRequest:
		if el.Index >= 0 && el.Index < len(doc.ToolReqs) {
			tr := doc.ToolReqs[el.Index]
			attrs := attrsToMap(tr.Attrs)
			attrs["id"] = tr.ID
			attrs["name"] = tr.Name
			attrs["parameters"] = tr.Parameters
			return attrs, ""
		}
	case poml.ElementToolResponse:
		if el.Index >= 0 && el.Index < len(doc.ToolResps) {
			tr := doc.ToolResps[el.Index]
			attrs := attrsToMap(tr.Attrs)
			attrs["id"] = tr.ID
			attrs["name"] = tr.Name
			return attrs, tr.Body
		}
	case poml.ElementToolResult:
		if el.Index >= 0 && el.Index < len(doc.ToolResults) {
			tr := doc.ToolResults[el.Index]
			attrs := attrsToMap(tr.Attrs)
			attrs["id"] = tr.ID
			attrs["name"] = tr.Name
			return attrs, tr.Body
		}
	case poml.ElementToolError:
		if el.Index >= 0 && el.Index < len(doc.ToolErrors) {
			te := doc.ToolErrors[el.Index]
			attrs := attrsToMap(te.Attrs)
			attrs["id"] = te.ID
			attrs["name"] = te.Name
			return attrs, te.Body
		}
	case poml.ElementOutputSchema:
		return attrsToMap(doc.Schema.Attrs), doc.Schema.Body
	case poml.ElementStyle:
		if el.Index >= 0 && el.Index < len(doc.Styles) {
			st := doc.Styles[el.Index]
			return attrsToMap(st.Attrs), ""
		}
	case poml.ElementOutputFormat:
		if el.Index >= 0 && el.Index < len(doc.OutFormats) {
			of := doc.OutFormats[el.Index]
			return attrsToMap(of.Attrs), of.Body
		}
	case poml.ElementRuntime:
		if el.Index >= 0 && el.Index < len(doc.Runtimes) {
			rt := doc.Runtimes[el.Index]
			return attrsToMap(rt.Attrs), ""
		}
	case poml.ElementDiagram:
		if el.Index >= 0 && el.Index < len(doc.Diagrams) {
			d := doc.Diagrams[el.Index]
			attrs := attrsToMap(d.Attrs)
			attrs["id"] = d.ID
			return attrs, ""
		}
	}
	return map[string]string{}, ""
}

func attrsToMap(attrs []xml.Attr) map[string]string {
	out := make(map[string]string)
	for _, a := range attrs {
		out[a.Name.Local] = a.Value
	}
	return out
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

func (s *Server) tools(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{
		"tool_definitions": s.doc.ToolDefs,
		"tool_requests":    s.doc.ToolReqs,
		"tool_responses":   s.doc.ToolResps,
		"tool_results":     s.doc.ToolResults,
		"tool_errors":      s.doc.ToolErrors,
	})
}

func (s *Server) diagram(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, s.doc.Diagrams)
}

func (s *Server) roundtrip(w http.ResponseWriter, _ *http.Request) {
	ctx, span := s.tracer.Start(context.Background(), "mcp.roundtrip")
	defer span.End()
	var buf strings.Builder
	if err := s.doc.EncodeWithOptions(&buf, poml.EncodeOptions{IncludeHeader: false, PreserveOrder: true, PreserveWS: true}); err != nil {
		http.Error(w, fmt.Sprintf("encode error: %v", err), http.StatusInternalServerError)
		return
	}
	encoded := buf.String()
	doc2, err := poml.ParseStringWithTrace(ctx, encoded, poml.TraceOptions{TracerProvider: span.TracerProvider()})
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("parse after encode failed: %v", err)})
		return
	}
	var buf2 strings.Builder
	if err := doc2.EncodeWithOptions(&buf2, poml.EncodeOptions{IncludeHeader: false, PreserveOrder: true, PreserveWS: true}); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("re-encode failed: %v", err)})
		return
	}
	ok := encoded == buf2.String()
	writeJSON(w, map[string]any{
		"ok":        ok,
		"original":  encoded,
		"roundtrip": buf2.String(),
	})
}

func (s *Server) diff(w http.ResponseWriter, r *http.Request) {
	var body struct {
		A string `json:"a"`
		B string `json:"b"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	docA, err := poml.ParseString(body.A)
	if err != nil {
		http.Error(w, fmt.Sprintf("parse A: %v", err), http.StatusBadRequest)
		return
	}
	docB, err := poml.ParseString(body.B)
	if err != nil {
		http.Error(w, fmt.Sprintf("parse B: %v", err), http.StatusBadRequest)
		return
	}
	equal := docA.Meta == docB.Meta && len(docA.Elements) == len(docB.Elements)
	writeJSON(w, map[string]any{
		"equal_meta":   docA.Meta == docB.Meta,
		"equal_counts": len(docA.Elements) == len(docB.Elements),
		"meta_a":       docA.Meta,
		"meta_b":       docB.Meta,
		"count_a":      len(docA.Elements),
		"count_b":      len(docB.Elements),
		"approx_equal": equal,
	})
}

func (s *Server) patch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Tag   string `json:"tag"`
		Index int    `json:"index"`
		Body  string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	doc := s.doc
	updated := false
	switch poml.ElementType(req.Tag) {
	case poml.ElementRole:
		doc.Role.Body = req.Body
		updated = true
	case poml.ElementTask:
		if req.Index >= 0 && req.Index < len(doc.Tasks) {
			doc.Tasks[req.Index].Body = req.Body
			updated = true
		}
	case poml.ElementHumanMsg, poml.ElementAssistantMsg, poml.ElementSystemMsg:
		if req.Index >= 0 && req.Index < len(doc.Messages) {
			doc.Messages[req.Index].Body = req.Body
			updated = true
		}
	}
	if !updated {
		http.Error(w, "unsupported tag or index", http.StatusBadRequest)
		return
	}
	var buf strings.Builder
	if err := doc.EncodeWithOptions(&buf, poml.EncodeOptions{IncludeHeader: false, PreserveOrder: true, PreserveWS: true}); err != nil {
		http.Error(w, fmt.Sprintf("encode error: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{
		"ok":    true,
		"poml":  buf.String(),
		"tag":   req.Tag,
		"index": req.Index,
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
