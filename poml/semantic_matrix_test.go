package poml

import (
	"os"
	"path/filepath"
	"strings"
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

// Extended/unknown fixture conversion to ensure unknown tags flow through message dict when Extended is enabled.
func TestSemanticFixtureExtendedUnknowns(t *testing.T) {
	p := filepath.Join("testdata", "semantic", "mixed_extended.poml")
	body, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	doc, err := ParseReaderWithOptions(strings.NewReader(string(body)), ParseOptions{
		PreserveWhitespace: true,
		Validate:           false,
		Extended:           ExtendedLenient,
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, err := Convert(doc, FormatMessageDict, ConvertOptions{Extended: ExtendedLenient})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	msgs, ok := out.([]messageDict)
	if !ok {
		t.Fatalf("unexpected type %T", out)
	}
	foundUnknown := false
	for _, m := range msgs {
		if payload, ok := m.Content.(map[string]any); ok && payload["type"] == "unknown" {
			if payload["name"] == "custom" {
				foundUnknown = true
			}
		}
	}
	if !foundUnknown {
		t.Fatalf("expected unknown element to be surfaced in message dict")
	}
}

func TestSemanticExtendedNestedUnknowns(t *testing.T) {
	p := filepath.Join("testdata", "semantic", "mixed_extended_nested.poml")
	doc, err := ParseReaderWithOptions(strings.NewReader(readFile(t, p)), ParseOptions{
		PreserveWhitespace: true,
		Validate:           false,
		Extended:           ExtendedStrict,
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// Extended off should drop unknowns in conversion.
	if out, err := Convert(doc, FormatMessageDict, ConvertOptions{Extended: ExtendedOff}); err != nil {
		t.Fatalf("convert extended off: %v", err)
	} else if msgs, ok := out.([]messageDict); ok {
		for _, m := range msgs {
			if payload, ok := m.Content.(map[string]any); ok && payload["type"] == "unknown" {
				t.Fatalf("expected unknown to be omitted when ExtendedOff")
			}
		}
	}
	// Extended lenient should surface unknown payloads.
	out, err := Convert(doc, FormatMessageDict, ConvertOptions{Extended: ExtendedLenient})
	if err != nil {
		t.Fatalf("convert extended lenient: %v", err)
	}
	msgs, ok := out.([]messageDict)
	if !ok {
		t.Fatalf("unexpected type %T", out)
	}
	found := false
	for _, m := range msgs {
		if payload, ok := m.Content.(map[string]any); ok && payload["type"] == "unknown" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected unknown element to be surfaced in extended lenient")
	}
}

// Table-driven semantic coverage across path/limit/schema/runtime axes.
func TestSemanticMatrixFixtures(t *testing.T) {
	cases := []struct {
		name      string
		file      string
		opts      ConvertOptions
		expectErr bool
	}{
		{
			name:      "base-dir-escape",
			file:      filepath.Join("testdata", "semantic", "base_dir_escape.poml"),
			opts:      ConvertOptions{BaseDir: "testdata/semantic"},
			expectErr: true,
		},
		{
			name:      "base-dir-absolute-disallowed",
			file:      filepath.Join("testdata", "semantic", "base_dir_absolute.poml"),
			opts:      ConvertOptions{BaseDir: "testdata/semantic", AllowAbsImagePaths: false},
			expectErr: true,
		},
		{
			name: "runtime-schema",
			file: filepath.Join("testdata", "semantic", "runtime_schema.poml"),
			opts: ConvertOptions{BaseDir: "testdata/semantic"},
		},
		{
			name:      "image-too-large",
			file:      filepath.Join("testdata", "semantic", "base_dir_ok.poml"),
			opts:      ConvertOptions{BaseDir: "testdata/semantic", MaxImageBytes: 8, MaxMediaBytes: 8},
			expectErr: true,
		},
		{
			name:      "abs-image-disallowed",
			file:      filepath.Join("testdata", "semantic", "base_dir_absolute.poml"),
			opts:      ConvertOptions{AllowAbsImagePaths: false},
			expectErr: true,
		},
		{
			name:      "unc-path-disallowed",
			file:      filepath.Join("testdata", "semantic", "base_dir_unc.poml"),
			opts:      ConvertOptions{AllowAbsImagePaths: false},
			expectErr: true,
		},
		{
			name: "malformed-data-uri",
			file: filepath.Join("testdata", "semantic", "malformed_data_uri.poml"),
			opts: ConvertOptions{BaseDir: "testdata/semantic"},
		},
		{
			name: "runtime-schema-multi",
			file: filepath.Join("testdata", "semantic", "runtime_schema_multi.poml"),
			opts: ConvertOptions{BaseDir: "testdata/semantic"},
		},
		{
			name: "media-mime-odd",
			file: filepath.Join("testdata", "semantic", "media_mime_odd.poml"),
			opts: ConvertOptions{BaseDir: "testdata/semantic", MaxImageBytes: 1 << 20, MaxMediaBytes: 1 << 20},
		},
		{
			name: "media-empty-body",
			file: filepath.Join("testdata", "semantic", "media_empty_body.poml"),
			opts: ConvertOptions{BaseDir: "testdata/semantic"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := ParseFile(tc.file)
			if err != nil {
				if tc.expectErr {
					return
				}
				t.Fatalf("parse: %v", err)
			}
			_, err = Convert(doc, FormatDict, tc.opts)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("convert: %v", err)
			}
		})
	}
}

func TestSemanticAbsoluteAllowedWithFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "img.bin")
	if err := os.WriteFile(path, []byte("hi"), 0o600); err != nil {
		t.Fatalf("write abs: %v", err)
	}
	doc := Document{
		Meta:     Meta{ID: "abs", Version: "1", Owner: "oss"},
		Role:     Block{Body: "r"},
		Tasks:    []Block{{Body: "t"}},
		Images:   []Image{{Src: path, Alt: "abs"}},
		Elements: []Element{{Type: ElementMeta}, {Type: ElementRole}, {Type: ElementTask, Index: 0}, {Type: ElementImage, Index: 0}},
	}
	if _, err := Convert(doc, FormatDict, ConvertOptions{AllowAbsImagePaths: true}); err != nil {
		t.Fatalf("convert abs allowed: %v", err)
	}
}

func TestSemanticInlineImageLimit(t *testing.T) {
	doc := Document{
		Meta:     Meta{ID: "inline", Version: "1", Owner: "oss"},
		Role:     Block{Body: "r"},
		Tasks:    []Block{{Body: "t"}},
		Images:   []Image{{Body: "0123456789", Alt: "inline"}},
		Elements: []Element{{Type: ElementMeta}, {Type: ElementRole}, {Type: ElementTask, Index: 0}, {Type: ElementImage, Index: 0}},
	}
	if _, err := Convert(doc, FormatDict, ConvertOptions{MaxImageBytes: 5}); err == nil {
		t.Fatalf("expected inline image to exceed limit")
	}
	if _, err := Convert(doc, FormatDict, ConvertOptions{MaxImageBytes: 16}); err != nil {
		t.Fatalf("expected inline image to succeed with higher limit: %v", err)
	}
}

func TestSemanticSymlinkEscape(t *testing.T) {
	base := t.TempDir()
	outside := t.TempDir()
	target := filepath.Join(outside, "file.bin")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	link := filepath.Join(base, "link.bin")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	doc := Document{
		Meta:     Meta{ID: "symlink", Version: "1", Owner: "oss"},
		Role:     Block{Body: "r"},
		Tasks:    []Block{{Body: "t"}},
		Images:   []Image{{Src: "link.bin"}},
		Elements: []Element{{Type: ElementMeta}, {Type: ElementRole}, {Type: ElementTask, Index: 0}, {Type: ElementImage, Index: 0}},
	}
	if _, err := Convert(doc, FormatDict, ConvertOptions{BaseDir: base}); err == nil {
		t.Fatalf("expected symlink escape to be rejected")
	}
}
