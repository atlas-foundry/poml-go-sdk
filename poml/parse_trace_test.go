package poml

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.opentelemetry.io/otel/trace/noop"
)

const minimalPOML = `<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task></poml>`

func TestParseReaderVariants(t *testing.T) {
	if _, err := ParseReaderFast(bytes.NewBufferString(minimalPOML)); err != nil {
		t.Fatalf("ParseReaderFast: %v", err)
	}
	if _, err := ParseReaderStrict(bytes.NewBufferString(minimalPOML)); err != nil {
		t.Fatalf("ParseReaderStrict: %v", err)
	}
	tp := noop.NewTracerProvider()
	opts := TraceOptions{TracerProvider: tp}
	if _, err := ParseReaderWithTrace(context.Background(), bytes.NewBufferString(minimalPOML), opts); err != nil {
		t.Fatalf("ParseReaderWithTrace: %v", err)
	}
}

func TestParseFileVariants(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "doc.poml")
	if err := os.WriteFile(tmp, []byte(minimalPOML), 0o644); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	if _, err := ParseFileStrict(tmp); err != nil {
		t.Fatalf("ParseFileStrict: %v", err)
	}
	if _, err := ParseFileWithTrace(context.Background(), tmp, TraceOptions{TracerProvider: noop.NewTracerProvider()}); err != nil {
		t.Fatalf("ParseFileWithTrace: %v", err)
	}
}

func TestParseStringWithTrace(t *testing.T) {
	if _, err := ParseStringWithTrace(context.Background(), minimalPOML, TraceOptions{TracerProvider: noop.NewTracerProvider()}); err != nil {
		t.Fatalf("ParseStringWithTrace: %v", err)
	}
}
