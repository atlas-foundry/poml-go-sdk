//go:build ignore

// go_bridge is a utility to compare Go SDK outputs with Python SDK outputs via the py_bridge.
// Run with: go run tools/go_bridge.go --format openai_chat --file poml/testdata/examples/101_explain_character.poml
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	sdk "github.com/atlas-foundry/poml-go-sdk/poml"
)

func main() {
	format := flag.String("format", "openai_chat", "message_dict|dict|openai_chat|langchain")
	file := flag.String("file", "", "POML file to parse")
	pyPath := flag.String("py", "python3", "python executable")
	flag.Parse()
	if *file == "" {
		fmt.Println("missing --file")
		os.Exit(1)
	}
	body, err := os.ReadFile(*file)
	if err != nil {
		panic(err)
	}
	doc, err := sdk.ParseString(string(body))
	if err != nil {
		panic(err)
	}
	goOut, err := sdk.Convert(doc, sdk.Format(*format), sdk.ConvertOptions{})
	if err != nil {
		panic(err)
	}
	goNorm := normalize(goOut)

	pyCmd := exec.Command(*pyPath, "tools/py_bridge.py", "--format", *format, "--file", *file)
	pyCmd.Dir = repoRoot()
	pyCmd.Stderr = os.Stderr
	var pyStdout bytes.Buffer
	pyCmd.Stdout = &pyStdout
	if err := pyCmd.Run(); err != nil {
		panic(err)
	}
	var pyAny any
	if err := json.Unmarshal(pyStdout.Bytes(), &pyAny); err != nil {
		panic(err)
	}
	pyNorm := normalize(pyAny)

	if diff := diffJSON(goNorm, pyNorm); diff != "" {
		fmt.Println("DIFF:\n", diff)
		os.Exit(1)
	}
	fmt.Println("OK: outputs match after normalization")
}

func repoRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return dir
		}
		dir = parent
	}
}

// normalize recursively sorts maps/slices and coerces numbers/strings for lenient diffing.
func normalize(v any) any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case map[string]any:
		normalized := make(map[string]any, len(val))
		for k, v2 := range val {
			normalized[k] = normalize(v2)
		}
		return normalized
	case []any:
		out := make([]any, 0, len(val))
		for _, v2 := range val {
			out = append(out, normalize(v2))
		}
		// attempt to sort by JSON string for deterministic order
		sort.Slice(out, func(i, j int) bool {
			ai, _ := json.Marshal(out[i])
			aj, _ := json.Marshal(out[j])
			return string(ai) < string(aj)
		})
		return out
	case string:
		return strings.TrimSpace(val)
	case float64, float32, int, int64, int32:
		return val
	default:
		// try to convert map[string]interface{} for pydantic
		b, err := json.Marshal(val)
		if err == nil {
			var m any
			if json.Unmarshal(b, &m) == nil {
				return normalize(m)
			}
		}
		return val
	}
}

func diffJSON(a, b any) string {
	aj, _ := json.MarshalIndent(a, "", "  ")
	bj, _ := json.MarshalIndent(b, "", "  ")
	if bytes.Equal(aj, bj) {
		return ""
	}
	return fmt.Sprintf("go:\n%s\npython:\n%s\n", aj, bj)
}
