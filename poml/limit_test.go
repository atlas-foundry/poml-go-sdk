package poml

import (
	"os"
	"testing"
)

func TestReadFileWithLimit(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/sample.bin"
	if err := os.WriteFile(path, []byte("12345"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	data, err := readFileWithLimit(path, 5)
	if err != nil {
		t.Fatalf("unexpected error under limit: %v", err)
	}
	if string(data) != "12345" {
		t.Fatalf("data mismatch: %q", string(data))
	}

	if _, err := readFileWithLimit(path, 4); err == nil {
		t.Fatalf("expected error when over limit")
	}
}
