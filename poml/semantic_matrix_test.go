package poml

import (
	"os"
	"path/filepath"
	"testing"
)

// Semantic-style integration to ensure BaseDir and media caps behave for full conversion.
func TestSemanticConvertWithBaseDirAndCaps(t *testing.T) {
	base := t.TempDir()
	imgPath := filepath.Join(base, "img.bin")
	audioPath := filepath.Join(base, "audio.bin")
	videoPath := filepath.Join(base, "video.bin")
	write := func(path string, size int) {
		if err := os.WriteFile(path, bytesOf('a', size), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	write(imgPath, 8)
	write(audioPath, 12)
	write(videoPath, 16)

	doc := Document{
		Meta: Meta{ID: "semantic.base", Version: "1", Owner: "oss"},
		Role: Block{Body: "r"},
		Tasks: []Block{
			{Body: "t"},
		},
		Images: []Image{
			{Src: "img.bin", Alt: "example"},
		},
		Audios: []Media{
			{Src: "audio.bin", Syntax: "audio/wav"},
		},
		Videos: []Media{
			{Src: "video.bin", Syntax: "video/mp4"},
		},
		Elements: []Element{
			{Type: ElementMeta},
			{Type: ElementRole},
			{Type: ElementTask, Index: 0},
			{Type: ElementImage, Index: 0},
			{Type: ElementAudio, Index: 0},
			{Type: ElementVideo, Index: 0},
		},
	}

	optsOK := ConvertOptions{BaseDir: base, MaxImageBytes: 16, MaxMediaBytes: 32}
	if _, err := Convert(doc, FormatMessageDict, optsOK); err != nil {
		t.Fatalf("convert with ok limits: %v", err)
	}

	optsTight := ConvertOptions{BaseDir: base, MaxImageBytes: 4, MaxMediaBytes: 8}
	if _, err := Convert(doc, FormatMessageDict, optsTight); err == nil {
		t.Fatalf("expected convert to fail under tight limits")
	}

	// Negative limits disable caps and should permit oversized assets.
	optsUnlimited := ConvertOptions{BaseDir: base, MaxImageBytes: -1, MaxMediaBytes: -1}
	if _, err := Convert(doc, FormatMessageDict, optsUnlimited); err != nil {
		t.Fatalf("convert with unlimited caps: %v", err)
	}
}

func TestSemanticBaseDirEscapeRejected(t *testing.T) {
	base := t.TempDir()
	outside := t.TempDir()
	evil := filepath.Join(outside, "evil.bin")
	if err := os.WriteFile(evil, []byte("nope"), 0o600); err != nil {
		t.Fatalf("write evil: %v", err)
	}

	doc := Document{
		Meta:   Meta{ID: "semantic.escape", Version: "1", Owner: "oss"},
		Role:   Block{Body: "r"},
		Tasks:  []Block{{Body: "t"}},
		Images: []Image{{Src: "../evil.bin"}},
		Elements: []Element{
			{Type: ElementMeta},
			{Type: ElementRole},
			{Type: ElementTask, Index: 0},
			{Type: ElementImage, Index: 0},
		},
	}
	if _, err := Convert(doc, FormatMessageDict, ConvertOptions{BaseDir: base}); err == nil {
		t.Fatalf("expected escape to be rejected")
	}
}

func bytesOf(ch byte, n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = ch
	}
	return b
}

// Fixture-based semantic test to ensure BaseDir + caps work on disk assets.
func TestSemanticFixtureAssets(t *testing.T) {
	doc, err := ParseFile(filepath.Join("testdata", "semantic", "base_dir_ok.poml"))
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	base := filepath.Join("testdata", "semantic")
	opts := ConvertOptions{BaseDir: base, MaxImageBytes: 64, MaxMediaBytes: 64}
	if _, err := Convert(doc, FormatMessageDict, opts); err != nil {
		t.Fatalf("convert fixture: %v", err)
	}
	optsTight := ConvertOptions{BaseDir: base, MaxImageBytes: 8, MaxMediaBytes: 8}
	if _, err := Convert(doc, FormatMessageDict, optsTight); err == nil {
		t.Fatalf("expected tight caps to fail for fixture")
	}
}
