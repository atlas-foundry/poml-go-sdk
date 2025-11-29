package poml

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
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

	goldenPath := filepath.Join("testdata", "traces", "parse_validate_convert.json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
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

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("trace dump mismatch (-want +got):\n%s", diff)
	}
}
