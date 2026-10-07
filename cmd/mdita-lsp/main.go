package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

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
			usage()
			os.Exit(0)
		case "--stdio":
			// default mode, accepted for compatibility
		}
	}

	closeLog := setupLogging()
	defer closeLog()

	log.Printf("mdita-lsp %s starting", version)

	s := lsp.NewServer()
	s.SetVersion(version)
	ctx := context.Background()
	err := s.Serve(ctx, os.Stdin, os.Stdout)

	switch {
	case err == nil:
		log.Printf("mdita-lsp exiting")
	case errors.Is(err, lsp.ErrExitWithoutShutdown()):
		// The spec asks for a non-zero status when `exit` arrives without a
		// preceding `shutdown`.
		log.Printf("exit without shutdown")
		closeLog()
		os.Exit(1)
	default:
		log.Printf("server error: %v", err)
		closeLog()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: mdita-lsp [--version] [--help] [--stdio]")
	fmt.Fprintln(os.Stderr, "  Runs as an LSP server over stdio.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Environment:")
	fmt.Fprintln(os.Stderr, "  MDITA_LSP_LOG   Log file path. Defaults to mdita-lsp.log in the")
	fmt.Fprintln(os.Stderr, "                  user cache directory, truncated on each launch.")
	fmt.Fprintln(os.Stderr, "                  Set it to \"-\" to log to stderr.")
}

// setupLogging opens the single log file this process writes to. The server
// used to create a fresh temp file on every launch, so an editor that restarts
// the server left a trail of logs nobody cleaned up.
func setupLogging() func() {
	path := os.Getenv("MDITA_LSP_LOG")
	if path == "-" {
		log.SetOutput(os.Stderr)
		return func() {}
	}
	if path == "" {
		dir, err := os.UserCacheDir()
		if err != nil {
			dir = os.TempDir()
		}
		dir = filepath.Join(dir, "mdita-lsp")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.SetOutput(io.Discard)
			return func() {}
		}
		path = filepath.Join(dir, "mdita-lsp.log")
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		log.SetOutput(io.Discard)
		return func() {}
	}
	log.SetOutput(f)
	return func() { _ = f.Close() }
}
