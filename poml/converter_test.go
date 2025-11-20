package poml

import "testing"

func TestConvertNotImplemented(t *testing.T) {
	doc := Document{}
	if _, err := Convert(doc, FormatMessageDict, ConvertOptions{}); err != ErrNotImplemented {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}
