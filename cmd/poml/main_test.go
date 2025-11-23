package main

import (
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
