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
