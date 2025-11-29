package main

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
	traceSeed := fs.String("trace-seed", "", "optional deterministic trace seed (in-memory exporter)")
	extendedStrict := fs.Bool("extended-strict", false, "enable strict POML Extended parsing/validation")
	extendedLenient := fs.Bool("extended", false, "enable lenient POML Extended parsing (no extra validation)")
	extractTags := fs.Bool("extract-embedded-tags", false, "attempt to lift inline <tag> fragments in mixed text (experimental)")
	mimeAllow := fs.String("allowed-mime", "", "comma-separated MIME types to allow for figures/objects (extends defaults)")
	mimeAllowFile := fs.String("allowed-mime-file", "", "path to file with newline-separated MIME types to extend defaults")
	mimeAllowEnv := fs.String("allowed-mime-env", "POML_ALLOWED_MIME", "env var with comma-separated MIME types (extends defaults)")
	opKinds := fs.String("allowed-op-kinds", "", "comma-separated op kinds to allow (extends defaults: builtin,custom,tool,function)")
	opKindsEnv := fs.String("allowed-op-kinds-env", "POML_ALLOWED_OP_KINDS", "env var with comma-separated op kinds")
	configPOML := fs.String("config-poml", "", "optional POML file containing <object name=\"allowed-mime\">[...]</object> or <object name=\"allowed-op-kinds\">[...]</object>")
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
	var recorder *poml.TraceRecorder
	switch {
	case *traceStdout:
		traceOpts.TracerProvider = mcp.StdoutTracerProvider()
	case *traceOTLPHTTP != "":
		traceOpts.TracerProvider = mcp.OTLPHTTPTracerProvider(*traceOTLPHTTP, *traceInsecure)
	case *traceOTLPGRPC != "":
		traceOpts.TracerProvider = mcp.OTLPGRPCTracerProvider(*traceOTLPGRPC, *traceInsecure)
	case strings.TrimSpace(*traceSeed) != "":
		rc := poml.NewTraceRecorder(strings.TrimSpace(*traceSeed))
		recorder = &rc
		traceOpts.TracerProvider = recorder.Provider
	}

	parseOpts := poml.ParseOptions{PreserveWhitespace: true, Validate: false, Extended: poml.ExtendedOff}
	validateOpts := poml.ValidateOptions{Extended: poml.ExtendedOff}
	if *extendedStrict {
		parseOpts.Extended = poml.ExtendedStrict
		parseOpts.Validate = true
		validateOpts.Extended = poml.ExtendedStrict
	} else if *extendedLenient {
		parseOpts.Extended = poml.ExtendedLenient
		validateOpts.Extended = poml.ExtendedLenient
	}
	parseOpts.ExtractEmbeddedTags = *extractTags
	allow := poml.DefaultAllowedMIMEs()
	if strings.TrimSpace(*configPOML) != "" {
		cfg, err := poml.ParseFile(*configPOML)
		if err != nil {
			log.Fatalf("parse config poml: %v", err)
		}
		if m := extractListFromConfig(cfg, "allowed-mime"); len(m) > 0 {
			for _, t := range m {
				allow[strings.ToLower(strings.TrimSpace(t))] = struct{}{}
			}
		}
		if ks := extractListFromConfig(cfg, "allowed-op-kinds"); len(ks) > 0 {
			validateOpts.AllowedOpKinds = append(validateOpts.AllowedOpKinds, poml.AllowedOpKinds...)
			validateOpts.AllowedOpKinds = append(validateOpts.AllowedOpKinds, ks...)
		}
	}
	if strings.TrimSpace(*mimeAllow) != "" {
		for _, t := range strings.Split(*mimeAllow, ",") {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}
			allow[strings.ToLower(t)] = struct{}{}
		}
	}
	if strings.TrimSpace(*mimeAllowFile) != "" {
		raw, err := readFile(*mimeAllowFile)
		if err != nil {
			log.Fatalf("read allowed-mime-file: %v", err)
		}
		for _, line := range strings.Split(string(raw), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			allow[strings.ToLower(line)] = struct{}{}
		}
	}
	if env := strings.TrimSpace(os.Getenv(*mimeAllowEnv)); env != "" {
		for _, t := range strings.Split(env, ",") {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}
			allow[strings.ToLower(t)] = struct{}{}
		}
	}
	if len(allow) != len(poml.DefaultAllowedMIMEs()) {
		validateOpts.AllowedMIMETypes = allow
	}
	if strings.TrimSpace(*opKinds) != "" {
		var allowKinds []string
		allowKinds = append(allowKinds, poml.AllowedOpKinds...)
		for _, k := range strings.Split(*opKinds, ",") {
			k = strings.TrimSpace(k)
			if k != "" {
				allowKinds = append(allowKinds, k)
			}
		}
		validateOpts.AllowedOpKinds = allowKinds
	}
	if env := strings.TrimSpace(os.Getenv(*opKindsEnv)); env != "" {
		if len(validateOpts.AllowedOpKinds) == 0 {
			validateOpts.AllowedOpKinds = append(validateOpts.AllowedOpKinds, poml.AllowedOpKinds...)
		}
		for _, k := range strings.Split(env, ",") {
			k = strings.TrimSpace(k)
			if k != "" {
				validateOpts.AllowedOpKinds = append(validateOpts.AllowedOpKinds, k)
			}
		}
	}

	doc, err := poml.ParseReaderWithOptions(strings.NewReader(string(body)), parseOpts)
	if err != nil {
		log.Fatalf("parse POML: %v", err)
	}
	if parseOpts.Validate {
		if err := doc.ValidateWithTraceOptions(context.Background(), traceOpts, validateOpts); err != nil {
			log.Fatalf("validate POML: %v", err)
		}
	}

	srv := mcp.New(doc, *file, traceOpts.TracerProvider, recorder)
	log.Printf("poml mcp serving on %s", *addr)
	if err := listenAndServe(*addr, srv.Handler()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
