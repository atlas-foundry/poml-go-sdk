package mcp

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/atlas-foundry/poml-go-sdk/poml"
	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// Server exposes a minimal MCP-style HTTP surface to inspect parsed POML docs.
// It is intentionally small: a health endpoint, an AST dump, and a summary.
type Server struct {
	doc        poml.Document
	mux        *http.ServeMux
	once       sync.Once
	tracer     trace.Tracer
	sourcePath string
	mu         sync.RWMutex
	watcher    *fsnotify.Watcher
	wsMu       sync.Mutex
	wsClients  map[*websocket.Conn]struct{}
}

type inspectSummary struct {
	Meta    poml.Meta          `json:"meta"`
	Counts  map[string]int     `json:"counts"`
	Created time.Time          `json:"created"`
	IDs     []string           `json:"element_ids,omitempty"`
	Types   []poml.ElementType `json:"element_types,omitempty"`
}

// New creates a server for the given document. sourcePath enables watch/reload when set.
func New(doc poml.Document, sourcePath string, tp trace.TracerProvider) *Server {
	if tp == nil {
		tp = noop.NewTracerProvider()
	}
	s := &Server{
		doc:        doc,
		mux:        http.NewServeMux(),
		tracer:     tp.Tracer("github.com/atlas-foundry/poml-go-sdk/mcp"),
		sourcePath: sourcePath,
		wsClients:  make(map[*websocket.Conn]struct{}),
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
	s.mux.HandleFunc("/watch", s.watch)
	s.mux.HandleFunc("/ws", s.ws)
	if sourcePath != "" {
		if w, err := fsnotify.NewWatcher(); err == nil {
			_ = w.Add(sourcePath)
			s.watcher = w
		}
	}
	return s
}

// Handler returns the server's HTTP handler (useful for tests/embedding).
func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) snapshot() poml.Document {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.doc
}

func (s *Server) updateDoc(doc poml.Document) {
	s.mu.Lock()
	s.doc = doc
	s.once = sync.Once{}
	s.mu.Unlock()
	s.broadcastSummary("update")
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) inspect(w http.ResponseWriter, _ *http.Request) {
	doc := s.snapshot()
	writeJSON(w, buildSummary(doc))
}

func (s *Server) ast(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, s.snapshot())
}

func (s *Server) validate(w http.ResponseWriter, _ *http.Request) {
	doc := s.snapshot()
	err := doc.Validate()
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
	doc := s.snapshot()
	out, err := poml.Convert(doc, format, poml.ConvertOptions{})
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

	doc := s.snapshot()
	var matches []map[string]any
	for _, el := range doc.Elements {
		attrs, body := extractAttrsBody(doc, el)
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
	doc := s.snapshot()
	writeJSON(w, map[string]any{
		"tool_definitions": doc.ToolDefs,
		"tool_requests":    doc.ToolReqs,
		"tool_responses":   doc.ToolResps,
		"tool_results":     doc.ToolResults,
		"tool_errors":      doc.ToolErrors,
	})
}

func (s *Server) diagram(w http.ResponseWriter, _ *http.Request) {
	doc := s.snapshot()
	writeJSON(w, doc.Diagrams)
}

func (s *Server) roundtrip(w http.ResponseWriter, _ *http.Request) {
	ctx, span := s.tracer.Start(context.Background(), "mcp.roundtrip")
	defer span.End()
	doc := s.snapshot()
	var buf strings.Builder
	if err := doc.EncodeWithOptions(&buf, poml.EncodeOptions{IncludeHeader: false, PreserveOrder: true, PreserveWS: true}); err != nil {
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
	doc := s.snapshot()
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
	s.updateDoc(doc)
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

func (s *Server) watch(w http.ResponseWriter, r *http.Request) {
	if s.watcher == nil || s.sourcePath == "" {
		http.Error(w, "watch not enabled (no source file)", http.StatusBadRequest)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sendSnapshot := func(event string) {
		doc := s.snapshot()
		payload := map[string]any{
			"event":   event,
			"summary": buildSummary(doc),
		}
		blob, _ := json.Marshal(payload)
		_, _ = fmt.Fprintf(w, "data: %s\n\n", blob)
		flusher.Flush()
	}

	sendSnapshot("initial")

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-s.watcher.Events:
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
				if doc, err := s.reload(); err == nil {
					s.updateDoc(doc)
					sendSnapshot("update")
				} else {
					payload := map[string]any{"event": "error", "error": err.Error()}
					blob, _ := json.Marshal(payload)
					_, _ = fmt.Fprintf(w, "data: %s\n\n", blob)
					flusher.Flush()
				}
			}
		case err := <-s.watcher.Errors:
			payload := map[string]any{"event": "error", "error": err.Error()}
			blob, _ := json.Marshal(payload)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", blob)
			flusher.Flush()
		}
	}
}

func (s *Server) reload() (poml.Document, error) {
	if s.sourcePath == "" {
		return poml.Document{}, fmt.Errorf("no source path")
	}
	data, err := os.ReadFile(s.sourcePath)
	if err != nil {
		return poml.Document{}, err
	}
	doc, err := poml.ParseString(string(data))
	if err != nil {
		return poml.Document{}, err
	}
	if err := doc.Validate(); err != nil {
		return poml.Document{}, err
	}
	return doc, nil
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *Server) ws(w http.ResponseWriter, r *http.Request) {
	if token := os.Getenv("POML_MCP_TOKEN"); token != "" {
		if r.URL.Query().Get("token") != token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("upgrade failed: %v", err), http.StatusBadRequest)
		return
	}
	s.registerWS(conn)
	defer s.unregisterWS(conn)

	// send initial summary + ast
	s.sendWSSummary(conn, "initial", true)

	// heartbeat
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// keep the connection alive; ignore incoming messages
	for {
		select {
		case <-ticker.C:
			_ = conn.WriteJSON(map[string]any{"event": "heartbeat", "ts": time.Now().UTC()})
		default:
			if _, _, err := conn.NextReader(); err != nil {
				return
			}
		}
	}
}

func (s *Server) registerWS(c *websocket.Conn) {
	s.wsMu.Lock()
	s.wsClients[c] = struct{}{}
	s.wsMu.Unlock()
}

func (s *Server) unregisterWS(c *websocket.Conn) {
	s.wsMu.Lock()
	delete(s.wsClients, c)
	s.wsMu.Unlock()
	_ = c.Close()
}

func (s *Server) sendWSSummary(c *websocket.Conn, event string, includeAST bool) {
	doc := s.snapshot()
	payload := map[string]any{
		"event":   event,
		"summary": buildSummary(doc),
	}
	if includeAST {
		payload["ast"] = doc
	}
	_ = c.WriteJSON(payload)
}

func (s *Server) broadcastSummary(event string) {
	s.wsMu.Lock()
	clients := make([]*websocket.Conn, 0, len(s.wsClients))
	for c := range s.wsClients {
		clients = append(clients, c)
	}
	s.wsMu.Unlock()

	for _, c := range clients {
		s.sendWSSummary(c, event, false)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
