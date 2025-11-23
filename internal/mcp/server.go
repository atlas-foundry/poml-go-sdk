package mcp

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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
	s.mux.HandleFunc("/validate", s.validate)
	s.mux.HandleFunc("/convert", s.convert)
	s.mux.HandleFunc("/search", s.search)
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

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
