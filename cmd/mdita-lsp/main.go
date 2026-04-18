package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aireilly/mdita-lsp/internal/lsp"
)

var version = "dev"

func main() {
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--version", "-v":
			fmt.Println("mdita-lsp " + version)
			os.Exit(0)
		case "--help", "-h":
			fmt.Fprintln(os.Stderr, "Usage: mdita-lsp [--version] [--help] [--stdio]")
			fmt.Fprintln(os.Stderr, "  Runs as an LSP server over stdio.")
			os.Exit(0)
		case "--stdio":
			// default mode, accepted for compatibility
		}
	}

	logFile, err := os.CreateTemp("", "mdita-lsp-*.log")
	if err == nil {
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	log.Printf("mdita-lsp %s starting", version)

	s := lsp.NewServer()
	s.SetVersion(version)
	ctx := context.Background()
	if err := s.Serve(ctx, os.Stdin, os.Stdout); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
