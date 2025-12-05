package poml

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestTraceRecorderGoldenParseValidateConvert(t *testing.T) {
	rec := NewTraceRecorder("trace-golden-1")
	ctx := context.Background()
	doc, err := ParseStringWithTrace(ctx, minimalPOML, TraceOptions{TracerProvider: rec.Provider})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := doc.ValidateWithTraceOptions(ctx, TraceOptions{TracerProvider: rec.Provider}, ValidateOptions{Extended: ExtendedOff}); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if _, err := ConvertWithTrace(ctx, doc, FormatOpenAIChat, ConvertOptions{Trace: TraceOptions{TracerProvider: rec.Provider}}); err != nil {
		t.Fatalf("convert: %v", err)
	}
	got := rec.Dump()

	// Sort spans by name for deterministic comparison
	sort.Slice(got, func(i, j int) bool { return got[i].Name < got[j].Name })

	goldenPath := filepath.Join("testdata", "traces", "parse_validate_convert.json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		// Sort before writing golden file too
		if err := rec.DumpToFile(goldenPath); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	wantRaw, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var want []SpanDump
	if err := json.Unmarshal(wantRaw, &want); err != nil {
		t.Fatalf("unmarshal golden: %v", err)
	}

	// Sort want spans too
	sort.Slice(want, func(i, j int) bool { return want[i].Name < want[j].Name })

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("trace dump mismatch (-want +got):\n%s", diff)
	}
}

// TestTraceAllFormats verifies that all output formats produce appropriate spans.
func TestTraceAllFormats(t *testing.T) {
	formats := []Format{
		FormatMessageDict,
		FormatDict,
		FormatOpenAIChat,
		FormatLangChain,
		FormatPydantic,
	}

	for _, format := range formats {
		t.Run(string(format), func(t *testing.T) {
			rec := NewTraceRecorder("trace-format-" + string(format))
			ctx := context.Background()
			doc, err := ParseString(minimalPOML)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			_, err = ConvertWithTrace(ctx, doc, format, ConvertOptions{
				Trace: TraceOptions{TracerProvider: rec.Provider},
			})
			if err != nil {
				t.Fatalf("convert: %v", err)
			}

			spans := rec.Dump()
			if len(spans) == 0 {
				t.Fatal("expected spans to be recorded")
			}

			// Verify we have the root convert span
			hasRoot := false
			hasFormatSpan := false
			hasTemplateSpan := false
			expectedFormatSpan := "poml.convert." + string(format)

			for _, span := range spans {
				switch span.Name {
				case "poml.convert":
					hasRoot = true
				case expectedFormatSpan:
					hasFormatSpan = true
				case "poml.template.expand":
					hasTemplateSpan = true
				}
			}

			if !hasRoot {
				t.Error("missing poml.convert root span")
			}
			if !hasFormatSpan {
				t.Errorf("missing %s format span", expectedFormatSpan)
			}
			if !hasTemplateSpan {
				t.Error("missing poml.template.expand span")
			}
		})
	}
}

// TestTraceTemplateExpansion verifies template-related child spans.
func TestTraceTemplateExpansion(t *testing.T) {
	pomlWithLet := `<poml>
		<meta><id>trace.template</id><version>1</version><owner>test</owner></meta>
		<let name="greeting" value="Hello"/>
		<let name="name" value="World"/>
		<role>{{ greeting }}, {{ name }}!</role>
	</poml>`

	rec := NewTraceRecorder("trace-template")
	ctx := context.Background()
	doc, err := ParseString(pomlWithLet)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	_, err = ConvertWithTrace(ctx, doc, FormatMessageDict, ConvertOptions{
		Trace:           TraceOptions{TracerProvider: rec.Provider},
		ExpandTemplates: true,
		TemplateVars:    map[string]any{"extra": "value"},
	})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}

	spans := rec.Dump()

	// Find template-related spans
	hasLetSpan := false
	hasInterpolateSpan := false
	var letBindingCount int

	for _, span := range spans {
		switch span.Name {
		case "poml.template.let":
			hasLetSpan = true
			for _, attr := range span.Attributes {
				if attr.Key == "poml.binding.count" {
					if v, ok := attr.Value.(float64); ok {
						letBindingCount = int(v)
					}
				}
			}
		case "poml.template.interpolate":
			hasInterpolateSpan = true
		}
	}

	if !hasLetSpan {
		t.Error("missing poml.template.let span")
	}
	if !hasInterpolateSpan {
		t.Error("missing poml.template.interpolate span")
	}
	// 2 let bindings + 1 template var
	if letBindingCount != 3 {
		t.Errorf("expected 3 bindings, got %d", letBindingCount)
	}
}

// TestTraceParentChildRelationships verifies parent-child span relationships.
func TestTraceParentChildRelationships(t *testing.T) {
	rec := NewTraceRecorder("trace-hierarchy")
	ctx := context.Background()
	doc, err := ParseString(minimalPOML)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	_, err = ConvertWithTrace(ctx, doc, FormatOpenAIChat, ConvertOptions{
		Trace: TraceOptions{TracerProvider: rec.Provider},
	})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}

	spans := rec.Dump()

	// Build a map of span IDs to spans
	spanByID := make(map[string]SpanDump)
	var rootSpan SpanDump
	for _, span := range spans {
		spanByID[span.SpanID] = span
		if span.Name == "poml.convert" {
			rootSpan = span
		}
	}

	if rootSpan.SpanID == "" {
		t.Fatal("no root poml.convert span found")
	}

	// Verify child spans have correct parent
	childSpans := []string{"poml.template.expand", "poml.convert.openai_chat"}
	for _, childName := range childSpans {
		found := false
		for _, span := range spans {
			if span.Name == childName {
				found = true
				if span.ParentSpanID != rootSpan.SpanID {
					t.Errorf("%s should have parent %s, got %s", childName, rootSpan.SpanID, span.ParentSpanID)
				}
				break
			}
		}
		if !found {
			t.Errorf("expected child span %s not found", childName)
		}
	}
}

// TestTraceErrorRecording verifies errors are recorded in spans.
func TestTraceErrorRecording(t *testing.T) {
	invalidPOML := `<poml>
		<meta><id>trace.error</id><version>99.0</version><owner>test</owner>
			<minVersion>99.0</minVersion>
		</meta>
		<role>test</role>
	</poml>`

	rec := NewTraceRecorder("trace-error")
	ctx := context.Background()
	doc, err := ParseString(invalidPOML)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	// This should fail due to version constraint
	_, err = ConvertWithTrace(ctx, doc, FormatMessageDict, ConvertOptions{
		Trace:           TraceOptions{TracerProvider: rec.Provider},
		EnforceVersions: true,
	})

	if err == nil {
		t.Fatal("expected version constraint error")
	}

	spans := rec.Dump()

	// Find the convert span and check for error attribute
	for _, span := range spans {
		if span.Name == "poml.convert" {
			hasErrorType := false
			for _, attr := range span.Attributes {
				if attr.Key == "poml.error.type" {
					hasErrorType = true
					if attr.Value != "version" {
						t.Errorf("expected error type 'version', got %v", attr.Value)
					}
				}
			}
			if !hasErrorType {
				t.Error("expected poml.error.type attribute on failed convert span")
			}
			return
		}
	}
	t.Error("no poml.convert span found")
}
