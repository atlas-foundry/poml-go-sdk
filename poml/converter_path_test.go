package poml

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveMediaPathWithinBaseDir(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "media.bin")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatalf("write media: %v", err)
	}

	resolved, err := resolveMediaPath("media.bin", ConvertOptions{BaseDir: base})
	if err != nil {
		t.Fatalf("resolve media: %v", err)
	}
	if resolved != target {
		t.Fatalf("expected resolved path %s, got %s", target, resolved)
	}

	if _, err := resolveMediaPath("../escape.bin", ConvertOptions{BaseDir: base}); err == nil {
		t.Fatalf("expected escape attempt to error")
	}
}

func TestResolveMediaPathAllowsAbsoluteWhenEnabled(t *testing.T) {
	target := filepath.Join(t.TempDir(), "media.bin")
	if err := os.WriteFile(target, []byte("ok"), 0o600); err != nil {
		t.Fatalf("write media: %v", err)
	}
	resolved, err := resolveMediaPath(target, ConvertOptions{AllowAbsImagePaths: true})
	if err != nil {
		t.Fatalf("resolve abs: %v", err)
	}
	if resolved != target {
		t.Fatalf("unexpected resolved path: %s", resolved)
	}
}
