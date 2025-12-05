package main

import (
	"errors"
	"net/http"
	"os"
	"testing"
)

func TestUsageNoArgs(t *testing.T) {
	var exitCode int
	exitFunc = func(code int) { exitCode = code }
	t.Cleanup(func() { exitFunc = os.Exit })

	os.Args = []string{"poml"}
	main()
	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCode)
	}
}

func TestUsageUnknownCommand(t *testing.T) {
	var exitCode int
	exitFunc = func(code int) { exitCode = code }
	t.Cleanup(func() { exitFunc = os.Exit })

	os.Args = []string{"poml", "unknown"}
	main()
	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCode)
	}
}

func TestRunMCPWithStdin(t *testing.T) {
	exitFunc = func(int) {}
	t.Cleanup(func() { exitFunc = os.Exit })

	readFile = func(path string) ([]byte, error) {
		return []byte(`<poml><meta><id>x</id><version>0.1</version><owner>y</owner></meta><role>r</role><task>t</task></poml>`), nil
	}
	t.Cleanup(func() { readFile = os.ReadFile })

	var listened bool
	listenAndServe = func(addr string, handler http.Handler) error {
		listened = true
		_ = addr
		_ = handler
		return nil
	}
	t.Cleanup(func() { listenAndServe = http.ListenAndServe })

	runMCP([]string{"--stdin"})

	if !listened {
		t.Fatalf("listenAndServe not called")
	}
}

func TestRunMCPWithFile(t *testing.T) {
	exitFunc = func(int) {}
	t.Cleanup(func() { exitFunc = os.Exit })

	readFile = func(path string) ([]byte, error) {
		if path == "/test/file.poml" {
			return []byte(`<poml><meta><id>x</id><version>0.1</version><owner>y</owner></meta><role>r</role><task>t</task></poml>`), nil
		}
		return nil, errors.New("file not found")
	}
	t.Cleanup(func() { readFile = os.ReadFile })

	var listenedAddr string
	listenAndServe = func(addr string, handler http.Handler) error {
		listenedAddr = addr
		return nil
	}
	t.Cleanup(func() { listenAndServe = http.ListenAndServe })

	runMCP([]string{"--file", "/test/file.poml", "--addr", ":9999"})

	if listenedAddr != ":9999" {
		t.Fatalf("expected addr :9999, got %s", listenedAddr)
	}
}

func TestRunMCPWithExtendedStrict(t *testing.T) {
	exitFunc = func(int) {}
	t.Cleanup(func() { exitFunc = os.Exit })

	readFile = func(path string) ([]byte, error) {
		return []byte(`<poml mode="extended"><meta><id>x</id><version>0.1</version><owner>y</owner></meta><role>r</role><task>t</task></poml>`), nil
	}
	t.Cleanup(func() { readFile = os.ReadFile })

	var listened bool
	listenAndServe = func(addr string, handler http.Handler) error {
		listened = true
		return nil
	}
	t.Cleanup(func() { listenAndServe = http.ListenAndServe })

	runMCP([]string{"--stdin", "--extended-strict"})

	if !listened {
		t.Fatalf("listenAndServe not called")
	}
}

func TestRunMCPWithExtendedLenient(t *testing.T) {
	exitFunc = func(int) {}
	t.Cleanup(func() { exitFunc = os.Exit })

	readFile = func(path string) ([]byte, error) {
		return []byte(`<poml><meta><id>x</id><version>0.1</version><owner>y</owner></meta><role>r</role><task>t</task></poml>`), nil
	}
	t.Cleanup(func() { readFile = os.ReadFile })

	var listened bool
	listenAndServe = func(addr string, handler http.Handler) error {
		listened = true
		return nil
	}
	t.Cleanup(func() { listenAndServe = http.ListenAndServe })

	runMCP([]string{"--stdin", "--extended"})

	if !listened {
		t.Fatalf("listenAndServe not called")
	}
}

func TestRunMCPWithTraceStdout(t *testing.T) {
	exitFunc = func(int) {}
	t.Cleanup(func() { exitFunc = os.Exit })

	readFile = func(path string) ([]byte, error) {
		return []byte(`<poml><meta><id>x</id><version>0.1</version><owner>y</owner></meta><role>r</role><task>t</task></poml>`), nil
	}
	t.Cleanup(func() { readFile = os.ReadFile })

	var listened bool
	listenAndServe = func(addr string, handler http.Handler) error {
		listened = true
		return nil
	}
	t.Cleanup(func() { listenAndServe = http.ListenAndServe })

	runMCP([]string{"--stdin", "--trace-stdout"})

	if !listened {
		t.Fatalf("listenAndServe not called")
	}
}

