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
