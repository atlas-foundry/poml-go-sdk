package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/atlas-foundry/poml-go-sdk/internal/mcp"
	"github.com/atlas-foundry/poml-go-sdk/poml"
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
	os.Exit(2)
}

func runMCP(args []string) {
	fs := flag.NewFlagSet("poml mcp", flag.ExitOnError)
	addr := fs.String("addr", ":7777", "address to listen on")
	file := fs.String("file", "", "path to POML file (required unless --stdin)")
	useStdin := fs.Bool("stdin", false, "read POML from stdin instead of file")
	fs.Parse(args)

	if *file == "" && !*useStdin {
		fmt.Fprintln(os.Stderr, "must provide --file or --stdin")
		os.Exit(2)
	}

	var body []byte
	var err error
	if *useStdin {
		body, err = os.ReadFile("/dev/stdin")
	} else {
		body, err = os.ReadFile(*file)
	}
	if err != nil {
		log.Fatalf("read POML: %v", err)
	}

	doc, err := poml.ParseString(string(body))
	if err != nil {
		log.Fatalf("parse POML: %v", err)
	}
	if err := doc.ValidateWithTrace(context.Background(), poml.TraceOptions{}); err != nil {
		log.Fatalf("validate POML: %v", err)
	}

	srv := mcp.New(doc)
	log.Printf("poml mcp serving on %s", *addr)
	if err := http.ListenAndServe(*addr, srv.Handler()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