func TestRunMCPWithTraceSeed(t *testing.T) {
	exitFunc = func(int) {}
	t.Cleanup(func() { exitFunc = os.Exit })

	readFile = func(path string) ([]byte, error) {
		return []byte(`<poml><meta><id>x</id><version>0.1</version><owner>y</owner></meta><role>r</role><task>t</task></poml>`), nil
	}
	t.Cleanup(func() { readFile = os.ReadFile })

	var listened bool
	listenAndServe = func(addr string, handler http.Handler) error {
		listened = true
		return nil
	}
	t.Cleanup(func() { listenAndServe = http.ListenAndServe })

	runMCP([]string{"--stdin", "--trace-seed", "test-seed"})

	if !listened {
		t.Fatalf("listenAndServe not called")
	}
}

func TestRunMCPWithMIMEFlags(t *testing.T) {
	exitFunc = func(int) {}
	t.Cleanup(func() { exitFunc = os.Exit })

	readFile = func(path string) ([]byte, error) {
		if path == "/test/mimes.txt" {
			return []byte("image/tiff\napplication/xml\n"), nil
		}
		return []byte(`<poml><meta><id>x</id><version>0.1</version><owner>y</owner></meta><role>r</role><task>t</task></poml>`), nil
	}
	t.Cleanup(func() { readFile = os.ReadFile })

	var listened bool
	listenAndServe = func(addr string, handler http.Handler) error {
		listened = true
		return nil
	}
	t.Cleanup(func() { listenAndServe = http.ListenAndServe })

	runMCP([]string{"--stdin", "--allowed-mime", "image/webp,video/mp4", "--allowed-mime-file", "/test/mimes.txt"})

	if !listened {
		t.Fatalf("listenAndServe not called")
	}
}

func TestRunMCPWithOpKindsFlag(t *testing.T) {
	exitFunc = func(int) {}
	t.Cleanup(func() { exitFunc = os.Exit })

	readFile = func(path string) ([]byte, error) {
		return []byte(`<poml><meta><id>x</id><version>0.1</version><owner>y</owner></meta><role>r</role><task>t</task></poml>`), nil
	}
	t.Cleanup(func() { readFile = os.ReadFile })

	var listened bool
	listenAndServe = func(addr string, handler http.Handler) error {
		listened = true
		return nil
	}
	t.Cleanup(func() { listenAndServe = http.ListenAndServe })

	runMCP([]string{"--stdin", "--allowed-op-kinds", "custom,special"})

	if !listened {
		t.Fatalf("listenAndServe not called")
	}
}

func TestRunMCPWithExtractEmbeddedTags(t *testing.T) {
	exitFunc = func(int) {}
	t.Cleanup(func() { exitFunc = os.Exit })

	readFile = func(path string) ([]byte, error) {
		return []byte(`<poml><meta><id>x</id><version>0.1</version><owner>y</owner></meta><role>r</role><task>t</task></poml>`), nil
	}
	t.Cleanup(func() { readFile = os.ReadFile })

	var listened bool
	listenAndServe = func(addr string, handler http.Handler) error {
		listened = true
		return nil
	}
	t.Cleanup(func() { listenAndServe = http.ListenAndServe })

	runMCP([]string{"--stdin", "--extract-embedded-tags"})

	if !listened {
		t.Fatalf("listenAndServe not called")
	}
}

func TestMainMCPCommand(t *testing.T) {
	exitFunc = func(int) {}
	t.Cleanup(func() { exitFunc = os.Exit })

	readFile = func(path string) ([]byte, error) {
		return []byte(`<poml><meta><id>x</id><version>0.1</version><owner>y</owner></meta><role>r</role><task>t</task></poml>`), nil
	}
	t.Cleanup(func() { readFile = os.ReadFile })

	var listened bool
	listenAndServe = func(addr string, handler http.Handler) error {
		listened = true
		return nil
	}
	t.Cleanup(func() { listenAndServe = http.ListenAndServe })

	os.Args = []string{"poml", "mcp", "--stdin"}
	main()

	if !listened {
		t.Fatalf("listenAndServe not called")
	}
}

