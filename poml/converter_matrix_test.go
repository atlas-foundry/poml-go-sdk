package poml

import (
	"errors"
	"path/filepath"
	"testing"
)

// Converter matrix: ensure key formats succeed on a media-bearing document.
func TestConverterMatrixFormats(t *testing.T) {
	doc, err := ParseFile(filepath.Join("testdata", "semantic", "base_dir_ok.poml"))
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	opts := ConvertOptions{
		BaseDir:            filepath.Join("testdata", "semantic"),
		MaxImageBytes:      1 << 20,
		MaxMediaBytes:      1 << 20,
		AllowAbsImagePaths: false,
	}
	formats := []Format{
		FormatMessageDict,
		FormatDict,
		FormatOpenAIChat,
		FormatLangChain,
		FormatPydantic,
	}
	for _, f := range formats {
		t.Run(string(f), func(t *testing.T) {
			if _, err := Convert(doc, f, opts); err != nil {
				t.Fatalf("convert %s: %v", f, err)
			}
		})
	}
}

func TestConverterMatrixFailureCases(t *testing.T) {
	doc, err := ParseFile(filepath.Join("testdata", "semantic", "base_dir_ok.poml"))
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	// Missing BaseDir for file-backed media should error.
	if _, err := Convert(doc, FormatMessageDict, ConvertOptions{}); err == nil {
		t.Fatalf("expected convert to fail without BaseDir for media")
	}
	// Unknown format should return ErrNotImplemented.
	if _, err := Convert(doc, Format("unknown_format"), ConvertOptions{}); !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented for unknown format, got %v", err)
	}
}
