package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "The premium MCP CLI has moved to poml-horse. Refer to ~/the/poml-horse/incoming/cmd/poml/main.go for the previous implementation.")
	os.Exit(1)
}package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/atlas-foundry/poml-go-sdk/internal/mcp"
	"github.com/atlas-foundry/poml-go-sdk/poml"
)

var (
	listenAndServe = http.ListenAndServe
	readFile       = os.ReadFile
	exitFunc       = os.Exit
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}
	switch os.Args[1] {
	case "mcp":
		runMCP(os.Args[2:])
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "poml commands:\n")
	fmt.Fprintf(os.Stderr, "  poml mcp --file <path> [--addr :7777]\n")
	exitFunc(2)
}

func runMCP(args []string) {
	fs := flag.NewFlagSet("poml mcp", flag.ExitOnError)
	addr := fs.String("addr", ":7777", "address to listen on")
	file := fs.String("file", "", "path to POML file (required unless --stdin)")
	useStdin := fs.Bool("stdin", false, "read POML from stdin instead of file")
	traceStdout := fs.Bool("trace-stdout", false, "enable OTEL stdout tracing")
	traceOTLPHTTP := fs.String("trace-otlp-http", "", "OTLP/HTTP endpoint for tracing (e.g., localhost:4318)")
	traceOTLPGRPC := fs.String("trace-otlp-grpc", "", "OTLP/gRPC endpoint for tracing (e.g., localhost:4317)")
	traceInsecure := fs.Bool("trace-insecure", true, "allow insecure OTLP exporters")
	extendedStrict := fs.Bool("extended-strict", false, "enable strict POML Extended parsing/validation")
	extendedLenient := fs.Bool("extended", false, "enable lenient POML Extended parsing (no extra validation)")
	if err := fs.Parse(args); err != nil {
		log.Fatalf("parse flags: %v", err)
	}

	if *file == "" && !*useStdin {
		fmt.Fprintln(os.Stderr, "must provide --file or --stdin")
		os.Exit(2)
	}

	var body []byte
	var err error
	if *useStdin {
		body, err = readFile("/dev/stdin")
	} else {
		body, err = readFile(*file)
	}
	if err != nil {
		log.Fatalf("read POML: %v", err)
	}

	var traceOpts poml.TraceOptions
	switch {
	case *traceStdout:
		traceOpts.TracerProvider = mcp.StdoutTracerProvider()
	case *traceOTLPHTTP != "":
		traceOpts.TracerProvider = mcp.OTLPHTTPTracerProvider(*traceOTLPHTTP, *traceInsecure)
	case *traceOTLPGRPC != "":
		traceOpts.TracerProvider = mcp.OTLPGRPCTracerProvider(*traceOTLPGRPC, *traceInsecure)
	}

	parseOpts := poml.ParseOptions{PreserveWhitespace: true, Validate: false, Extended: poml.ExtendedOff}
	if *extendedStrict {
		parseOpts.Extended = poml.ExtendedStrict
		parseOpts.Validate = true
	} else if *extendedLenient {
		parseOpts.Extended = poml.ExtendedLenient
	}

	doc, err := poml.ParseReaderWithOptions(strings.NewReader(string(body)), parseOpts)
	if err != nil {
		log.Fatalf("parse POML: %v", err)
	}
	if parseOpts.Validate {
		if err := doc.ValidateWithTrace(context.Background(), traceOpts); err != nil {
			log.Fatalf("validate POML: %v", err)
		}
	}

	srv := mcp.New(doc, *file, traceOpts.TracerProvider)
	log.Printf("poml mcp serving on %s", *addr)
	if err := listenAndServe(*addr, srv.Handler()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