func TestRunMCPWithMIMEEnv(t *testing.T) {
	exitFunc = func(int) {}
	t.Cleanup(func() { exitFunc = os.Exit })

	readFile = func(path string) ([]byte, error) {
		return []byte(`<poml><meta><id>x</id><version>0.1</version><owner>y</owner></meta><role>r</role><task>t</task></poml>`), nil
	}
	t.Cleanup(func() { readFile = os.ReadFile })

	var listened bool
	listenAndServe = func(addr string, handler http.Handler) error {
		listened = true
		return nil
	}
	t.Cleanup(func() { listenAndServe = http.ListenAndServe })

	// Set env variable for MIME types
	_ = os.Setenv("POML_ALLOWED_MIME", "image/custom,video/custom")
	t.Cleanup(func() { _ = os.Unsetenv("POML_ALLOWED_MIME") })

	runMCP([]string{"--stdin"})

	if !listened {
		t.Fatalf("listenAndServe not called")
	}
}

func TestRunMCPWithOpKindsEnv(t *testing.T) {
	exitFunc = func(int) {}
	t.Cleanup(func() { exitFunc = os.Exit })

	readFile = func(path string) ([]byte, error) {
		return []byte(`<poml><meta><id>x</id><version>0.1</version><owner>y</owner></meta><role>r</role><task>t</task></poml>`), nil
	}
	t.Cleanup(func() { readFile = os.ReadFile })

	var listened bool
	listenAndServe = func(addr string, handler http.Handler) error {
		listened = true
		return nil
	}
	t.Cleanup(func() { listenAndServe = http.ListenAndServe })

	// Set env variable for op kinds
	_ = os.Setenv("POML_ALLOWED_OP_KINDS", "custom-op,special-op")
	t.Cleanup(func() { _ = os.Unsetenv("POML_ALLOWED_OP_KINDS") })

	runMCP([]string{"--stdin"})

	if !listened {
		t.Fatalf("listenAndServe not called")
	}
}

func TestRunMCPWithConfigPOML(t *testing.T) {
	exitFunc = func(int) {}
	t.Cleanup(func() { exitFunc = os.Exit })

	// Create a real temp file for config
	tmpFile, err := os.CreateTemp("", "config-*.poml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	configContent := `<poml><meta><id>cfg</id><version>1</version><owner>o</owner></meta><object name="allowed-mime">["image/custom","video/custom"]</object><object name="allowed-op-kinds">["custom-kind"]</object></poml>`
	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	_ = tmpFile.Close()

	readFile = func(path string) ([]byte, error) {
		return []byte(`<poml><meta><id>x</id><version>0.1</version><owner>y</owner></meta><role>r</role><task>t</task></poml>`), nil
	}
	t.Cleanup(func() { readFile = os.ReadFile })

	var listened bool
	listenAndServe = func(addr string, handler http.Handler) error {
		listened = true
		return nil
	}
	t.Cleanup(func() { listenAndServe = http.ListenAndServe })

	runMCP([]string{"--stdin", "--config-poml", tmpFile.Name()})

	if !listened {
		t.Fatalf("listenAndServe not called")
	}
}

func TestRunMCPWithTraceOTLPHTTP(t *testing.T) {
	exitFunc = func(int) {}
	t.Cleanup(func() { exitFunc = os.Exit })

	readFile = func(path string) ([]byte, error) {
		return []byte(`<poml><meta><id>x</id><version>0.1</version><owner>y</owner></meta><role>r</role><task>t</task></poml>`), nil
	}
	t.Cleanup(func() { readFile = os.ReadFile })

	var listened bool
	listenAndServe = func(addr string, handler http.Handler) error {
		listened = true
		return nil
	}
	t.Cleanup(func() { listenAndServe = http.ListenAndServe })

	runMCP([]string{"--stdin", "--trace-otlp-http", "localhost:4318"})

	if !listened {
		t.Fatalf("listenAndServe not called")
	}
}

func TestRunMCPWithTraceOTLPGRPC(t *testing.T) {
	exitFunc = func(int) {}
	t.Cleanup(func() { exitFunc = os.Exit })

	readFile = func(path string) ([]byte, error) {
		return []byte(`<poml><meta><id>x</id><version>0.1</version><owner>y</owner></meta><role>r</role><task>t</task></poml>`), nil
	}
	t.Cleanup(func() { readFile = os.ReadFile })

	var listened bool
	listenAndServe = func(addr string, handler http.Handler) error {
		listened = true
		return nil
	}
	t.Cleanup(func() { listenAndServe = http.ListenAndServe })

	runMCP([]string{"--stdin", "--trace-otlp-grpc", "localhost:4317"})

	if !listened {
		t.Fatalf("listenAndServe not called")
	}
}
