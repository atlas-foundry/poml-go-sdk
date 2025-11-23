package poml

import (
	"os"
	"path/filepath"
	"strings"
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
	if resolved != target && !strings.HasSuffix(resolved, "media.bin") {
		t.Fatalf("expected resolved path ending with media.bin, got %s", resolved)
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
	if resolved != target && !strings.HasSuffix(resolved, "media.bin") {
		t.Fatalf("unexpected resolved path: %s", resolved)
	}
}

func TestResolveMediaPathRejectsAbsoluteWhenDisabled(t *testing.T) {
	target := filepath.Join(t.TempDir(), "media.bin")
	if err := os.WriteFile(target, []byte("ok"), 0o600); err != nil {
		t.Fatalf("write media: %v", err)
	}
	if _, err := resolveMediaPath(target, ConvertOptions{AllowAbsImagePaths: false}); err == nil {
		t.Fatalf("expected absolute path rejection")
	}
}

func TestResolveImagePathRejectsSymlinkEscape(t *testing.T) {
	base := t.TempDir()
	outside := t.TempDir()
	target := filepath.Join(outside, "file.bin")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	link := filepath.Join(base, "link.bin")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if _, err := resolveImagePath("link.bin", ConvertOptions{BaseDir: base}); err == nil {
		t.Fatalf("expected symlink escape to be rejected")
	}
}

func TestResolveImagePathRejectsBaseSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	actual := filepath.Join(root, "real")
	if err := os.MkdirAll(actual, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("mkdir outside: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outside, "steal.bin"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	baseLink := filepath.Join(root, "base-link")
	if err := os.Symlink(actual, baseLink); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	if err := os.Symlink(filepath.Join("..", "outside", "steal.bin"), filepath.Join(actual, "escape.bin")); err != nil {
		t.Fatalf("symlink escape: %v", err)
	}
	if _, err := resolveImagePath("escape.bin", ConvertOptions{BaseDir: baseLink}); err == nil {
		t.Fatalf("expected escape through base symlink to be rejected")
	}
}
