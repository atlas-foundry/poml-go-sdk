package poml

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// Boundary tests for media/image limits and path safety.
func TestBuildMediaPartAppliesDefaultLimit(t *testing.T) {
	tmp := t.TempDir()
	blob := bytes.Repeat([]byte("a"), int(defaultMaxMediaBytes)+1)
	path := filepath.Join(tmp, "big.bin")
	if err := os.WriteFile(path, blob, 0o600); err != nil {
		t.Fatalf("write big media: %v", err)
	}
	if _, err := buildMediaPart(Media{Src: path}, ConvertOptions{MaxMediaBytes: 0, AllowAbsImagePaths: true}); err == nil {
		t.Fatalf("expected default limit (%d bytes) to reject oversized media", defaultMaxMediaBytes)
	}
}

func TestBuildMediaPartNegativeLimitAllowsLarge(t *testing.T) {
	tmp := t.TempDir()
	blob := bytes.Repeat([]byte("b"), int(defaultMaxMediaBytes)+1)
	path := filepath.Join(tmp, "large.bin")
	if err := os.WriteFile(path, blob, 0o600); err != nil {
		t.Fatalf("write large media: %v", err)
	}
	if _, err := buildMediaPart(Media{Src: path}, ConvertOptions{MaxMediaBytes: -1, AllowAbsImagePaths: true}); err != nil {
		t.Fatalf("expected negative limit to allow large media: %v", err)
	}
}

func TestResolveMediaPathSymlinkEscape(t *testing.T) {
	base := t.TempDir()
	outside := t.TempDir()
	target := filepath.Join(outside, "media.bin")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	link := filepath.Join(base, "media.bin")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if _, err := resolveMediaPath("media.bin", ConvertOptions{BaseDir: base}); err == nil {
		t.Fatalf("expected symlink escape to be rejected for media")
	}
}
