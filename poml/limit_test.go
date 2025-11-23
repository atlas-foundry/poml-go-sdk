package poml

import (
	"encoding/base64"
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

func TestBuildImagePartRespectsSizeLimit(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/img.bin"
	if err := os.WriteFile(path, []byte("12345"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := buildImagePart(Image{Src: path}, ConvertOptions{MaxImageBytes: 3, AllowAbsImagePaths: true}); err == nil {
		t.Fatalf("expected size limit error")
	}
	part, err := buildImagePart(Image{Src: path}, ConvertOptions{MaxImageBytes: 10, AllowAbsImagePaths: true})
	if err != nil {
		t.Fatalf("unexpected error with higher limit: %v", err)
	}
	if part["base64"] == "" {
		t.Fatalf("expected base64 payload")
	}
}

func TestBuildMediaPartRespectsSizeLimit(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/audio.bin"
	if err := os.WriteFile(path, []byte("12345"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := buildMediaPart(Media{Src: path}, ConvertOptions{MaxMediaBytes: 3, AllowAbsImagePaths: true}); err == nil {
		t.Fatalf("expected size limit error")
	}
	part, err := buildMediaPart(Media{Src: path}, ConvertOptions{MaxMediaBytes: 10, AllowAbsImagePaths: true})
	if err != nil {
		t.Fatalf("unexpected error with higher limit: %v", err)
	}
	if part["base64"] == "" {
		t.Fatalf("expected base64 payload")
	}
}

func TestMediaDataURIRespectsLimit(t *testing.T) {
	payload := base64.StdEncoding.EncodeToString([]byte{0x01, 0x02, 0x03, 0x04})
	dataURI := "data:audio/wav;base64," + payload
	if _, err := buildMediaPart(Media{Src: dataURI}, ConvertOptions{MaxMediaBytes: 3}); err == nil {
		t.Fatalf("expected data URI to be size-checked")
	}

	smallPayload := base64.StdEncoding.EncodeToString([]byte{0x01, 0x02})
	smallDataURI := "data:audio/wav;base64," + smallPayload
	if _, err := buildMediaPart(Media{Src: smallDataURI}, ConvertOptions{MaxMediaBytes: 3}); err != nil {
		t.Fatalf("small data URI should pass: %v", err)
	}
}
