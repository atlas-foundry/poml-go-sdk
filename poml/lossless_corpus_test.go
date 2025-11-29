package poml

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Lossless corpus pass over examples/diagrams/goldens with lenient round-trip, optional strict, and conversions.
func TestLosslessCorpus(t *testing.T) {
	var total, skippedStrict, conversions int

	exampleFiles, err := filepath.Glob(filepath.Join("testdata", "examples", "*.poml"))
	if err != nil {
		t.Fatalf("glob examples: %v", err)
	}
	exampleFiles = append(exampleFiles, filepath.Join("testdata", "examples", "core_full.poml"))

	for _, path := range exampleFiles {
		path := path
		t.Run("example/"+filepath.Base(path), func(t *testing.T) {
			total++
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			lowerBody := strings.ToLower(string(body))
			isExtended := strings.Contains(lowerBody, `mode="extended"`) || strings.Contains(lowerBody, `extended="true"`)
			lenientOpts := ParseOptions{PreserveWhitespace: true, Validate: false, Extended: ExtendedLenient}
			doc, err := ParseReaderWithOptions(strings.NewReader(string(body)), lenientOpts)
			if err != nil {
				t.Fatalf("lenient parse: %v", err)
			}
			encOpts := EncodeOptions{IncludeHeader: false, PreserveOrder: true, PreserveWS: true}
			first, err := encodeToString(doc, encOpts)
			if err != nil {
				t.Fatalf("encode1: %v", err)
			}
			doc2, err := ParseReaderWithOptions(strings.NewReader(first), lenientOpts)
			if err != nil {
				t.Fatalf("parse2: %v", err)
			}
			second, err := encodeToString(doc2, encOpts)
			if err != nil {
				t.Fatalf("encode2: %v", err)
			}
			if first != second {
				t.Fatalf("lenient round-trip mismatch\n--- first ---\n%s\n--- second ---\n%s", first, second)
			}

			if !hasMetaRoleTask(doc2) {
				skippedStrict++
				return
			}

			if isExtended {
				skippedStrict++
				return
			}

			strictOpts := ParseOptions{PreserveWhitespace: true, Validate: true, Extended: ExtendedOff}
			if _, err := ParseReaderWithOptions(strings.NewReader(second), strictOpts); err != nil {
				t.Fatalf("strict parse failed: %v", err)
			}

			convFormats := []Format{FormatMessageDict, FormatDict, FormatOpenAIChat, FormatLangChain, FormatPydantic, FormatScene, FormatSceneJSON}
			opts := ConvertOptions{
				BaseDir:            filepath.Dir(path),
				MaxImageBytes:      1 << 20,
				MaxMediaBytes:      1 << 20,
				AllowAbsImagePaths: true,
				Extended:           ExtendedLenient,
			}
			for _, f := range convFormats {
				if _, err := Convert(doc2, f, opts); err != nil {
					t.Fatalf("convert %s: %v", f, err)
				}
				conversions++
			}
		})
	}

	t.Logf("Lossless corpus: %d examples checked, %d skipped strict/conversions (missing meta/role/task), %d conversions executed", total, skippedStrict, conversions)
}

// Diagram → scene determinism check against sorted goldens when available.
func TestLosslessDiagramsToSceneSorted(t *testing.T) {
	diagramFiles, err := filepath.Glob(filepath.Join("testdata", "diagrams", "*.poml"))
	if err != nil {
		t.Fatalf("glob diagrams: %v", err)
	}
	for _, path := range diagramFiles {
		path := path
		base := strings.TrimSuffix(filepath.Base(path), ".poml")
		sorted := filepath.Join(filepath.Dir(path), base+"_scene_sorted.json")
		// Only run when a sorted golden is present.
		if _, err := os.Stat(sorted); err != nil {
			continue
		}
		t.Run(base, func(t *testing.T) {
			doc, err := ParseFile(path)
			if err != nil {
				t.Fatalf("parse diagram: %v", err)
			}
			scenes, err := diagramsToScenes(doc.Diagrams, defaultSceneExportOptions)
			if err != nil {
				t.Fatalf("diagramsToScenes: %v", err)
			}
			if len(scenes) == 0 {
				t.Fatalf("no scenes produced")
			}
			got := normalizeScene(scenes[0])
			wantBytes, err := os.ReadFile(sorted)
			if err != nil {
				t.Fatalf("read sorted: %v", err)
			}
			var want Scene
			if err := json.Unmarshal(wantBytes, &want); err != nil {
				t.Fatalf("unmarshal sorted: %v", err)
			}
			want = normalizeScene(want)
			gotBytes, _ := json.Marshal(got)
			wantNorm, _ := json.Marshal(want)
			if string(gotBytes) != string(wantNorm) {
				t.Fatalf("diagram→scene mismatch\n--- got ---\n%s\n--- want ---\n%s", string(gotBytes), string(wantNorm))
			}
		})
	}
}

func hasMetaRoleTask(doc Document) bool {
	return strings.TrimSpace(doc.Meta.ID) != "" &&
		strings.TrimSpace(doc.Meta.Version) != "" &&
		strings.TrimSpace(doc.Meta.Owner) != "" &&
		strings.TrimSpace(doc.Role.Body) != "" &&
		len(doc.Tasks) > 0
}
