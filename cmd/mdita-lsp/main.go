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
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Println(version)
			os.Exit(0)
		case "--help", "-h":
			fmt.Fprintln(os.Stderr, "Usage: mdita-lsp [--version] [--help]")
			fmt.Fprintln(os.Stderr, "  Runs as an LSP server over stdio.")
			os.Exit(0)
		}
	}

	logFile, err := os.CreateTemp("", "mdita-lsp-*.log")
	if err == nil {
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	log.Printf("mdita-lsp %s starting", version)

	s := lsp.NewServer()
	ctx := context.Background()
	if err := s.Serve(ctx, os.Stdin, os.Stdout); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
